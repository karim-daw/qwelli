package indexer

// EmbeddingProvider is the interface that all embedding providers must implement
type EmbeddingProvider interface {
	Embed(text string) ([]float32, error)
	EmbedBatch(texts []string) ([][]float32, error)
}

// MultimodalEmbeddingProvider extends EmbeddingProvider with image support
type MultimodalEmbeddingProvider interface {
	EmbeddingProvider
	EmbedImage(imageBase64 string) ([]float32, error)
	EmbedMultimodal(inputs []MultimodalInput) ([][]float32, error)
}
