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
	_ = godotenv.Load(".env")
	totalStart := time.Now()

	testFolder := "tests/demo/testdata"
	dbPath := "demo.db"
	modelName := os.Getenv("QWELLI_EMBEDDING_MODEL") // text-embedding-3-small
	apiKey := os.Getenv("QWELLI_EMBEDDING_KEY")
	endpoint := os.Getenv("QWELLI_EMBEDDING_ENDPOINT") // https://api.openai.com/v1/embeddings

	// Clean up existing database
	os.Remove(dbPath)

	// Initialize embedder
	fmt.Println("🤖 Initializing embedder...")
	embedder, err := indexer.NewEmbedder(apiKey, modelName, endpoint)
	if err != nil {
		log.Fatalf("Failed to initialize embedder: %v", err)
	}

	// Get dimension
	testEmbed, err := embedder.Embed("test")
	if err != nil {
		log.Fatalf("Failed to get embedding dimension: %v", err)
	}
	dimension := len(testEmbed)
	fmt.Printf("  ✓ Model: %s, Dimension: %d\n", modelName, dimension)

	// Open database
	projectDB, err := db.OpenProjectDB(dbPath, dimension)
	if err != nil {
		log.Fatalf("Failed to open database: %v", err)
	}
	defer projectDB.Close()

	// Scan folder
	fmt.Printf("\n📁 Scanning: %s\n", testFolder)
	files, err := scanFolder(testFolder)
	if err != nil {
		log.Fatalf("Failed to scan folder: %v", err)
	}
	fmt.Printf("  Found %d file(s)\n", len(files))

	// Process files
	fmt.Println("\n📄 Indexing files...")
	var docs []db.Document
	var contents []string

	for _, path := range files {
		content, err := os.ReadFile(path)
		if err != nil {
			continue
		}
		info, _ := os.Stat(path)

		metadata, _ := json.Marshal(map[string]string{
			"indexed_at": time.Now().Format(time.RFC3339),
			"file_name":  filepath.Base(path),
		})

		docs = append(docs, db.Document{
			ID:           generateDocID(path),
			Path:         path,
			FileType:     getFileType(path),
			ModifiedAt:   info.ModTime(),
			Size:         info.Size(),
			TextMetadata: metadata,
			Content:      string(content),
		})
		contents = append(contents, string(content))
	}

	// Insert documents
	for _, doc := range docs {
		projectDB.InsertDocument(doc)
	}

	// Generate embeddings
	embeddings, err := embedder.EmbedBatch(contents)
	if err != nil {
		log.Fatalf("Failed to generate embeddings: %v", err)
	}

	// Insert embeddings
	for i, vec := range embeddings {
		projectDB.InsertEmbedding(db.Embedding{DocID: docs[i].ID, Vector: vec})
		fmt.Printf("  ✓ %s\n", filepath.Base(docs[i].Path))
	}

	// Build index
	fmt.Println("\n🔍 Building HNSW index...")
	if err := projectDB.BuildHNSWIndex(); err != nil {
		log.Fatalf("Failed to build index: %v", err)
	}

	// Test searches
	fmt.Println("\n🔎 Searching...")
	queries := []string{
		"hello my name",
		"neural networks",
		"cookie recipe",
		"team meeting",
	}

	for _, q := range queries {
		vec, _ := embedder.Embed(q)
		results, _ := projectDB.SearchANN(vec, 3)
		fmt.Printf("\n  \"%s\":\n", q)
		for i, r := range results {
			doc, _ := projectDB.GetDocument(r.DocID)
			fmt.Printf("    %d. %s (%.4f)\n", i+1, filepath.Base(doc.Path), r.Distance)
		}
	}

	fmt.Printf("\n✅ Demo completed in %v\n", time.Since(totalStart))
}

func scanFolder(root string) ([]string, error) {
	var files []string
	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err == nil && !d.IsDir() {
			files = append(files, path)
		}
		return nil
	})
	return files, err
}

func generateDocID(path string) string {
	hash := md5.Sum([]byte(path))
	return hex.EncodeToString(hash[:])
}

func getFileType(path string) string {
	ext := strings.ToLower(filepath.Ext(path))
	if ext == "" {
		return "unknown"
	}
	return ext[1:]
}

// run
// go run tests/demo/main.go
