package chunker

// Chunk represents a piece of content with metadata
// It unifies text-based, PDF-based, and image-based chunking
type Chunk struct {
	Content     string
	ContentType string // "text", "image", or "multimodal"
	PageNumbers []int  // Empty for non-PDF content
	ChunkIndex  int
	TotalChunks int
	StartToken  int // Optional: token position in original document
	EndToken    int // Optional: token position in original document
	Metadata    map[string]interface{}

	// Image-specific fields (nil/empty for text chunks)
	ImageData   []byte
	ImageFormat string
	ImageWidth  int
	ImageHeight int
}

// ChunkerConfig defines chunking parameters
type ChunkerConfig struct {
	ChunkSize   int // Target tokens per chunk (e.g., 1000)
	OverlapSize int // Overlap tokens between chunks (e.g., 150)
}

// StrategyError represents an error in chunking
type StrategyError struct {
	Message string
}

func (e *StrategyError) Error() string {
	return e.Message
}
