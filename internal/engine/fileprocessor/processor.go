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

// Registry manages file processors
type Registry struct {
	processors []FileProcessor
}

// NewRegistry creates a new file processor registry with default processors
func NewRegistry() *Registry {
	return &Registry{
		processors: []FileProcessor{
			NewPDFProcessor(),
			NewTextProcessor(),
		},
	}
}

// Register adds a new file processor to the registry
func (r *Registry) Register(processor FileProcessor) {
	r.processors = append(r.processors, processor)
}

// GetProcessor returns the first processor that can handle the given file type
func (r *Registry) GetProcessor(fileType string) FileProcessor {
	for _, processor := range r.processors {
		if processor.CanProcess(fileType) {
			return processor
		}
	}
	return nil
}

// DefaultRegistry is the default global registry
var DefaultRegistry = NewRegistry()

// GetProcessor returns a processor from the default registry
func GetProcessor(fileType string) FileProcessor {
	return DefaultRegistry.GetProcessor(fileType)
}
