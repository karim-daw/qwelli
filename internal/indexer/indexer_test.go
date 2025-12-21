package indexer

import (
	"os"
	"testing"

	"github.com/joho/godotenv"
)

func TestNewEmbedder_Validation(t *testing.T) {
	t.Run("missing_api_key", func(t *testing.T) {
		_, err := NewEmbedder("voyage", "", "model", "endpoint")
		if err == nil {
			t.Fatal("NewEmbedder() expected error for missing API key")
		}
	})

	t.Run("valid_config", func(t *testing.T) {
		embedder, err := NewEmbedder("voyage", "test-key", "test-model", "http://localhost/v1/embeddings")
		if err != nil {
			t.Fatalf("NewEmbedder() error = %v", err)
		}
		if embedder == nil {
			t.Fatal("NewEmbedder() returned nil")
		}
	})
}

func TestEmbedder_RealAPI(t *testing.T) {
	_ = godotenv.Load("../../.env")

	apiKey := os.Getenv("QWELLI_EMBEDDING_KEY")
	model := os.Getenv("QWELLI_EMBEDDING_MODEL")
	endpoint := os.Getenv("QWELLI_EMBEDDING_ENDPOINT")

	if apiKey == "" || model == "" || endpoint == "" {
		t.Fatal("QWELLI_EMBEDDING_KEY, QWELLI_EMBEDDING_MODEL, and QWELLI_EMBEDDING_ENDPOINT must be set for real API tests")
	}

	// Always use Voyage API

	embedder, err := NewEmbedder("voyage", apiKey, model, endpoint)
	if err != nil {
		t.Fatalf("NewEmbedder() error = %v", err)
	}

	t.Run("embed_via_wrapper", func(t *testing.T) {
		embedding, err := embedder.Embed("Test embedding via wrapper")
		if err != nil {
			t.Fatalf("Embedder.Embed() error = %v", err)
		}

		if len(embedding) == 0 {
			t.Fatal("Embedder.Embed() returned empty embedding")
		}
	})

	t.Run("embed_batch_via_wrapper", func(t *testing.T) {
		texts := []string{"text one", "text two"}
		embeddings, err := embedder.EmbedBatch(texts)
		if err != nil {
			t.Fatalf("Embedder.EmbedBatch() error = %v", err)
		}

		if len(embeddings) != len(texts) {
			t.Fatalf("len(embeddings) = %d, want %d", len(embeddings), len(texts))
		}
	})
}
