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

// ProcessOptions contains options for file processing
type ProcessOptions struct {
	EnableMultimodal bool
	ChunkSize        int
	OverlapSize      int
	FileContent      string // Optional: file content for text files
}

// DefaultProcessOptions returns default processing options
func DefaultProcessOptions() ProcessOptions {
	return ProcessOptions{
		EnableMultimodal: false,
		ChunkSize:        300,
		OverlapSize:      10,
	}
}
