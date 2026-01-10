package embeddings

import (
	"context"
	"os"
	"strings"
	"testing"

	"github.com/joho/godotenv"
	"github.com/karim-daw/qwelli/internal/voyage"
)

func TestNewEmbedder_Validation(t *testing.T) {
	t.Run("missing_api_key", func(t *testing.T) {
		_, err := voyage.NewClient(voyage.ClientConfig{APIKey: ""})
		if err == nil {
			t.Fatal("NewClient() expected error for missing API key, got nil")
		}
		// Check that error message mentions API key
		if !strings.Contains(strings.ToLower(err.Error()), "api key") {
			t.Fatalf("NewClient() error = %v, want error mentioning 'API key'", err)
		}
	})

	t.Run("valid_config", func(t *testing.T) {
		client, err := voyage.NewClient(voyage.ClientConfig{
			APIKey:            "test-key",
			EmbeddingModel:    "test-model",
			EmbeddingEndpoint: "http://localhost/v1/embeddings",
		})
		if err != nil {
			t.Fatalf("NewClient() error = %v", err)
		}

		embedder, err := NewEmbedder(client)
		if err != nil {
			t.Fatalf("NewEmbedder() error = %v", err)
		}
		if embedder == nil {
			t.Fatal("NewEmbedder() returned nil")
		}
	})
}

func TestEmbedder_RealAPI(t *testing.T) {
	// Try loading .env file from project root
	// Try multiple possible paths
	envPaths := []string{"../../.env", "../../../.env", ".env"}
	var envLoaded bool
	for _, path := range envPaths {
		if err := godotenv.Load(path); err == nil {
			envLoaded = true
			break
		}
	}
	if !envLoaded {
		t.Logf("Note: Could not load .env file from any of: %v (will use environment variables if set)", envPaths)
	}

	apiKey := os.Getenv("QWELLI_EMBEDDING_KEY")
	model := os.Getenv("QWELLI_EMBEDDING_MODEL")
	endpoint := os.Getenv("QWELLI_EMBEDDING_ENDPOINT")

	if apiKey == "" || model == "" || endpoint == "" {
		var missing []string
		if apiKey == "" {
			missing = append(missing, "QWELLI_EMBEDDING_KEY")
		}
		if model == "" {
			missing = append(missing, "QWELLI_EMBEDDING_MODEL")
		}
		if endpoint == "" {
			missing = append(missing, "QWELLI_EMBEDDING_ENDPOINT")
		}
		t.Fatalf("Missing required environment variables for real API tests: %v. Set these in your .env file (uncommented) or environment.", missing)
	}

	client, err := voyage.NewClient(voyage.ClientConfig{
		APIKey:            apiKey,
		EmbeddingModel:    model,
		EmbeddingEndpoint: endpoint,
	})
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}

	embedder, err := NewEmbedder(client)
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
		embeddings, err := embedder.EmbedBatch(context.Background(), texts, nil)
		if err != nil {
			t.Fatalf("Embedder.EmbedBatch() error = %v", err)
		}

		if len(embeddings) != len(texts) {
			t.Fatalf("len(embeddings) = %d, want %d", len(embeddings), len(texts))
		}
	})
}
