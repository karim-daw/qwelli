package main

import (
	"fmt"
	"log"

	"github.com/joho/godotenv"
	"github.com/karim-daw/qwelli/internal/indexer"
)

func main() {
	// Load .env file if it exists (ignore error if not found)
	_ = godotenv.Load()

	// Initialize embedder with OpenAI API
	fmt.Println("🤖 Initializing embedder...")
	embedder, err := indexer.NewEmbedder("text-embedding-3-small", "", 0)
	if err != nil {
		log.Fatalf("Failed to initialize embedder: %v", err)
	}

	fmt.Printf("✓ Embedder initialized (dimension: %d)\n\n", embedder.Dimension())

	// Generate embeddings for a batch of strings
	texts := []string{
		"Hello world",
		"This is a test",
		"Embeddings are useful",
		"Cloud APIs are fast and reliable",
	}

	fmt.Printf("Generating embeddings for %d texts...\n\n", len(texts))

	// Single embedding test
	fmt.Println("--- Single Embedding Test ---")
	embedding, err := embedder.Embed(texts[0])
	if err != nil {
		log.Fatalf("Failed to generate embedding: %v", err)
	}
	fmt.Printf("Text: %q\n", texts[0])
	fmt.Printf("Embedding dimension: %d\n", len(embedding))
	previewLen := min(len(embedding), 5)
	fmt.Printf("First %d values: %v\n\n", previewLen, embedding[:previewLen])

	// Batch embedding test
	fmt.Println("--- Batch Embedding Test ---")
	batchEmbeddings, err := embedder.EmbedBatch(texts)
	if err != nil {
		log.Fatalf("Failed to generate batch embeddings: %v", err)
	}

	for i, emb := range batchEmbeddings {
		fmt.Printf("\nText %d: %q\n", i+1, texts[i])
		fmt.Printf("Embedding dimension: %d\n", len(emb))
		previewLen := min(len(emb), 5)
		fmt.Printf("First %d values: %v\n", previewLen, emb[:previewLen])
	}

	// Optionally, print full result as JSON
	// result := map[string]any{
	// 	"model":      "nomic-embed-text",
	// 	"dimension":  embedder.Dimension(),
	// 	"embeddings": batchEmbeddings,
	// }
	// resultJSON, err := json.MarshalIndent(result, "", " ")
	// if err != nil {
	// 	log.Printf("Failed to marshal JSON: %v", err)
	// } else {
	// 	fmt.Printf("\n\n--- Full Result (JSON) ---\n%s\n", string(resultJSON))
	// }

	fmt.Println("\n✅ Test completed successfully!")
}

// to run: go run tests/test_embedding.go
