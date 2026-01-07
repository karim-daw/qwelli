package fileprocessor

import (
	"testing"

	"github.com/karim-daw/qwelli/internal/db"
	"github.com/karim-daw/qwelli/internal/engine/chunker"
)

func TestNewFileProcessingService(t *testing.T) {
	config := DefaultProcessingConfig()
	service := NewFileProcessingService(config)

	if service == nil {
		t.Fatal("NewFileProcessingService returned nil")
	}

	if service.chunkService == nil {
		t.Error("chunkService not initialized")
	}

	if service.pdfProcessor == nil {
		t.Error("pdfProcessor not initialized")
	}

	if service.imageExtractor == nil {
		t.Error("imageExtractor not initialized")
	}
}

func TestFileProcessingService_CanProcess(t *testing.T) {
	service := NewFileProcessingService(DefaultProcessingConfig())

	tests := []struct {
		name     string
		fileType string
		want     bool
	}{
		{"text file", "txt", true},
		{"go file", "go", true},
		{"pdf file", "pdf", true},
		{"markdown", "md", true},
		{"unsupported", "exe", false},
		{"unknown", "xyz", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := service.CanProcess(tt.fileType)
			if got != tt.want {
				t.Errorf("CanProcess(%s) = %v, want %v", tt.fileType, got, tt.want)
			}
		})
	}
}

func TestFileProcessingService_ProcessText(t *testing.T) {
	service := NewFileProcessingService(DefaultProcessingConfig())

	file := db.File{
		FileID:   "test_file_1",
		Path:     "/path/to/test.txt",
		FileType: "txt",
	}

	content := "This is a test file with some content."

	chunks, contents, err := service.ProcessText(file, content)
	if err != nil {
		t.Fatalf("ProcessText failed: %v", err)
	}

	if len(chunks) == 0 {
		t.Error("Expected at least one chunk")
	}

	if len(contents) != len(chunks) {
		t.Errorf("Contents length %d != chunks length %d", len(contents), len(chunks))
	}

	if chunks[0].Content != content {
		t.Errorf("Chunk content doesn't match input")
	}
}

func TestFileProcessingService_ProcessTextLarge(t *testing.T) {
	service := NewFileProcessingService(DefaultProcessingConfig())

	file := db.File{
		FileID:   "test_file_2",
		Path:     "/path/to/large.txt",
		FileType: "txt",
	}

	// Create a large text that exceeds 1000 tokens
	var largeText string
	for i := 0; i < 500; i++ {
		largeText += "This is sentence number " + string(rune(i)) + ". It contains some meaningful content. "
	}

	chunks, contents, err := service.ProcessText(file, largeText)
	if err != nil {
		t.Fatalf("ProcessText failed: %v", err)
	}

	if len(chunks) <= 1 {
		t.Error("Expected multiple chunks for large text")
	}

	if len(contents) != len(chunks) {
		t.Errorf("Contents length %d != chunks length %d", len(contents), len(chunks))
	}
}

func TestSetContentTypeMode(t *testing.T) {
	service := NewFileProcessingService(DefaultProcessingConfig())

	// Default should be ContentTypeBoth
	if service.config.ContentTypeMode != ContentTypeBoth {
		t.Errorf("Default content type mode should be ContentTypeBoth, got %s", service.config.ContentTypeMode)
	}

	// Test changing mode
	service.SetContentTypeMode(ContentTypeText)
	if service.config.ContentTypeMode != ContentTypeText {
		t.Errorf("Content type mode not updated, got %s", service.config.ContentTypeMode)
	}

	service.SetContentTypeMode(ContentTypeImages)
	if service.config.ContentTypeMode != ContentTypeImages {
		t.Errorf("Content type mode not updated, got %s", service.config.ContentTypeMode)
	}
}

func TestFilterChunksByContentType(t *testing.T) {
	service := NewFileProcessingService(DefaultProcessingConfig())

	chunks := []chunker.Chunk{
		{Content: "text1", ContentType: "text"},
		{Content: "img1", ContentType: "image"},
		{Content: "text2", ContentType: "text"},
		{Content: "img2", ContentType: "image"},
	}

	tests := []struct {
		name string
		mode ContentTypeMode
		want int
	}{
		{"both", ContentTypeBoth, 4},
		{"text only", ContentTypeText, 2},
		{"images only", ContentTypeImages, 2},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			filtered := service.filterChunksByContentType(chunks, tt.mode)
			if len(filtered) != tt.want {
				t.Errorf("filterChunksByContentType(%s) returned %d chunks, want %d", tt.mode, len(filtered), tt.want)
			}
		})
	}
}
