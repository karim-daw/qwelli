package processor

import (
	"testing"
)

// TestMultimodalChunker_ImageDimensionFiltering verifies that images are filtered
// based on minimum and maximum dimensions
func TestMultimodalChunker_ImageDimensionFiltering(t *testing.T) {
	pdfChunker := NewPDFChunker(ChunkerConfig{
		ChunkSize:   300,
		OverlapSize: 10,
	})
	imageExtractor := NewImageExtractor(1024, 1024)
	chunker := NewMultimodalChunker(pdfChunker, imageExtractor)

	pages := []PDFPage{
		{PageNumber: 1, Text: "Page one text."},
	}

	// Test various image sizes
	images := []PDFImage{
		// Too small (below 500x350, 175k pixels)
		{
			Data:       []byte("tiny"),
			Format:     "jpeg",
			Width:      100,
			Height:     100,
			PageNumber: 1,
			Base64:     "dGlueQ==",
		},
		// Too small (meets width but not height)
		{
			Data:       []byte("small"),
			Format:     "jpeg",
			Width:      600,
			Height:     200, // Below 350
			PageNumber: 1,
			Base64:     "c21hbGw=",
		},
		// Valid size (500x350 = 175k pixels, exactly at minimum)
		{
			Data:       []byte("valid1"),
			Format:     "jpeg",
			Width:      500,
			Height:     350,
			PageNumber: 1,
			Base64:     "dmFsaWQx",
		},
		// Valid size (800x600 = 480k pixels)
		{
			Data:       []byte("valid2"),
			Format:     "jpeg",
			Width:      800,
			Height:     600,
			PageNumber: 1,
			Base64:     "dmFsaWQy",
		},
		// Too large (exceeds 4000x3000, 12M pixels)
		{
			Data:       []byte("huge"),
			Format:     "jpeg",
			Width:      5000,
			Height:     4000,
			PageNumber: 1,
			Base64:     "aHVnZQ==",
		},
		// Too large (exceeds max pixels)
		{
			Data:       []byte("huge2"),
			Format:     "jpeg",
			Width:      4000,
			Height:     4000, // 16M pixels, exceeds 12M limit
			PageNumber: 1,
			Base64:     "aHVnZTI=",
		},
		// Valid size (at maximum: 4000x3000 = 12M pixels)
		{
			Data:       []byte("max"),
			Format:     "jpeg",
			Width:      4000,
			Height:     3000,
			PageNumber: 1,
			Base64:     "bWF4",
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

	// Count image chunks that passed filtering
	imageChunks := 0
	for _, chunk := range chunks {
		if chunk.ContentType == "image" {
			imageChunks++
			// Verify dimensions are within valid range
			if chunk.ImageWidth < 500 || chunk.ImageHeight < 350 {
				t.Errorf("Image chunk passed filter but has invalid dimensions: %dx%d", chunk.ImageWidth, chunk.ImageHeight)
			}
			if chunk.ImageWidth > 4000 || chunk.ImageHeight > 3000 {
				t.Errorf("Image chunk passed filter but exceeds max dimensions: %dx%d", chunk.ImageWidth, chunk.ImageHeight)
			}
			pixelCount := chunk.ImageWidth * chunk.ImageHeight
			if pixelCount < 175_000 {
				t.Errorf("Image chunk passed filter but has too few pixels: %d", pixelCount)
			}
			if pixelCount > 12_000_000 {
				t.Errorf("Image chunk passed filter but has too many pixels: %d", pixelCount)
			}
		}
	}

	// Should have exactly 3 valid images (valid1, valid2, max)
	expectedValidImages := 3
	if imageChunks != expectedValidImages {
		t.Errorf("ChunkPDF() returned %d image chunks, want %d (should filter out tiny and huge images)", imageChunks, expectedValidImages)
	}
}

// TestMultimodalChunker_ImageDimensionFiltering_EdgeCases tests edge cases
func TestMultimodalChunker_ImageDimensionFiltering_EdgeCases(t *testing.T) {
	pdfChunker := NewPDFChunker(ChunkerConfig{
		ChunkSize:   300,
		OverlapSize: 10,
	})
	imageExtractor := NewImageExtractor(1024, 1024)
	chunker := NewMultimodalChunker(pdfChunker, imageExtractor)

	pages := []PDFPage{
		{PageNumber: 1, Text: "Page one text."},
	}

	images := []PDFImage{
		// Exactly at minimum width but below minimum height
		{
			Data:       []byte("edge1"),
			Format:     "jpeg",
			Width:      500,
			Height:     349, // 1 pixel below minimum
			PageNumber: 1,
			Base64:     "ZWRnZTE=",
		},
		// Exactly at minimum height but below minimum width
		{
			Data:       []byte("edge2"),
			Format:     "jpeg",
			Width:      499, // 1 pixel below minimum
			Height:     350,
			PageNumber: 1,
			Base64:     "ZWRnZTI=",
		},
		// Exactly at minimum pixels but wrong aspect ratio
		{
			Data:       []byte("edge3"),
			Format:     "jpeg",
			Width:      1000,
			Height:     175, // Exactly 175k pixels but height too small
			PageNumber: 1,
			Base64:     "ZWRnZTM=",
		},
		// Valid: exactly at all minimums
		{
			Data:       []byte("edge4"),
			Format:     "jpeg",
			Width:      500,
			Height:     350,
			PageNumber: 1,
			Base64:     "ZWRnZTQ=",
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

	// Should have exactly 1 valid image (edge4)
	imageChunks := 0
	for _, chunk := range chunks {
		if chunk.ContentType == "image" {
			imageChunks++
		}
	}

	if imageChunks != 1 {
		t.Errorf("ChunkPDF() returned %d image chunks, want 1 (edge cases should be filtered)", imageChunks)
	}
}
