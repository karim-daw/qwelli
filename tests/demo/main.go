package main

import (
	"context"
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"io/fs"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/joho/godotenv"
	"github.com/karim-daw/qwelli/internal/db"
	"github.com/karim-daw/qwelli/internal/engine/embeddings"
	"github.com/karim-daw/qwelli/internal/engine/extraction"
	"github.com/karim-daw/qwelli/internal/voyage"
)

//// asoidjaoisjd

func main() {
	_ = godotenv.Load(".env")
	totalStart := time.Now()

	testFolder := "tests/demo/testdata"
	dbPath := "demo.db"
	modelName := os.Getenv("QWELLI_EMBEDDING_MODEL") // voyage-multimodal-3
	apiKey := os.Getenv("QWELLI_EMBEDDING_KEY")
	endpoint := os.Getenv("QWELLI_EMBEDDING_ENDPOINT") // https://api.voyageai.com/v1/multimodalembeddings

	// Clean up existing database
	os.Remove(dbPath)

	// Initialize voyage client and embedder
	fmt.Println("🤖 Initializing embedder...")
	voyageClient, err := voyage.NewClient(voyage.ClientConfig{
		APIKey:            apiKey,
		EmbeddingModel:    modelName,
		EmbeddingEndpoint: endpoint,
	})
	if err != nil {
		log.Fatalf("Failed to create voyage client: %v", err)
	}

	embedder, err := embeddings.NewEmbedder(voyageClient)
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
	var filesList []db.File
	var chunksList []db.Chunk
	var contents []string

	for _, path := range files {
		// Normalize path to absolute
		absPath, err := filepath.Abs(path)
		if err != nil {
			continue
		}

		content, err := os.ReadFile(absPath)
		if err != nil {
			continue
		}
		info, err := os.Stat(absPath)
		if err != nil {
			continue
		}

		// Compute file hash
		fileHash, err := extraction.ComputeSHA256(absPath)
		if err != nil {
			log.Printf("⚠️  Failed to compute hash for %s: %v", absPath, err)
			continue
		}

		// Create file record
		fileID := generateFileID(absPath)
		file := db.File{
			FileID:     fileID,
			Path:       absPath,
			FileType:   getFileType(absPath),
			FileHash:   fileHash,
			ModifiedAt: info.ModTime(),
			Size:       info.Size(),
			IndexedAt:  time.Now(),
		}
		filesList = append(filesList, file)

		// Create a single chunk for the entire file content
		chunkID := generateChunkID(fileID, 0)
		chunk := db.Chunk{
			ChunkID:     chunkID,
			FileID:      fileID,
			FilePath:    absPath,
			FileType:    getFileType(absPath),
			ChunkIndex:  0,
			TotalChunks: 1,
			Content:     string(content),
		}
		chunksList = append(chunksList, chunk)
		contents = append(contents, string(content))
	}

	ctx := context.Background()

	// Insert files
	for _, file := range filesList {
		if err := projectDB.InsertFile(ctx, file); err != nil {
			log.Printf("⚠️  Failed to insert file %s: %v", file.Path, err)
			continue
		}
	}

	// Insert chunks
	for i, chunk := range chunksList {
		chunkID, err := projectDB.InsertChunk(ctx, chunk)
		if err != nil {
			log.Printf("⚠️  Failed to insert chunk for %s: %v", chunk.FilePath, err)
			continue
		}
		// Update chunk ID for embedding insertion
		chunksList[i].ChunkID = chunkID
	}

	// Generate embeddings
	embeddings, err := embedder.EmbedBatch(context.Background(), contents, nil)
	if err != nil {
		log.Fatalf("Failed to generate embeddings: %v", err)
	}

	// Insert embeddings
	for i, vec := range embeddings {
		if err := projectDB.InsertEmbedding(ctx, chunksList[i].ChunkID, vec); err != nil {
			log.Printf("⚠️  Failed to insert embedding for %s: %v", chunksList[i].FilePath, err)
			continue
		}
		fmt.Printf("  ✓ %s\n", filepath.Base(chunksList[i].FilePath))
	}

	// Build index
	fmt.Println("\n🔍 Building HNSW index...")
	if err := projectDB.BuildHNSWIndex(ctx); err != nil {
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
		results, _ := projectDB.SearchANN(ctx, vec, 3)
		fmt.Printf("\n  \"%s\":\n", q)
		for i, r := range results {
			fmt.Printf("    %d. %s (%.4f)\n", i+1, filepath.Base(r.FilePath), r.Distance)
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

func generateFileID(path string) string {
	hash := md5.Sum([]byte(path))
	return hex.EncodeToString(hash[:])
}

func generateChunkID(fileID string, chunkIndex int) string {
	source := fmt.Sprintf("%s:chunk:%d", fileID, chunkIndex)
	hash := md5.Sum([]byte(source))
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
