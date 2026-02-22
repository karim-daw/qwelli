package fileprocessor

// ContentTypeMode specifies what content types to index
type ContentTypeMode string

const (
	ContentTypeText          ContentTypeMode = "text"          // Text files + PDF text only
	ContentTypeImages        ContentTypeMode = "images"        // Standalone images only
	ContentTypeTextImages    ContentTypeMode = "text_images"   // Text + standalone images (no PDF image extraction)
	ContentTypeTextPDFImages ContentTypeMode = "text_pdfimages" // Text + PDF image extraction (no standalone images)
	ContentTypeAll           ContentTypeMode = "both"          // Everything: text + standalone images + PDF image extraction
)

// IncludesText returns true if text files and PDF text should be indexed.
func (m ContentTypeMode) IncludesText() bool {
	return m != ContentTypeImages
}

// IncludesStandaloneImages returns true if standalone image files should be indexed.
func (m ContentTypeMode) IncludesStandaloneImages() bool {
	return m == ContentTypeImages || m == ContentTypeTextImages || m == ContentTypeAll
}

// IncludesPDFImages returns true if images should be extracted from PDFs.
func (m ContentTypeMode) IncludesPDFImages() bool {
	return m == ContentTypeTextPDFImages || m == ContentTypeAll
}

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
		ContentTypeMode:  ContentTypeAll,
		ChunkSize:        100,
		OverlapSize:      5,
	}
}
