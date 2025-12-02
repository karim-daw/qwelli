package indexer

import "fmt"

// EmbeddingProvider is the interface that all embedding providers must implement
type EmbeddingProvider interface {
	// Embed generates an embedding for a single text
	Embed(text string) ([]float32, error)

	// EmbedBatch generates embeddings for multiple texts
	EmbedBatch(texts []string) ([][]float32, error)

	// Dimension returns the embedding dimension
	Dimension() int

	// Close cleans up resources
	Close() error
}

// Config holds configuration for creating an embedding provider
type Config struct {
	// Provider type: "openai", "cohere"
	Provider string

	// Model name (provider-specific)
	Model string

	// API key
	APIKey string

	// Endpoint URL (for custom endpoints)
	Endpoint string

	// Expected dimension (0 for auto-detect)
	Dimension int
}

// NewProvider creates an embedding provider based on configuration
func NewProvider(cfg Config) (EmbeddingProvider, error) {
	switch cfg.Provider {
	case "openai", "":
		// Default to OpenAI
		return NewOpenAIProvider(cfg)

	case "cohere":
		return NewCohereProvider(cfg)

	default:
		return nil, fmt.Errorf("unknown provider: %s (supported: openai, cohere)", cfg.Provider)
	}
}
