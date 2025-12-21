package processor

import (
	"testing"
)

func TestNewMultimodalChunker(t *testing.T) {
	pdfChunker := NewPDFChunker(ChunkerConfig{
		ChunkSize:   300,
		OverlapSize: 10,
	})
	imageExtractor := NewImageExtractor(1024, 1024)

	chunker := NewMultimodalChunker(pdfChunker, imageExtractor)
	if chunker == nil {
		t.Fatal("NewMultimodalChunker() returned nil")
	}
	if chunker.pdfChunker == nil {
		t.Error("NewMultimodalChunker() pdfChunker is nil")
	}
	if chunker.imageExtractor == nil {
		t.Error("NewMultimodalChunker() imageExtractor is nil")
	}
}

func TestMultimodalChunker_ChunkPDF_TextOnly(t *testing.T) {
	pdfChunker := NewPDFChunker(ChunkerConfig{
		ChunkSize:   300,
		OverlapSize: 10,
	})
	imageExtractor := NewImageExtractor(1024, 1024)
	chunker := NewMultimodalChunker(pdfChunker, imageExtractor)

	pages := []PDFPage{
		{PageNumber: 1, Text: "This is page one content."},
		{PageNumber: 2, Text: "This is page two content."},
	}
	images := []PDFImage{} // No images
	metadata := &PDFMetadata{
		PageCount: 2,
		Title:     "Test PDF",
	}

	chunks, err := chunker.ChunkPDF(pages, images, metadata, "test.pdf")
	if err != nil {
		t.Fatalf("ChunkPDF() error = %v", err)
	}

	if len(chunks) == 0 {
		t.Fatal("ChunkPDF() returned no chunks")
	}

	// Verify all chunks are text type
	for i, chunk := range chunks {
		if chunk.ContentType != "text" {
			t.Errorf("ChunkPDF() chunk %d ContentType = %q, want %q", i, chunk.ContentType, "text")
		}
		if chunk.PageNumber < 1 || chunk.PageNumber > 2 {
			t.Errorf("ChunkPDF() chunk %d PageNumber = %d, want 1 or 2", i, chunk.PageNumber)
		}
		if chunk.ChunkIndex != i {
			t.Errorf("ChunkPDF() chunk %d ChunkIndex = %d, want %d", i, chunk.ChunkIndex, i)
		}
		if chunk.TotalChunks != len(chunks) {
			t.Errorf("ChunkPDF() chunk %d TotalChunks = %d, want %d", i, chunk.TotalChunks, len(chunks))
		}
	}
}

func TestMultimodalChunker_ChunkPDF_WithImages(t *testing.T) {
	pdfChunker := NewPDFChunker(ChunkerConfig{
		ChunkSize:   300,
		OverlapSize: 10,
	})
	imageExtractor := NewImageExtractor(1024, 1024)
	chunker := NewMultimodalChunker(pdfChunker, imageExtractor)

	pages := []PDFPage{
		{PageNumber: 1, Text: "Page one text."},
		{PageNumber: 2, Text: "Page two text."},
	}

	// Create mock images
	images := []PDFImage{
		{
			Data:       []byte("fake image data 1"),
			Format:     "jpeg",
			Width:      800,
			Height:     600,
			PageNumber: 1,
			Base64:     "ZmFrZSBpbWFnZSBkYXRhIDE=", // base64 of "fake image data 1"
		},
		{
			Data:       []byte("fake image data 2"),
			Format:     "png",
			Width:      1200,
			Height:     900,
			PageNumber: 2,
			Base64:     "ZmFrZSBpbWFnZSBkYXRhIDI=", // base64 of "fake image data 2"
		},
	}

	metadata := &PDFMetadata{
		PageCount: 2,
		Title:     "Test PDF",
	}

	chunks, err := chunker.ChunkPDF(pages, images, metadata, "test.pdf")
	if err != nil {
		t.Fatalf("ChunkPDF() error = %v", err)
	}

	// Should have text chunks + image chunks
	if len(chunks) < len(pages)+len(images) {
		t.Errorf("ChunkPDF() returned %d chunks, want at least %d", len(chunks), len(pages)+len(images))
	}

	// Count text and image chunks
	textCount := 0
	imageCount := 0
	for _, chunk := range chunks {
		switch chunk.ContentType {
		case "text":
			textCount++
		case "image":
			imageCount++
			// Verify image chunk properties
			if len(chunk.ImageData) == 0 {
				t.Error("ChunkPDF() image chunk has empty ImageData")
			}
			if chunk.ImageFormat == "" {
				t.Error("ChunkPDF() image chunk has empty ImageFormat")
			}
			if chunk.ImageWidth == 0 || chunk.ImageHeight == 0 {
				t.Error("ChunkPDF() image chunk has invalid dimensions")
			}
		}
	}

	if textCount == 0 {
		t.Error("ChunkPDF() returned no text chunks")
	}
	if imageCount != len(images) {
		t.Errorf("ChunkPDF() returned %d image chunks, want %d", imageCount, len(images))
	}

	// Verify chunks are sorted by page number
	for i := 1; i < len(chunks); i++ {
		if chunks[i].PageNumber < chunks[i-1].PageNumber {
			t.Errorf("ChunkPDF() chunks not sorted: chunk %d page %d < chunk %d page %d",
				i, chunks[i].PageNumber, i-1, chunks[i-1].PageNumber)
		}
		// On same page, text should come before images
		if chunks[i].PageNumber == chunks[i-1].PageNumber {
			if chunks[i-1].ContentType == "image" && chunks[i].ContentType == "text" {
				t.Errorf("ChunkPDF() on page %d, image chunk comes before text chunk",
					chunks[i].PageNumber)
			}
		}
	}
}

func TestMultimodalChunker_ChunkPDF_ImageOnly(t *testing.T) {
	pdfChunker := NewPDFChunker(ChunkerConfig{
		ChunkSize:   300,
		OverlapSize: 10,
	})
	imageExtractor := NewImageExtractor(1024, 1024)
	chunker := NewMultimodalChunker(pdfChunker, imageExtractor)

	// Empty pages (image-only PDF)
	pages := []PDFPage{
		{PageNumber: 1, Text: ""},
		{PageNumber: 2, Text: ""},
	}

	images := []PDFImage{
		{
			Data:       []byte("image1"),
			Format:     "jpeg",
			Width:      800,
			Height:     600,
			PageNumber: 1,
			Base64:     "aW1hZ2Ux",
		},
		{
			Data:       []byte("image2"),
			Format:     "png",
			Width:      1200,
			Height:     900,
			PageNumber: 2,
			Base64:     "aW1hZ2Uy",
		},
	}

	metadata := &PDFMetadata{
		PageCount: 2,
		Title:     "Image PDF",
	}

	chunks, err := chunker.ChunkPDF(pages, images, metadata, "test.pdf")
	if err != nil {
		t.Fatalf("ChunkPDF() error = %v", err)
	}

	// Should have at least the image chunks
	if len(chunks) < len(images) {
		t.Errorf("ChunkPDF() returned %d chunks, want at least %d", len(chunks), len(images))
	}

	// Verify all images are present
	imageChunks := 0
	for _, chunk := range chunks {
		if chunk.ContentType == "image" {
			imageChunks++
		}
	}

	if imageChunks != len(images) {
		t.Errorf("ChunkPDF() returned %d image chunks, want %d", imageChunks, len(images))
	}
}

func TestMultimodalChunker_ChunkPDF_Sequencing(t *testing.T) {
	pdfChunker := NewPDFChunker(ChunkerConfig{
		ChunkSize:   100, // Small chunks to create multiple text chunks
		OverlapSize: 10,
	})
	imageExtractor := NewImageExtractor(1024, 1024)
	chunker := NewMultimodalChunker(pdfChunker, imageExtractor)

	// Create a page with enough text to create multiple chunks
	longText := ""
	for i := 0; i < 500; i++ {
		longText += "word "
	}

	pages := []PDFPage{
		{PageNumber: 1, Text: longText},
	}

	images := []PDFImage{
		{
			Data:       []byte("image"),
			Format:     "jpeg",
			Width:      800,
			Height:     600,
			PageNumber: 1,
			Base64:     "aW1hZ2U=",
		},
	}

	metadata := &PDFMetadata{
		PageCount: 1,
		Title:     "Test PDF",
	}

	chunks, err := chunker.ChunkPDF(pages, images, metadata, "test.pdf")
	if err != nil {
		t.Fatalf("ChunkPDF() error = %v", err)
	}

	// Verify chunk indices are sequential
	for i, chunk := range chunks {
		if chunk.ChunkIndex != i {
			t.Errorf("ChunkPDF() chunk %d ChunkIndex = %d, want %d", i, chunk.ChunkIndex, i)
		}
		if chunk.TotalChunks != len(chunks) {
			t.Errorf("ChunkPDF() chunk %d TotalChunks = %d, want %d", i, chunk.TotalChunks, len(chunks))
		}
	}
}
