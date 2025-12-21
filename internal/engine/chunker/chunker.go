package chunker

import (
	"github.com/karim-daw/qwelli/internal/engine/processor"
)

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

// ChunkStrategy defines the interface for different chunking strategies
// This interface allows for extensible chunking of different content types:
// - TextChunkStrategy: plain text
// - PDFChunkStrategy: PDF pages (text)
// - ImageChunkStrategy: standalone images (future)
// - MultimodalChunkStrategy: PDFs with text + images
type ChunkStrategy interface {
	// Chunkify processes content and returns chunks
	// The content type depends on the strategy implementation:
	// - TextChunkStrategy: string
	// - PDFChunkStrategy: []processor.PDFPage
	// - ImageChunkStrategy: []processor.PDFImage or []ImageFile (future)
	Chunkify(content interface{}, config ChunkerConfig, baseMetadata map[string]interface{}) ([]Chunk, error)
}

// Chunker splits content into overlapping chunks using a strategy
type Chunker struct {
	config   ChunkerConfig
	strategy ChunkStrategy
}

// NewChunker creates a new Chunker with the given configuration and strategy
func NewChunker(config ChunkerConfig) *Chunker {
	return &Chunker{
		config:   config,
		strategy: &TextChunkStrategy{},
	}
}

// NewChunkerWithStrategy creates a new Chunker with a custom strategy
func NewChunkerWithStrategy(config ChunkerConfig, strategy ChunkStrategy) *Chunker {
	return &Chunker{
		config:   config,
		strategy: strategy,
	}
}

// ChunkByTokens splits text into overlapping chunks by token count
// It attempts to respect sentence boundaries when possible
// This is a convenience method that uses the default text strategy
func (c *Chunker) ChunkByTokens(text string, baseMetadata map[string]interface{}) ([]Chunk, error) {
	return c.strategy.Chunkify(text, c.config, baseMetadata)
}

// ChunkPDFPages chunks PDF pages while preserving page number context
// This is a convenience method that uses the PDF strategy
func (c *Chunker) ChunkPDFPages(pages []processor.PDFPage, metadata *processor.PDFMetadata, filePath string) ([]Chunk, error) {
	pdfStrategy := NewPDFChunkStrategy(metadata, filePath)
	return pdfStrategy.Chunkify(pages, c.config, make(map[string]interface{}))
}

// StrategyError represents an error in chunking strategy
type StrategyError struct {
	Message string
}

func (e *StrategyError) Error() string {
	return e.Message
}
