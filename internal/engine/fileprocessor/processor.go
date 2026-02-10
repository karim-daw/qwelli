package fileprocessor

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
	FileBytes        []byte // Optional: pre-read file bytes for PDF (avoids redundant disk read)
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
