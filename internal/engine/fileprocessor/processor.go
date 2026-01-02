package fileprocessor

import (
	"github.com/karim-daw/qwelli/internal/db"
)

// FileProcessor defines the interface for processing different file types
// This allows for extensible file type support (PDF, text, images, etc.)
type FileProcessor interface {
	// CanProcess returns true if this processor can handle the given file type
	CanProcess(fileType string) bool

	// Process processes a file and returns chunks
	// The file parameter contains file metadata
	// Returns chunks and content strings for embedding
	Process(file db.File, options ProcessOptions) ([]db.Chunk, []string, error)
}

// ContentTypeMode specifies what content types to index
type ContentTypeMode string

const (
	ContentTypeBoth   ContentTypeMode = "both"   // Index both text and images
	ContentTypeText   ContentTypeMode = "text"   // Index text only
	ContentTypeImages ContentTypeMode = "images" // Index images only
)

// ProcessOptions contains options for file processing
type ProcessOptions struct {
	EnableMultimodal bool
	ContentTypeMode  ContentTypeMode // What content types to index: "both", "text", or "images"
	ChunkSize        int
	OverlapSize      int
	FileContent      string // Optional: file content for text files
}

// DefaultProcessOptions returns default processing options
func DefaultProcessOptions() ProcessOptions {
	return ProcessOptions{
		EnableMultimodal: true,
		ContentTypeMode:  ContentTypeBoth,
		ChunkSize:        100,
		OverlapSize:      5,
	}
}
