package chunker

import (
	"testing"

	"github.com/karim-daw/qwelli/internal/engine/processor"
)

func TestNewChunkService(t *testing.T) {
	config := ChunkerConfig{
		ChunkSize:   100,
		OverlapSize: 10,
	}

	service := NewChunkService(config)

	if service == nil {
		t.Fatal("NewChunkService returned nil")
	}

	if service.config.ChunkSize != 100 {
		t.Errorf("Expected ChunkSize 100, got %d", service.config.ChunkSize)
	}

	if service.config.OverlapSize != 10 {
		t.Errorf("Expected OverlapSize 10, got %d", service.config.OverlapSize)
	}
}

func TestDefaultConfig(t *testing.T) {
	if DefaultConfig.ChunkSize != 300 {
		t.Errorf("Expected DefaultConfig.ChunkSize 300, got %d", DefaultConfig.ChunkSize)
	}

	if DefaultConfig.OverlapSize != 10 {
		t.Errorf("Expected DefaultConfig.OverlapSize 10, got %d", DefaultConfig.OverlapSize)
	}
}

func TestChunkService_ChunkText(t *testing.T) {
	service := NewChunkService(DefaultConfig)

	text := "This is a test. This is another test sentence."

	chunks, err := service.ChunkText(text)
	if err != nil {
		t.Fatalf("ChunkText failed: %v", err)
	}

	if len(chunks) == 0 {
		t.Fatal("Expected at least one chunk")
	}

	if chunks[0].ContentType != "text" {
		t.Errorf("Expected ContentType 'text', got '%s'", chunks[0].ContentType)
	}
}

func TestChunkService_ChunkPDF(t *testing.T) {
	service := NewChunkService(DefaultConfig)

	// Create test PDF pages
	pages := []processor.PDFPage{
		{PageNumber: 1, Text: "This is page one content."},
		{PageNumber: 2, Text: "This is page two content."},
	}

	metadata := &processor.PDFMetadata{
		Title:     "Test PDF",
		PageCount: 2,
	}

	chunks, err := service.ChunkPDF(pages, metadata, "/path/to/test.pdf")
	if err != nil {
		t.Fatalf("ChunkPDF failed: %v", err)
	}

	if len(chunks) == 0 {
		t.Fatal("Expected at least one chunk")
	}

	// Verify chunks have page numbers
	for _, chunk := range chunks {
		if len(chunk.PageNumbers) == 0 {
			t.Error("Expected chunk to have page numbers")
		}
	}
}

func TestChunkService_ChunkMultimodal(t *testing.T) {
	service := NewChunkService(DefaultConfig)

	// Create test PDF pages
	pages := []processor.PDFPage{
		{PageNumber: 1, Text: "This is page one content."},
	}

	// Create test images
	images := []processor.PDFImage{
		{
			PageNumber: 1,
			Width:      1000,
			Height:     800,
			Format:     "jpeg",
			Base64:     "dGVzdA==", // "test" in base64
		},
	}

	metadata := &processor.PDFMetadata{
		Title:     "Test PDF",
		PageCount: 1,
	}

	chunks, err := service.ChunkMultimodal(pages, images, metadata, "/path/to/test.pdf")
	if err != nil {
		t.Fatalf("ChunkMultimodal failed: %v", err)
	}

	if len(chunks) < 2 {
		t.Fatalf("Expected at least 2 chunks (text + image), got %d", len(chunks))
	}

	// Verify we have both text and image chunks
	hasText := false
	hasImage := false
	for _, chunk := range chunks {
		if chunk.ContentType == "text" {
			hasText = true
		}
		if chunk.ContentType == "image" {
			hasImage = true
		}
	}

	if !hasText {
		t.Error("Expected at least one text chunk")
	}

	if !hasImage {
		t.Error("Expected at least one image chunk")
	}
}
