package indexer

import (
	"fmt"
)

// EmbeddingProvider is the interface that all embedding providers must implement
type EmbeddingProvider interface {
	Embed(text string) ([]float32, error)
	EmbedBatch(texts []string) ([][]float32, error)
}

// Embedder wraps an EmbeddingProvider
type Embedder struct {
	provider EmbeddingProvider
}

// NewEmbedder creates an embedder with explicit configuration
// Uses HTTP-based provider compatible with OpenAI, VoyageAI, and other APIs
func NewEmbedder(apiKey, model, endpoint string) (*Embedder, error) {
	provider, err := NewHTTPEmbeddingProvider(apiKey, model, endpoint)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize embedding provider: %w", err)
	}
	return &Embedder{provider: provider}, nil
}

// Embed generates an embedding for a single text
func (e *Embedder) Embed(text string) ([]float32, error) {
	return e.provider.Embed(text)
}

// EmbedBatch generates embeddings for multiple texts
func (e *Embedder) EmbedBatch(texts []string) ([][]float32, error) {
	return e.provider.EmbedBatch(texts)
}
