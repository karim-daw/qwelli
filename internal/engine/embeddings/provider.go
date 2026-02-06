package embeddings

import (
	"context"
	"fmt"
	"log"

	"github.com/karim-daw/qwelli/internal/voyage"
)

// MultimodalInput represents either text or image input for embedding
type MultimodalInput = voyage.MultimodalInput

// EmbeddingProvider is the interface for embedding providers
type EmbeddingProvider interface {
	Embed(text string) ([]float32, error)
	EmbedBatch(ctx context.Context, texts []string, progressCallback func(current, total int)) ([][]float32, error)
}

// MultimodalEmbeddingProvider extends EmbeddingProvider with image support
type MultimodalEmbeddingProvider interface {
	EmbeddingProvider
	EmbedMultimodal(ctx context.Context, inputs []MultimodalInput, progressCallback func(current, total int)) ([][]float32, error)
}

// Embedder wraps a voyage.Client with a convenient interface
type Embedder struct {
	client *voyage.Client
}

func NewEmbedder(client *voyage.Client) (*Embedder, error) {
	if client == nil {
		return nil, fmt.Errorf("voyage client is required")
	}
	log.Printf("Voyage embedding provider initialized (model: %s)", client.EmbeddingModel())
	return &Embedder{client: client}, nil
}

func (e *Embedder) Embed(text string) ([]float32, error) {
	return e.client.Embed(text)
}

func (e *Embedder) EmbedBatch(ctx context.Context, texts []string, cb func(int, int)) ([][]float32, error) {
	return e.client.EmbedBatch(ctx, texts, cb)
}

func (e *Embedder) IsMultimodal() bool {
	return true
}

func (e *Embedder) EmbedMultimodal(ctx context.Context, inputs []MultimodalInput, cb func(int, int)) ([][]float32, error) {
	return e.client.EmbedMultimodal(ctx, inputs, cb)
}
