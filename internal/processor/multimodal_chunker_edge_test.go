package processor

import (
	"testing"
)

func TestMultimodalChunker_EmptyPagesAndImages(t *testing.T) {
	pdfChunker := NewPDFChunker(ChunkerConfig{
		ChunkSize:   300,
		OverlapSize: 10,
	})
	imageExtractor := NewImageExtractor(1024, 1024)
	chunker := NewMultimodalChunker(pdfChunker, imageExtractor)

	chunks, err := chunker.ChunkPDF([]PDFPage{}, []PDFImage{}, nil, "test.pdf")
	if err != nil {
		t.Fatalf("ChunkPDF() with empty inputs error = %v", err)
	}

	// Should return empty chunks or handle gracefully
	// The actual behavior depends on PDFChunker - it might return an empty chunk
	if chunks == nil {
		t.Error("ChunkPDF() returned nil, want empty slice or valid chunks")
	}
}

func TestMultimodalChunker_ManyImagesSamePage(t *testing.T) {
	pdfChunker := NewPDFChunker(ChunkerConfig{
		ChunkSize:   300,
		OverlapSize: 10,
	})
	imageExtractor := NewImageExtractor(1024, 1024)
	chunker := NewMultimodalChunker(pdfChunker, imageExtractor)

	pages := []PDFPage{
		{PageNumber: 1, Text: "Page with many images"},
	}

	// Create 5 images on the same page
	images := make([]PDFImage, 5)
	for i := 0; i < 5; i++ {
		images[i] = PDFImage{
			Data:       []byte("image data"),
			Format:     "jpeg",
			Width:      800,
			Height:     600,
			PageNumber: 1,
			Base64:     "base64data",
		}
	}

	metadata := &PDFMetadata{
		PageCount: 1,
		Title:     "Test PDF",
	}

	chunks, err := chunker.ChunkPDF(pages, images, metadata, "test.pdf")
	if err != nil {
		t.Fatalf("ChunkPDF() error = %v", err)
	}

	// Should have text chunk + 5 image chunks
	if len(chunks) < 6 {
		t.Errorf("ChunkPDF() returned %d chunks, want at least 6", len(chunks))
	}

	// Verify all images are present
	imageCount := 0
	for _, chunk := range chunks {
		if chunk.ContentType == "image" {
			imageCount++
		}
	}

	if imageCount != 5 {
		t.Errorf("ChunkPDF() returned %d image chunks, want 5", imageCount)
	}
}

func TestMultimodalChunker_ImageMetadata(t *testing.T) {
	pdfChunker := NewPDFChunker(ChunkerConfig{
		ChunkSize:   300,
		OverlapSize: 10,
	})
	imageExtractor := NewImageExtractor(1024, 1024)
	chunker := NewMultimodalChunker(pdfChunker, imageExtractor)

	pages := []PDFPage{
		{PageNumber: 1, Text: "Test page"},
	}

	images := []PDFImage{
		{
			Data:       []byte("image"),
			Format:     "png",
			Width:      1920,
			Height:     1080,
			PageNumber: 1,
			Base64:     "base64",
		},
	}

	metadata := &PDFMetadata{
		PageCount: 1,
		Title:     "Test",
	}

	chunks, err := chunker.ChunkPDF(pages, images, metadata, "test.pdf")
	if err != nil {
		t.Fatalf("ChunkPDF() error = %v", err)
	}

	// Find image chunk
	var imageChunk *MultimodalChunk
	for i := range chunks {
		if chunks[i].ContentType == "image" {
			imageChunk = &chunks[i]
			break
		}
	}

	if imageChunk == nil {
		t.Fatal("ChunkPDF() did not create image chunk")
	}

	// Verify metadata
	if imageChunk.ImageFormat != "png" {
		t.Errorf("ChunkPDF() image chunk ImageFormat = %q, want %q", imageChunk.ImageFormat, "png")
	}
	if imageChunk.ImageWidth != 1920 {
		t.Errorf("ChunkPDF() image chunk ImageWidth = %d, want %d", imageChunk.ImageWidth, 1920)
	}
	if imageChunk.ImageHeight != 1080 {
		t.Errorf("ChunkPDF() image chunk ImageHeight = %d, want %d", imageChunk.ImageHeight, 1080)
	}

	// Verify metadata map
	if imageChunk.Metadata == nil {
		t.Fatal("ChunkPDF() image chunk Metadata is nil")
	}

	if format, ok := imageChunk.Metadata["image_format"].(string); !ok || format != "png" {
		t.Errorf("ChunkPDF() image chunk metadata image_format = %v, want %q", imageChunk.Metadata["image_format"], "png")
	}

	if width, ok := imageChunk.Metadata["image_width"].(int); !ok || width != 1920 {
		t.Errorf("ChunkPDF() image chunk metadata image_width = %v, want %d", imageChunk.Metadata["image_width"], 1920)
	}

	if height, ok := imageChunk.Metadata["image_height"].(int); !ok || height != 1080 {
		t.Errorf("ChunkPDF() image chunk metadata image_height = %v, want %d", imageChunk.Metadata["image_height"], 1080)
	}
}

func TestMultimodalChunker_SortingOrder(t *testing.T) {
	pdfChunker := NewPDFChunker(ChunkerConfig{
		ChunkSize:   300,
		OverlapSize: 10,
	})
	imageExtractor := NewImageExtractor(1024, 1024)
	chunker := NewMultimodalChunker(pdfChunker, imageExtractor)

	pages := []PDFPage{
		{PageNumber: 1, Text: "Page 1"},
		{PageNumber: 2, Text: "Page 2"},
		{PageNumber: 3, Text: "Page 3"},
	}

	// Images on pages 2 and 3
	images := []PDFImage{
		{
			Data:       []byte("img1"),
			Format:     "jpeg",
			Width:      800,
			Height:     600,
			PageNumber: 3, // Image on page 3
			Base64:     "base64",
		},
		{
			Data:       []byte("img2"),
			Format:     "png",
			Width:      800,
			Height:     600,
			PageNumber: 2, // Image on page 2
			Base64:     "base64",
		},
	}

	metadata := &PDFMetadata{
		PageCount: 3,
		Title:     "Test",
	}

	chunks, err := chunker.ChunkPDF(pages, images, metadata, "test.pdf")
	if err != nil {
		t.Fatalf("ChunkPDF() error = %v", err)
	}

	// Verify sorting: should be page 1 text, page 2 text, page 2 image, page 3 text, page 3 image
	if len(chunks) < 5 {
		t.Fatalf("ChunkPDF() returned %d chunks, want at least 5", len(chunks))
	}

	// Check page order
	for i := 1; i < len(chunks); i++ {
		if chunks[i].PageNumber < chunks[i-1].PageNumber {
			t.Errorf("ChunkPDF() chunks not sorted: chunk %d page %d < chunk %d page %d",
				i, chunks[i].PageNumber, i-1, chunks[i-1].PageNumber)
		}
	}

	// On page 2, text should come before image
	foundPage2Text := false
	for _, chunk := range chunks {
		if chunk.PageNumber == 2 {
			switch chunk.ContentType {
			case "text":
				foundPage2Text = true
			case "image":
				if !foundPage2Text {
					t.Error("ChunkPDF() page 2: image chunk comes before text chunk")
				}
			}
		}
	}
}
