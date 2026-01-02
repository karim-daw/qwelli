package indexer

import (
	"context"
	"fmt"
)

// Embedder wraps an EmbeddingProvider
type Embedder struct {
	provider EmbeddingProvider
}

// NewEmbedder creates an embedder with Voyage AI provider
func NewEmbedder(apiKey, model, endpoint string) (*Embedder, error) {
	provider, err := NewVoyageEmbeddingProvider(apiKey, model, endpoint)
	if err != nil {
		return nil, err
	}
	return &Embedder{provider: provider}, nil
}

// Embed generates an embedding for a single text
func (e *Embedder) Embed(text string) ([]float32, error) {
	return e.provider.Embed(text)
}

// EmbedBatch generates embeddings for multiple texts
// progressCallback is called with (current, total) as batches are processed
func (e *Embedder) EmbedBatch(ctx context.Context, texts []string, progressCallback func(current, total int)) ([][]float32, error) {
	return e.provider.EmbedBatch(ctx, texts, progressCallback)
}

// IsMultimodal returns true if the provider supports multimodal embeddings
func (e *Embedder) IsMultimodal() bool {
	_, ok := e.provider.(MultimodalEmbeddingProvider)
	return ok
}

// GetMultimodalProvider returns the provider as MultimodalEmbeddingProvider if supported
func (e *Embedder) GetMultimodalProvider() (MultimodalEmbeddingProvider, bool) {
	provider, ok := e.provider.(MultimodalEmbeddingProvider)
	return provider, ok
}

// EmbedImage generates an embedding for a single image (base64-encoded)
// Returns an error if the provider doesn't support multimodal embeddings
func (e *Embedder) EmbedImage(imageBase64 string) ([]float32, error) {
	multimodalProvider, ok := e.provider.(MultimodalEmbeddingProvider)
	if !ok {
		return nil, fmt.Errorf("provider does not support image embeddings")
	}
	return multimodalProvider.EmbedImage(imageBase64)
}

// EmbedMultimodal generates embeddings for mixed text and image inputs
// Returns an error if the provider doesn't support multimodal embeddings
// progressCallback is called with (current, total) as batches are processed
func (e *Embedder) EmbedMultimodal(ctx context.Context, inputs []MultimodalInput, progressCallback func(current, total int)) ([][]float32, error) {
	multimodalProvider, ok := e.provider.(MultimodalEmbeddingProvider)
	if !ok {
		return nil, fmt.Errorf("provider does not support multimodal embeddings")
	}
	return multimodalProvider.EmbedMultimodal(ctx, inputs, progressCallback)
}
