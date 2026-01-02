package indexer

import (
	"context"
)

// EmbeddingProvider is the interface that all embedding providers must implement
type EmbeddingProvider interface {
	Embed(text string) ([]float32, error)
	EmbedBatch(ctx context.Context, texts []string, progressCallback func(current, total int)) ([][]float32, error)
}

// MultimodalEmbeddingProvider extends EmbeddingProvider with image support
type MultimodalEmbeddingProvider interface {
	EmbeddingProvider
	EmbedImage(imageBase64 string) ([]float32, error)
	EmbedMultimodal(ctx context.Context, inputs []MultimodalInput, progressCallback func(current, total int)) ([][]float32, error)
}
