package processor

import (
	"bytes"
	"encoding/base64"
	"image"
	"image/png"
	"testing"
)

func TestExtractPageNumber_EdgeCases(t *testing.T) {
	extractor := NewImageExtractor(1024, 1024)

	tests := []struct {
		name     string
		filename string
		want     int
	}{
		{
			name:     "empty filename",
			filename: "",
			want:     1,
		},
		{
			name:     "no underscore",
			filename: "image.jpg",
			want:     1,
		},
		{
			name:     "page at end",
			filename: "something_page_3",
			want:     3,
		},
		{
			name:     "multiple page keywords",
			filename: "page_2_page_5.jpg",
			want:     2, // Should take first one
		},
		{
			name:     "non-numeric after page",
			filename: "page_abc_img_0.jpg",
			want:     1, // Default when parsing fails
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := extractor.extractPageNumber(tt.filename)
			if got != tt.want {
				t.Errorf("extractPageNumber(%q) = %d, want %d", tt.filename, got, tt.want)
			}
		})
	}
}

func TestCompressImage_EdgeCases(t *testing.T) {
	extractor := NewImageExtractor(1024, 1024)

	// Create test image
	img := image.NewRGBA(image.Rect(0, 0, 100, 100))
	var buf bytes.Buffer
	png.Encode(&buf, img)
	imageData := buf.Bytes()

	tests := []struct {
		name     string
		width    int
		height   int
		format   string
		wantSame bool
	}{
		{
			name:     "exact max dimensions",
			width:    1024,
			height:   1024,
			format:   "png",
			wantSame: true,
		},
		{
			name:     "one pixel over width",
			width:    1025,
			height:   1024,
			format:   "png",
			wantSame: false,
		},
		{
			name:     "one pixel over height",
			width:    1024,
			height:   1025,
			format:   "png",
			wantSame: false,
		},
		{
			name:     "zero dimensions",
			width:    0,
			height:   0,
			format:   "png",
			wantSame: true,
		},
		{
			name:     "very large image",
			width:    10000,
			height:   10000,
			format:   "jpeg",
			wantSame: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := extractor.compressImage(imageData, tt.format, tt.width, tt.height)
			if err != nil {
				t.Errorf("compressImage() error = %v", err)
				return
			}

			if tt.wantSame && len(result) != len(imageData) {
				t.Errorf("compressImage() returned different length, want same")
			}
		})
	}
}

func TestPDFImage_GetImageBase64_MultipleCalls(t *testing.T) {
	testData := []byte("test image data")
	expectedBase64 := base64.StdEncoding.EncodeToString(testData)

	img := PDFImage{
		Data:   testData,
		Base64: "",
	}

	// First call should encode
	first := img.GetImageBase64()
	if first != expectedBase64 {
		t.Errorf("GetImageBase64() first call = %q, want %q", first, expectedBase64)
	}

	// Second call should return cached
	second := img.GetImageBase64()
	if second != expectedBase64 {
		t.Errorf("GetImageBase64() second call = %q, want %q", second, expectedBase64)
	}

	// Should be same
	if first != second {
		t.Error("GetImageBase64() returned different values on multiple calls")
	}
}

func TestParseImage_InvalidData(t *testing.T) {
	extractor := NewImageExtractor(1024, 1024)

	invalidData := []byte("not an image")

	_, _, err := extractor.parseImage(invalidData)
	if err == nil {
		t.Error("parseImage() with invalid data expected error, got nil")
	}
}

func TestParseImage_DifferentFormats(t *testing.T) {
	extractor := NewImageExtractor(1024, 1024)

	// Test PNG
	pngImg := image.NewRGBA(image.Rect(0, 0, 50, 50))
	var pngBuf bytes.Buffer
	png.Encode(&pngBuf, pngImg)

	_, format, err := extractor.parseImage(pngBuf.Bytes())
	if err != nil {
		t.Fatalf("parseImage() PNG error = %v", err)
	}
	if format != "png" {
		t.Errorf("parseImage() PNG format = %q, want %q", format, "png")
	}
}



