package indexer

import (
	"fmt"
	"os"
)

// Embedder wraps an EmbeddingProvider for backward compatibility
type Embedder struct {
	provider EmbeddingProvider
}

// NewEmbedder creates a new embedder using environment variables or defaults
// modelName: model to use (provider-specific)
// modelDir: deprecated, kept for backward compatibility (ignored)
// expectedDim: deprecated, dimension is auto-detected
func NewEmbedder(modelName string, modelDir string, expectedDim int) (*Embedder, error) {
	// Check environment variable for provider selection
	providerType := os.Getenv("QWELLI_EMBEDDING_PROVIDER")
	if providerType == "" {
		providerType = "openai" // Default to OpenAI
	}

	cfg := Config{
		Provider:  providerType,
		Model:     modelName,
		APIKey:    "", // Will be read from env in provider
		Endpoint:  "",
		Dimension: expectedDim,
	}

	// Override model from environment if set
	if envModel := os.Getenv("QWELLI_EMBEDDING_MODEL"); envModel != "" {
		cfg.Model = envModel
	}

	provider, err := NewProvider(cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize embedding provider: %w", err)
	}

	return &Embedder{
		provider: provider,
	}, nil
}

// NewEmbedderWithProvider creates an embedder with explicit configuration
func NewEmbedderWithProvider(cfg Config) (*Embedder, error) {
	provider, err := NewProvider(cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize embedding provider: %w", err)
	}

	return &Embedder{
		provider: provider,
	}, nil
}

// Embed generates an embedding for a single text
func (e *Embedder) Embed(text string) ([]float32, error) {
	return e.provider.Embed(text)
}

// EmbedBatch generates embeddings for multiple texts
func (e *Embedder) EmbedBatch(texts []string) ([][]float32, error) {
	return e.provider.EmbedBatch(texts)
}

// Dimension returns the embedding dimension
func (e *Embedder) Dimension() int {
	return e.provider.Dimension()
}

// Close cleans up resources
func (e *Embedder) Close() error {
	return e.provider.Close()
}
