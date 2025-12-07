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

	"github.com/joho/godotenv"
	"github.com/karim-daw/qwelli/internal/db"
	"github.com/karim-daw/qwelli/internal/indexer"
)

func main() {
	// Load .env file if it exists (ignore error if not found)
	_ = godotenv.Load(".env")

	totalStart := time.Now()

	// Configuration
	testFolder := "tests/demo/testdata"
	dbPath := "demo.db"
	modelName := os.Getenv("QWELLI_EMBEDDING_MODEL")
	apiKey := os.Getenv("QWELLI_EMBEDDING_KEY")
	endpoint := os.Getenv("QWELLI_EMBEDDING_ENDPOINT")

	// Clean up existing database
	if _, err := os.Stat(dbPath); err == nil {
		fmt.Println("🗑️  Deleting existing demo.db")
		os.Remove(dbPath)
	}

	// Initialize embedder first to get dimension
	fmt.Println("🤖 Initializing embedder...")
	embedderStart := time.Now()

	embedder, err := indexer.NewEmbedder(apiKey, modelName, endpoint)
	if err != nil {
		log.Fatalf("Failed to initialize embedder: %v", err)
	}
	fmt.Printf("  ✓ Embedder initialized (model: %s)\n", modelName)
	fmt.Printf("  ⏱️  Embedder initialization: %v\n", time.Since(embedderStart))

	// Get embedding dimension by generating a test embedding
	fmt.Println("\n🔍 Detecting embedding dimension...")
	dimStart := time.Now()
	testEmbed, err := embedder.Embed("test")
	if err != nil {
		log.Fatalf("Failed to get embedding dimension: %v", err)
	}
	dimension := len(testEmbed)
	fmt.Printf("  ✓ Embedding dimension: %d\n", dimension)
	fmt.Printf("  ⏱️  Dimension detection: %v\n", time.Since(dimStart))

	// Open the database
	fmt.Println("\n📂 Opening database...")
	dbStart := time.Now()
	projectDB, err := db.OpenProjectDB(dbPath, dimension)
	if err != nil {
		log.Fatalf("Failed to open database: %v", err)
	}
	defer projectDB.Close()

	fmt.Println("✓ Database opened successfully")
	fmt.Printf("  ⏱️  Database initialization: %v\n", time.Since(dbStart))

	// Scan test folder and index files
	fmt.Printf("\n📁 Scanning folder: %s\n", testFolder)
	scanStart := time.Now()
	files, err := scanFolder(testFolder)
	if err != nil {
		log.Fatalf("Failed to scan folder: %v", err)
	}

	if len(files) == 0 {
		log.Fatalf("No files found in %s", testFolder)
	}

	fmt.Printf("  Found %d file(s)\n", len(files))
	fmt.Printf("  ⏱️  Folder scan: %v\n", time.Since(scanStart))

	// Process and index files
	fmt.Println("\n📄 Indexing files...")
	indexStart := time.Now()

	// First, prepare all documents
	var docs []db.Document
	var fileContents []string
	var fileNames []string

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
			Metadata:   metadataJSON, // Store as []byte for JSON column
			Content:    string(content),
		}

		docs = append(docs, doc)
		fileContents = append(fileContents, string(content))
		fileNames = append(fileNames, filepath.Base(file.Path))
	}

	// Insert all documents
	fmt.Printf("  Inserting %d documents...\n", len(docs))
	for _, doc := range docs {
		if err := projectDB.InsertDocument(doc); err != nil {
			log.Printf("  ⚠️  Failed to insert document %s: %v", doc.ID, err)
		}
	}

	// Generate all embeddings in batch
	fmt.Printf("  Generating embeddings for %d documents...\n", len(fileContents))
	embeddings, err := embedder.EmbedBatch(fileContents)
	if err != nil {
		log.Fatalf("  ⚠️  Failed to generate batch embeddings: %v", err)
	}

	// Get or create model
	modelID, err := projectDB.GetOrCreateModel(modelName, dimension)
	if err != nil {
		log.Fatalf("  ⚠️  Failed to get model: %v", err)
	}

	// Insert all embeddings
	fmt.Printf("  Inserting %d embeddings...\n", len(embeddings))
	for i, embedding := range embeddings {
		emb := db.Embedding{
			DocID:   docs[i].ID,
			ModelID: modelID,
			Vector:  embedding,
		}

		if err := projectDB.InsertEmbedding(emb); err != nil {
			log.Printf("  ⚠️  Failed to insert embedding for %s: %v", docs[i].ID, err)
			continue
		}

		fmt.Printf("  ✓ Indexed: %s (%d bytes)\n", fileNames[i], docs[i].Size)
	}

	docCount := len(docs)

	fmt.Printf("\n✓ Successfully indexed %d document(s)\n", docCount)
	fmt.Printf("  ⏱️  Indexing took: %v (avg: %v per document)\n",
		time.Since(indexStart), time.Since(indexStart)/time.Duration(docCount))

	// Build HNSW index for fast similarity search
	fmt.Println("\n🔍 Building HNSW index...")
	hnswStart := time.Now()
	if err := projectDB.BuildHNSWIndex(); err != nil {
		log.Fatalf("Failed to build index: %v", err)
	}
	fmt.Println("  ✓ Index built successfully")
	fmt.Printf("  ⏱️  HNSW index build: %v\n", time.Since(hnswStart))

	// Perform similarity searches
	fmt.Println("\n🔎 Performing similarity searches...")
	searchStart := time.Now()

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
		queryStart := time.Now()
		fmt.Printf("\n  Search %d: \"%s\" (%s)\n", i+1, test.query, test.name)
		queryVector, err := embedder.Embed(test.query)
		if err != nil {
			log.Printf("  ⚠️  Failed to embed query: %v", err)
			continue
		}

		results, err := projectDB.SearchANN(queryVector, test.topK, modelID)
		if err != nil {
			log.Printf("  ⚠️  Search failed: %v", err)
			continue
		}

		printResults(results, projectDB)
		fmt.Printf("    ⏱️  Query time: %v\n", time.Since(queryStart))
	}

	fmt.Printf("\n  ⏱️  Total search time: %v (avg: %v per query)\n",
		time.Since(searchStart), time.Since(searchStart)/time.Duration(len(testQueries)))

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
	fmt.Printf("\n⏱️  TOTAL TIME: %v\n", time.Since(totalStart))
	fmt.Println("\n📊 Performance Breakdown:")
	fmt.Printf("   • Embedder initialization: included in total\n")
	fmt.Printf("   • Database setup: included in total\n")
	fmt.Printf("   • File indexing: included in total\n")
	fmt.Printf("   • HNSW index build: included in total\n")
	fmt.Printf("   • Searches (10 queries): included in total\n")
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

// truncate shortens a string to the specified length
func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}
