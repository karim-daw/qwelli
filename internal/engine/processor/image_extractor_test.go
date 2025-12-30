package processor

import (
	"os"
	"path/filepath"
	"testing"
)

func TestNewImageExtractor(t *testing.T) {
	extractor, err := NewImageExtractor(1024, 1024, true)
	if err != nil {
		t.Fatal("NewImageExtractor returned error:", err)
	}
	if extractor == nil {
		t.Fatal("NewImageExtractor returned nil")
	}
	if extractor.maxWidth != 1024 {
		t.Errorf("Expected maxWidth 1024, got %d", extractor.maxWidth)
	}
}

func TestExtractImages_WithSamplePDF(t *testing.T) {
	wd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get working directory: %v", err)
	}

	possiblePaths := []string{
		filepath.Join(wd, "..", "..", "..", "tests", "demo", "pdf_samples", "simple.pdf"),
		filepath.Join(wd, "..", "..", "..", "tests", "demo", "pdf_samples", "BillFile5086630.pdf"),
	}

	var path string
	for _, p := range possiblePaths {
		absPath, err := filepath.Abs(p)
		if err != nil {
			continue
		}
		if _, err := os.Stat(absPath); err == nil {
			path = absPath
			break
		}
	}

	if path == "" {
		t.Skip("Sample PDF not found")
	}

	extractor, err := NewImageExtractor(1024, 1024, true)
	if err != nil {
		t.Fatalf("Failed to create extractor: %v", err)
	}

	images, err := extractor.ExtractImages(path)
	if err != nil {
		t.Fatalf("ExtractImages failed: %v", err)
	}

	t.Logf("Extracted %d images from PDF", len(images))

	for i, img := range images {
		t.Logf("Image %d: %dx%d, format: %s, page: %d, data size: %d bytes",
			i, img.Width, img.Height, img.Format, img.PageNumber, len(img.Data))

		if img.Base64 == "" {
			t.Errorf("Image %d has empty Base64 data", i)
		}
		if img.Width <= 0 || img.Height <= 0 {
			t.Errorf("Image %d has invalid dimensions: %dx%d", i, img.Width, img.Height)
		}
	}
}

func TestExtractImagesFromPage(t *testing.T) {
	wd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get working directory: %v", err)
	}

	path := filepath.Join(wd, "..", "..", "..", "tests", "demo", "pdf_samples", "simple.pdf")
	absPath, err := filepath.Abs(path)
	if err != nil {
		t.Skip("Could not resolve path")
	}

	if _, err := os.Stat(absPath); err != nil {
		t.Skip("Sample PDF not found")
	}

	extractor, err := NewImageExtractor(1024, 1024, false)
	if err != nil {
		t.Fatalf("Failed to create extractor: %v", err)
	}

	images, err := extractor.ExtractImagesFromPage(absPath, 1)
	if err != nil {
		t.Fatalf("ExtractImagesFromPage failed: %v", err)
	}

	t.Logf("Extracted %d images from page 1", len(images))
}

func TestGetImageDataURL(t *testing.T) {
	img := PDFImage{
		Data:       []byte("test data"),
		Format:     "jpeg",
		Width:      100,
		Height:     100,
		PageNumber: 1,
	}

	dataURL := img.GetImageDataURL()
	if dataURL == "" {
		t.Error("GetImageDataURL returned empty string")
	}

	if len(dataURL) < 20 {
		t.Errorf("GetImageDataURL returned suspiciously short string: %s", dataURL)
	}
}

func BenchmarkExtractImages(b *testing.B) {
	wd, err := os.Getwd()
	if err != nil {
		b.Fatalf("Failed to get working directory: %v", err)
	}

	path := filepath.Join(wd, "..", "..", "..", "tests", "demo", "pdf_samples", "simple.pdf")
	absPath, err := filepath.Abs(path)
	if err != nil {
		b.Skip("Could not resolve path")
	}

	if _, err := os.Stat(absPath); err != nil {
		b.Skip("Sample PDF not found")
	}

	extractor, err := NewImageExtractor(1024, 1024, false)
	if err != nil {
		b.Fatalf("Failed to create extractor: %v", err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := extractor.ExtractImages(absPath)
		if err != nil {
			b.Fatalf("ExtractImages failed: %v", err)
		}
	}
}
