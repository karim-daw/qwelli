package voyage

import "context"

// ClientInterface defines the interface for Voyage AI client operations
// This interface allows for easier testing by enabling mock implementations
type ClientInterface interface {
	// Embedding methods
	Embed(text string) ([]float32, error)
	EmbedBatch(ctx context.Context, texts []string, progressCallback func(current, total int)) ([][]float32, error)
	EmbedMultimodal(ctx context.Context, inputs []MultimodalInput, progressCallback func(current, total int)) ([][]float32, error)

	// Reranking methods
	Rerank(ctx context.Context, query string, documents []string) ([]RerankResult, error)
	RerankWithMetadata(ctx context.Context, query string, inputs []RerankInput) ([]RerankResult, error)

	// Model information
	EmbeddingModel() string
	RerankModel() string
}

// Verify that Client implements ClientInterface at compile time
var _ ClientInterface = (*Client)(nil)
