package chunker

import (
	"testing"

	"github.com/karim-daw/qwelli/internal/engine/processor"
)

func TestNewMultimodalChunker(t *testing.T) {
	pdfChunker := NewChunker(ChunkerConfig{
		ChunkSize:   300,
		OverlapSize: 10,
	})
	imageExtractor, err := processor.NewImageExtractor(1024, 1024, false)
	if err != nil {
		t.Fatalf("Failed to create image extractor: %v", err)
	}

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
	pdfChunker := NewChunker(ChunkerConfig{
		ChunkSize:   300,
		OverlapSize: 10,
	})
	imageExtractor, err := processor.NewImageExtractor(1024, 1024, false)
	if err != nil {
		t.Fatalf("Failed to create image extractor: %v", err)
	}
	chunker := NewMultimodalChunker(pdfChunker, imageExtractor)

	pages := []processor.PDFPage{
		{PageNumber: 1, Text: "This is page one content."},
		{PageNumber: 2, Text: "This is page two content."},
	}
	images := []processor.PDFImage{} // No images
	metadata := &processor.PDFMetadata{
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
		pageNum := 0
		if len(chunk.PageNumbers) > 0 {
			pageNum = chunk.PageNumbers[0]
		}
		if pageNum < 1 || pageNum > 2 {
			t.Errorf("ChunkPDF() chunk %d PageNumber = %d, want 1 or 2", i, pageNum)
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
	pdfChunker := NewChunker(ChunkerConfig{
		ChunkSize:   300,
		OverlapSize: 10,
	})
	imageExtractor, err := processor.NewImageExtractor(1024, 1024, false)
	if err != nil {
		t.Fatalf("Failed to create image extractor: %v", err)
	}
	chunker := NewMultimodalChunker(pdfChunker, imageExtractor)

	pages := []processor.PDFPage{
		{PageNumber: 1, Text: "Page one text."},
		{PageNumber: 2, Text: "Page two text."},
	}

	// Create mock images
	images := []processor.PDFImage{
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

	metadata := &processor.PDFMetadata{
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
		pageI := 0
		pageJ := 0
		if len(chunks[i].PageNumbers) > 0 {
			pageI = chunks[i].PageNumbers[0]
		}
		if len(chunks[i-1].PageNumbers) > 0 {
			pageJ = chunks[i-1].PageNumbers[0]
		}
		if pageI < pageJ {
			t.Errorf("ChunkPDF() chunks not sorted: chunk %d page %d < chunk %d page %d",
				i, pageI, i-1, pageJ)
		}
		// On same page, text should come before images
		if pageI == pageJ {
			if chunks[i-1].ContentType == "image" && chunks[i].ContentType == "text" {
				t.Errorf("ChunkPDF() on page %d, image chunk comes before text chunk", pageI)
			}
		}
	}
}

func TestMultimodalChunker_ChunkPDF_ImageOnly(t *testing.T) {
	pdfChunker := NewChunker(ChunkerConfig{
		ChunkSize:   300,
		OverlapSize: 10,
	})
	imageExtractor, err := processor.NewImageExtractor(1024, 1024, false)
	if err != nil {
		t.Fatalf("Failed to create image extractor: %v", err)
	}
	chunker := NewMultimodalChunker(pdfChunker, imageExtractor)

	// Empty pages (image-only PDF)
	pages := []processor.PDFPage{
		{PageNumber: 1, Text: ""},
		{PageNumber: 2, Text: ""},
	}

	images := []processor.PDFImage{
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

	metadata := &processor.PDFMetadata{
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
	pdfChunker := NewChunker(ChunkerConfig{
		ChunkSize:   100, // Small chunks to create multiple text chunks
		OverlapSize: 10,
	})
	imageExtractor, err := processor.NewImageExtractor(1024, 1024, false)
	if err != nil {
		t.Fatalf("Failed to create image extractor: %v", err)
	}
	chunker := NewMultimodalChunker(pdfChunker, imageExtractor)

	// Create a page with enough text to create multiple chunks
	longText := ""
	for i := 0; i < 500; i++ {
		longText += "word "
	}

	pages := []processor.PDFPage{
		{PageNumber: 1, Text: longText},
	}

	images := []processor.PDFImage{
		{
			Data:       []byte("image"),
			Format:     "jpeg",
			Width:      800,
			Height:     600,
			PageNumber: 1,
			Base64:     "aW1hZ2U=",
		},
	}

	metadata := &processor.PDFMetadata{
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

// TestMultimodalChunker_ImageDimensionFiltering verifies that images are filtered
// based on minimum and maximum dimensions
func TestMultimodalChunker_ImageDimensionFiltering(t *testing.T) {
	pdfChunker := NewChunker(ChunkerConfig{
		ChunkSize:   300,
		OverlapSize: 10,
	})
	imageExtractor, err := processor.NewImageExtractor(1024, 1024, false)
	if err != nil {
		t.Fatalf("Failed to create image extractor: %v", err)
	}
	chunker := NewMultimodalChunker(pdfChunker, imageExtractor)

	pages := []processor.PDFPage{
		{PageNumber: 1, Text: "Page one text."},
	}

	// Test various image sizes
	images := []processor.PDFImage{
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

	metadata := &processor.PDFMetadata{
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
	pdfChunker := NewChunker(ChunkerConfig{
		ChunkSize:   300,
		OverlapSize: 10,
	})
	imageExtractor, err := processor.NewImageExtractor(1024, 1024, false)
	if err != nil {
		t.Fatalf("Failed to create image extractor: %v", err)
	}
	chunker := NewMultimodalChunker(pdfChunker, imageExtractor)

	pages := []processor.PDFPage{
		{PageNumber: 1, Text: "Page one text."},
	}

	images := []processor.PDFImage{
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

	metadata := &processor.PDFMetadata{
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

func TestMultimodalChunker_EmptyPagesAndImages(t *testing.T) {
	pdfChunker := NewChunker(ChunkerConfig{
		ChunkSize:   300,
		OverlapSize: 10,
	})
	imageExtractor, err := processor.NewImageExtractor(1024, 1024, false)
	if err != nil {
		t.Fatalf("Failed to create image extractor: %v", err)
	}
	chunker := NewMultimodalChunker(pdfChunker, imageExtractor)

	chunks, err := chunker.ChunkPDF([]processor.PDFPage{}, []processor.PDFImage{}, nil, "test.pdf")
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
	pdfChunker := NewChunker(ChunkerConfig{
		ChunkSize:   300,
		OverlapSize: 10,
	})
	imageExtractor, err := processor.NewImageExtractor(1024, 1024, false)
	if err != nil {
		t.Fatalf("Failed to create image extractor: %v", err)
	}
	chunker := NewMultimodalChunker(pdfChunker, imageExtractor)

	pages := []processor.PDFPage{
		{PageNumber: 1, Text: "Page with many images"},
	}

	// Create 5 images on the same page
	images := make([]processor.PDFImage, 5)
	for i := 0; i < 5; i++ {
		images[i] = processor.PDFImage{
			Data:       []byte("image data"),
			Format:     "jpeg",
			Width:      800,
			Height:     600,
			PageNumber: 1,
			Base64:     "base64data",
		}
	}

	metadata := &processor.PDFMetadata{
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
	pdfChunker := NewChunker(ChunkerConfig{
		ChunkSize:   300,
		OverlapSize: 10,
	})
	imageExtractor, err := processor.NewImageExtractor(1024, 1024, false)
	if err != nil {
		t.Fatalf("Failed to create image extractor: %v", err)
	}
	chunker := NewMultimodalChunker(pdfChunker, imageExtractor)

	pages := []processor.PDFPage{
		{PageNumber: 1, Text: "Test page"},
	}

	images := []processor.PDFImage{
		{
			Data:       []byte("image"),
			Format:     "png",
			Width:      1920,
			Height:     1080,
			PageNumber: 1,
			Base64:     "base64",
		},
	}

	metadata := &processor.PDFMetadata{
		PageCount: 1,
		Title:     "Test",
	}

	chunks, err := chunker.ChunkPDF(pages, images, metadata, "test.pdf")
	if err != nil {
		t.Fatalf("ChunkPDF() error = %v", err)
	}

	// Find image chunk
	var imageChunk *Chunk
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
	pdfChunker := NewChunker(ChunkerConfig{
		ChunkSize:   300,
		OverlapSize: 10,
	})
	imageExtractor, err := processor.NewImageExtractor(1024, 1024, false)
	if err != nil {
		t.Fatalf("Failed to create image extractor: %v", err)
	}
	chunker := NewMultimodalChunker(pdfChunker, imageExtractor)

	pages := []processor.PDFPage{
		{PageNumber: 1, Text: "Page 1"},
		{PageNumber: 2, Text: "Page 2"},
		{PageNumber: 3, Text: "Page 3"},
	}

	// Images on pages 2 and 3
	images := []processor.PDFImage{
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

	metadata := &processor.PDFMetadata{
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
		pageI := 0
		pageJ := 0
		if len(chunks[i].PageNumbers) > 0 {
			pageI = chunks[i].PageNumbers[0]
		}
		if len(chunks[i-1].PageNumbers) > 0 {
			pageJ = chunks[i-1].PageNumbers[0]
		}
		if pageI < pageJ {
			t.Errorf("ChunkPDF() chunks not sorted: chunk %d page %d < chunk %d page %d",
				i, pageI, i-1, pageJ)
		}
	}

	// On page 2, text should come before image
	foundPage2Text := false
	for _, chunk := range chunks {
		pageNum := 0
		if len(chunk.PageNumbers) > 0 {
			pageNum = chunk.PageNumbers[0]
		}
		if pageNum == 2 {
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
