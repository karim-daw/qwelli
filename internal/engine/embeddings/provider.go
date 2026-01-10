package embeddings

import (
	"context"
	"fmt"
	"log"

	"github.com/karim-daw/qwelli/internal/voyage"
)

// ============================================================================
// INTERFACES
// ============================================================================

// EmbeddingProvider is the interface that all embedding providers must implement
type EmbeddingProvider interface {
	Embed(text string) ([]float32, error)
	EmbedBatch(ctx context.Context, texts []string, progressCallback func(current, total int)) ([][]float32, error)
}

// MultimodalEmbeddingProvider extends EmbeddingProvider with multimodal support
type MultimodalEmbeddingProvider interface {
	EmbeddingProvider
	EmbedMultimodal(ctx context.Context, inputs []MultimodalInput, progressCallback func(current, total int)) ([][]float32, error)
}

// ============================================================================
// SHARED TYPES
// ============================================================================

// MultimodalInput represents either text or image input for embedding
type MultimodalInput struct {
	Type        string // "text" or "image"
	Text        string // For text inputs
	ImageBase64 string // For image inputs (base64-encoded, without data URI prefix)
	ImageFormat string // For image inputs: "jpeg", "png", "gif", etc. (used to add data URI prefix)
	ImagePixels int    // For image inputs: pixel count (width * height) for token calculation
}

// ============================================================================
// EMBEDDER
// ============================================================================

// Embedder wraps an EmbeddingProvider and provides a unified interface
type Embedder struct {
	provider EmbeddingProvider
}

// NewEmbedder creates an embedder using a Voyage client
func NewEmbedder(client *voyage.Client) (*Embedder, error) {
	if client == nil {
		return nil, fmt.Errorf("voyage client is required")
	}
	provider := &voyageEmbeddingProvider{client: client}
	log.Printf("Voyage embedding provider initialized (model: %s)", client.EmbeddingModel())
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

// ============================================================================
// VOYAGE AI PROVIDER (internal, thin wrapper around voyage.Client)
// ============================================================================

// voyageEmbeddingProvider implements MultimodalEmbeddingProvider using voyage.Client
type voyageEmbeddingProvider struct {
	client *voyage.Client
}

// Embed generates an embedding for a single text
func (p *voyageEmbeddingProvider) Embed(text string) ([]float32, error) {
	return p.client.Embed(text)
}

// EmbedBatch generates embeddings for multiple texts
func (p *voyageEmbeddingProvider) EmbedBatch(ctx context.Context, texts []string, progressCallback func(current, total int)) ([][]float32, error) {
	return p.client.EmbedBatch(ctx, texts, progressCallback)
}

// EmbedMultimodal generates embeddings for mixed text and image inputs
func (p *voyageEmbeddingProvider) EmbedMultimodal(ctx context.Context, inputs []MultimodalInput, progressCallback func(current, total int)) ([][]float32, error) {
	// Convert to voyage.MultimodalInput
	voyageInputs := make([]voyage.MultimodalInput, len(inputs))
	for i, input := range inputs {
		voyageInputs[i] = voyage.MultimodalInput{
			Type:        input.Type,
			Text:        input.Text,
			ImageBase64: input.ImageBase64,
			ImageFormat: input.ImageFormat,
			ImagePixels: input.ImagePixels,
		}
	}
	return p.client.EmbedMultimodal(ctx, voyageInputs, progressCallback)
}
