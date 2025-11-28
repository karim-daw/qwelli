package main

import (
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io/fs"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/karim-daw/qwelli/internal/db"
	"github.com/karim-daw/qwelli/internal/indexer"
)

func main() {
	// Configuration
	// Try multiple possible paths for test folder
	testFolder := findTestFolder()
	dbPath := "demo.db"
	vectorDim := 384

	// Clean up existing database
	if _, err := os.Stat(dbPath); err == nil {
		fmt.Println("🗑️  Deleting existing demo.db")
		os.Remove(dbPath)
	}

	// Open the database
	fmt.Println("📂 Opening database...")
	projectDB, err := db.OpenProjectDB(dbPath, vectorDim)
	if err != nil {
		log.Fatalf("Failed to open database: %v", err)
	}
	defer projectDB.Close()

	fmt.Println("✓ Database opened successfully")
	fmt.Printf("  Vector dimension: %d\n", vectorDim)

	// Initialize embedder
	fmt.Println("\n🤖 Initializing embedder...")
	modelDir := indexer.GetModelPath("")
	embedder, err := indexer.NewEmbedder("KnightsAnalytics/all-MiniLM-L6-v2", modelDir, vectorDim)
	if err != nil {
		log.Fatalf("Failed to initialize embedder: %v", err)
	}
	defer embedder.Close()
	fmt.Println("  ✓ Embedder initialized")

	// Scan test folder and index files
	fmt.Printf("\n📁 Scanning folder: %s\n", testFolder)
	files, err := scanFolder(testFolder)
	if err != nil {
		log.Fatalf("Failed to scan folder: %v", err)
	}

	if len(files) == 0 {
		log.Fatalf("No files found in %s", testFolder)
	}

	fmt.Printf("  Found %d file(s)\n", len(files))

	// Process and index files
	fmt.Println("\n📄 Indexing files...")
	docCount := 0
	for _, file := range files {
		// Read file content
		content, err := os.ReadFile(file.Path)
		if err != nil {
			log.Printf("  ⚠️  Failed to read %s: %v", file.Path, err)
			continue
		}

		// Get file info
		info, err := os.Stat(file.Path)
		if err != nil {
			log.Printf("  ⚠️  Failed to stat %s: %v", file.Path, err)
			continue
		}

		// Generate document ID from file path (using MD5 hash)
		docID := generateDocID(file.Path)

		// Determine file type
		fileType := getFileType(file.Path)

		// Create metadata
		metadata := map[string]interface{}{
			"indexed_at": time.Now().Format(time.RFC3339),
			"file_name":  filepath.Base(file.Path),
		}
		metadataJSON, _ := json.Marshal(metadata)

		// Create document
		doc := db.Document{
			ID:         docID,
			Path:       file.Path,
			FileType:   fileType,
			ModifiedAt: info.ModTime(),
			Size:       info.Size(),
			Metadata:   string(metadataJSON),
			Content:    string(content),
		}

		// Insert document
		if err := projectDB.InsertDocument(doc); err != nil {
			log.Printf("  ⚠️  Failed to insert document %s: %v", docID, err)
			continue
		}

		// Generate embedding from content
		embedding, err := embedder.Embed(string(content))
		if err != nil {
			log.Printf("  ⚠️  Failed to generate embedding for %s: %v", docID, err)
			continue
		}

		// Insert embedding
		emb := db.Embedding{
			DocID:  docID,
			Vector: embedding,
		}

		if err := projectDB.InsertEmbedding(emb); err != nil {
			log.Printf("  ⚠️  Failed to insert embedding for %s: %v", docID, err)
			continue
		}

		fmt.Printf("  ✓ Indexed: %s (%d bytes)\n", filepath.Base(file.Path), info.Size())
		docCount++
	}

	fmt.Printf("\n✓ Successfully indexed %d document(s)\n", docCount)

	// Build HNSW index for fast similarity search
	fmt.Println("\n🔍 Building HNSW index...")
	if err := projectDB.BuildHNSWIndex(); err != nil {
		log.Fatalf("Failed to build index: %v", err)
	}
	fmt.Println("  ✓ Index built successfully")

	// Perform similarity searches
	fmt.Println("\n🔎 Performing similarity searches...")

	// Test queries covering different topics
	testQueries := []struct {
		name  string
		query string
		topK  int
	}{
		{"Greeting", "hello my name", 3},
		{"Farewell", "bye farewell goodbye", 3},
		{"Machine Learning", "neural networks and deep learning algorithms", 3},
		{"Cooking", "chocolate chip cookie recipe ingredients", 3},
		{"Travel", "Paris attractions and tourist destinations", 3},
		{"Programming", "Python code functions and recursion", 3},
		{"Database", "vector database architecture and HNSW indexing", 3},
		{"Business", "team meeting notes and action items", 3},
		{"API", "REST API endpoints and documentation", 3},
		{"Literature", "poetry and poems about roads", 3},
	}

	for i, test := range testQueries {
		fmt.Printf("\n  Search %d: \"%s\" (%s)\n", i+1, test.query, test.name)
		queryVector, err := embedder.Embed(test.query)
		if err != nil {
			log.Printf("  ⚠️  Failed to embed query: %v", err)
			continue
		}

		results, err := projectDB.SearchANN(queryVector, test.topK)
		if err != nil {
			log.Printf("  ⚠️  Search failed: %v", err)
			continue
		}

		printResults(results, projectDB)
	}

	// Show all indexed documents
	fmt.Println("\n📊 Summary:")
	allEmbeddings, err := projectDB.LoadAllEmbeddings()
	if err != nil {
		log.Printf("  ⚠️  Failed to load embeddings: %v", err)
	} else {
		fmt.Printf("  ✓ Total embeddings in database: %d\n", len(allEmbeddings))
	}

	fmt.Println("\n✅ End-to-end demo completed successfully!")
	fmt.Printf("   Database saved to: %s\n", dbPath)
}

// scanFolder recursively scans a folder and returns all files
func scanFolder(root string) ([]struct{ Path string }, error) {
	var files []struct{ Path string }

	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if !d.IsDir() {
			files = append(files, struct{ Path string }{Path: path})
		}

		return nil
	})

	return files, err
}

// generateDocID creates a unique document ID from file path
func generateDocID(path string) string {
	hash := md5.Sum([]byte(path))
	return hex.EncodeToString(hash[:])
}

// getFileType determines file type from extension
func getFileType(path string) string {
	ext := strings.ToLower(filepath.Ext(path))
	if ext == "" {
		return "unknown"
	}
	// Remove the dot
	return ext[1:]
}

// printResults prints search results
func printResults(results []db.SearchResult, projectDB *db.ProjectDB) {
	if len(results) == 0 {
		fmt.Println("    No results found")
		return
	}

	for i, result := range results {
		doc, err := projectDB.GetDocument(result.DocID)
		if err != nil {
			fmt.Printf("    Result %d: [Error loading document: %v]\n", i+1, err)
			continue
		}

		fmt.Printf("    Result %d:\n", i+1)
		fmt.Printf("      File: %s\n", filepath.Base(doc.Path))
		fmt.Printf("      Path: %s\n", doc.Path)
		fmt.Printf("      Distance: %.4f\n", result.Distance)
		fmt.Printf("      Content: %s\n", truncate(strings.TrimSpace(doc.Content), 80))
	}
}

// findTestFolder finds the test folder by trying multiple possible paths
func findTestFolder() string {
	// Try paths relative to different execution contexts
	possiblePaths := []string{
		"tests/test_folder",       // From project root
		"../tests/test_folder",    // From examples/demo
		"../../tests/test_folder", // From examples/demo/ subdirectory
	}

	for _, path := range possiblePaths {
		if _, err := os.Stat(path); err == nil {
			return path
		}
	}

	// Default fallback
	return "tests/test_folder"
}

// truncate shortens a string to the specified length
func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}
