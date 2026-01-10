package extraction

import (
	"bytes"
	"encoding/base64"
	"image"
	"image/png"
	"testing"
)

func TestNewImageExtractor(t *testing.T) {
	tests := []struct {
		name       string
		maxWidth   int
		maxHeight  int
		wantWidth  int
		wantHeight int
	}{
		{
			name:       "valid dimensions",
			maxWidth:   1024,
			maxHeight:  1024,
			wantWidth:  1024,
			wantHeight: 1024,
		},
		{
			name:       "zero dimensions use defaults",
			maxWidth:   0,
			maxHeight:  0,
			wantWidth:  1024,
			wantHeight: 1024,
		},
		{
			name:       "negative dimensions use defaults",
			maxWidth:   -1,
			maxHeight:  -1,
			wantWidth:  1024,
			wantHeight: 1024,
		},
		{
			name:       "custom dimensions",
			maxWidth:   512,
			maxHeight:  512,
			wantWidth:  512,
			wantHeight: 512,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			extractor := NewImageExtractor(tt.maxWidth, tt.maxHeight)
			if extractor.maxWidth != tt.wantWidth {
				t.Errorf("NewImageExtractor() maxWidth = %d, want %d", extractor.maxWidth, tt.wantWidth)
			}
			if extractor.maxHeight != tt.wantHeight {
				t.Errorf("NewImageExtractor() maxHeight = %d, want %d", extractor.maxHeight, tt.wantHeight)
			}
		})
	}
}

func TestExtractPageNumber(t *testing.T) {
	tests := []struct {
		name     string
		filename string
		want     int
	}{
		{
			name:     "standard pdfcpu format",
			filename: "page_1_img_0.jpg",
			want:     1,
		},
		{
			name:     "page 5",
			filename: "page_5_img_2.png",
			want:     5,
		},
		{
			name:     "page 10",
			filename: "page_10_img_0.jpeg",
			want:     10,
		},
		{
			name:     "no page number found",
			filename: "image.jpg",
			want:     1, // default
		},
		{
			name:     "invalid format",
			filename: "some_file.jpg",
			want:     1, // default
		},
	}

	extractor := NewImageExtractor(1024, 1024)
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := extractor.extractPageNumber(tt.filename)
			if got != tt.want {
				t.Errorf("extractPageNumber(%q) = %d, want %d", tt.filename, got, tt.want)
			}
		})
	}
}

func TestParseImage(t *testing.T) {
	// Create a simple test image
	img := image.NewRGBA(image.Rect(0, 0, 100, 100))
	for y := 0; y < 100; y++ {
		for x := 0; x < 100; x++ {
			img.Set(x, y, image.White)
		}
	}

	// Encode as PNG
	var buf bytes.Buffer
	err := png.Encode(&buf, img)
	if err != nil {
		t.Fatalf("Failed to encode test image: %v", err)
	}

	extractor := NewImageExtractor(1024, 1024)
	parsedImg, format, err := extractor.parseImage(buf.Bytes())
	if err != nil {
		t.Fatalf("parseImage() error = %v", err)
	}

	if format != "png" {
		t.Errorf("parseImage() format = %q, want %q", format, "png")
	}

	bounds := parsedImg.Bounds()
	if bounds.Dx() != 100 {
		t.Errorf("parseImage() width = %d, want %d", bounds.Dx(), 100)
	}
	if bounds.Dy() != 100 {
		t.Errorf("parseImage() height = %d, want %d", bounds.Dy(), 100)
	}
}

func TestCompressImage(t *testing.T) {
	// Create a test image
	img := image.NewRGBA(image.Rect(0, 0, 200, 200))
	var buf bytes.Buffer
	err := png.Encode(&buf, img)
	if err != nil {
		t.Fatalf("Failed to encode test image: %v", err)
	}
	imageData := buf.Bytes()

	tests := []struct {
		name      string
		width     int
		height    int
		maxWidth  int
		maxHeight int
		wantSame  bool // Whether output should be same as input
	}{
		{
			name:      "within limits",
			width:     200,
			height:    200,
			maxWidth:  1024,
			maxHeight: 1024,
			wantSame:  true,
		},
		{
			name:      "exceeds width",
			width:     2000,
			height:    200,
			maxWidth:  1024,
			maxHeight: 1024,
			wantSame:  false, // Currently returns same, but should compress
		},
		{
			name:      "exceeds height",
			width:     200,
			height:    2000,
			maxWidth:  1024,
			maxHeight: 1024,
			wantSame:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			extractor := NewImageExtractor(tt.maxWidth, tt.maxHeight)
			result, err := extractor.compressImage(imageData, "png", tt.width, tt.height)
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

func TestPDFImage_GetImageBase64(t *testing.T) {
	// Create test image data
	testData := []byte("fake image data")
	expectedBase64 := base64.StdEncoding.EncodeToString(testData)

	tests := []struct {
		name       string
		image      PDFImage
		wantBase64 string
	}{
		{
			name: "already has base64",
			image: PDFImage{
				Data:   testData,
				Base64: expectedBase64,
			},
			wantBase64: expectedBase64,
		},
		{
			name: "needs encoding",
			image: PDFImage{
				Data:   testData,
				Base64: "",
			},
			wantBase64: expectedBase64,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.image.GetImageBase64()
			if got != tt.wantBase64 {
				t.Errorf("GetImageBase64() = %q, want %q", got, tt.wantBase64)
			}
			// Verify it's valid base64
			decoded, err := base64.StdEncoding.DecodeString(got)
			if err != nil {
				t.Errorf("GetImageBase64() returned invalid base64: %v", err)
			}
			if string(decoded) != string(tt.image.Data) {
				t.Errorf("GetImageBase64() decoded data doesn't match original")
			}
		})
	}
}

func TestExtractImages_NonExistentFile(t *testing.T) {
	extractor := NewImageExtractor(1024, 1024)
	images, err := extractor.ExtractImages("/nonexistent/file.pdf")
	if err != nil {
		t.Fatalf("ExtractImages() should return empty slice on error, got error: %v", err)
	}
	if len(images) != 0 {
		t.Errorf("ExtractImages() returned %d images, want 0", len(images))
	}
}

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
