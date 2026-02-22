package extraction

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"image"
	_ "image/jpeg"
	_ "image/png"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/pdfcpu/pdfcpu/pkg/api"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/model"
)

// PDFImage represents an extracted image from a PDF
type PDFImage struct {
	Data       []byte // Image data (raw bytes)
	Format     string // Image format: "jpeg", "png", etc.
	Width      int    // Image width in pixels
	Height     int    // Image height in pixels
	PageNumber int    // Page number where image was found
	Base64     string // Base64-encoded image data
}

// Image size thresholds for filtering (shared with chunker)
const (
	MinImageWidth  = 500
	MinImageHeight = 350
	MinImagePixels = 175_000 // ~1/3 of A4 page minimum
	MaxImageWidth  = 4000
	MaxImageHeight = 3000
	MaxImagePixels = 12_000_000 // 12 million pixels
)

// ImageExtractor handles extraction of images from PDFs
type ImageExtractor struct {
	maxWidth  int
	maxHeight int
}

// NewImageExtractor creates a new image extractor
// maxWidth and maxHeight specify maximum dimensions for image compression (0 = no limit)
func NewImageExtractor(maxWidth, maxHeight int) *ImageExtractor {
	if maxWidth <= 0 {
		maxWidth = 1024
	}
	if maxHeight <= 0 {
		maxHeight = 1024
	}
	return &ImageExtractor{
		maxWidth:  maxWidth,
		maxHeight: maxHeight,
	}
}

// ExtractImages extracts all images from a PDF file
// This method extracts all images at once - page numbers are extracted from filenames
func (e *ImageExtractor) ExtractImages(pdfPath string) ([]PDFImage, error) {
	// Create temporary directory for extracted images
	tmpDir, err := os.MkdirTemp("", "qwelli-images-*")
	if err != nil {
		return nil, fmt.Errorf("failed to create temp directory: %w", err)
	}
	defer os.RemoveAll(tmpDir) // Clean up temp directory

	// Extract images using pdfcpu
	conf := model.NewDefaultConfiguration()
	err = api.ExtractImagesFile(pdfPath, tmpDir, nil, conf)
	if err != nil {
		// If extraction fails, return empty slice (graceful degradation)
		// This allows text-only processing to continue
		return []PDFImage{}, nil
	}

	// Read extracted image files
	entries, err := os.ReadDir(tmpDir)
	if err != nil {
		return nil, fmt.Errorf("failed to read temp directory: %w", err)
	}

	if len(entries) == 0 {
		return []PDFImage{}, nil
	}

	var images []PDFImage
	skippedSmall := 0
	skippedLarge := 0

	for _, entry := range entries {
		if entry.IsDir() {
			continue // Skip directories
		}

		imagePath := filepath.Join(tmpDir, entry.Name())
		imageData, err := os.ReadFile(imagePath)
		if err != nil {
			continue // Skip unreadable files
		}

		// Parse image to get dimensions and format
		img, format, err := e.parseImage(imageData)
		if err != nil {
			continue // Skip unparseable images
		}

		// Get dimensions
		bounds := img.Bounds()
		width := bounds.Dx()
		height := bounds.Dy()
		pixels := width * height

		// Pre-filter: skip images that are too small BEFORE base64 encoding (expensive)
		if width < MinImageWidth || height < MinImageHeight || pixels < MinImagePixels {
			skippedSmall++
			continue // Skip images below minimum dimensions
		}

		// Pre-filter: skip images that are too large
		if width > MaxImageWidth || height > MaxImageHeight || pixels > MaxImagePixels {
			skippedLarge++
			continue // Skip images above maximum dimensions
		}

		// Compress image if needed
		compressedData, err := e.compressImage(imageData, format, width, height)
		if err != nil {
			compressedData = imageData
		}

		// Extract page number from filename (pdfcpu format: page_X_img_Y.ext)
		pageNumber := e.extractPageNumber(entry.Name())

		// Now encode to base64 (only for images that passed filters)
		base64Data := base64.StdEncoding.EncodeToString(compressedData)

		images = append(images, PDFImage{
			Data:       compressedData,
			Format:     format,
			Width:      width,
			Height:     height,
			PageNumber: pageNumber,
			Base64:     base64Data,
		})
	}

	if skippedSmall > 0 || skippedLarge > 0 {
		log.Printf("  Pre-filtered images: %d too small, %d too large", skippedSmall, skippedLarge)
	}

	return images, nil
}

// ExtractImagesByPage extracts images from a specific page of a PDF
// This ensures correct page number assignment
func (e *ImageExtractor) ExtractImagesByPage(pdfPath string, pageNumber int) ([]PDFImage, error) {
	// Create temporary directory for extracted images
	tmpDir, err := os.MkdirTemp("", fmt.Sprintf("qwelli-images-page%d-*", pageNumber))
	if err != nil {
		return nil, fmt.Errorf("failed to create temp directory: %w", err)
	}
	defer os.RemoveAll(tmpDir) // Clean up temp directory

	// Extract images from specific page using pdfcpu
	conf := model.NewDefaultConfiguration()
	// Create page selection for this specific page (pdfcpu uses 1-based indexing)
	pageSelection := []string{fmt.Sprintf("%d", pageNumber)}
	err = api.ExtractImagesFile(pdfPath, tmpDir, pageSelection, conf)
	if err != nil {
		// If extraction fails for this page, return empty slice (graceful degradation)
		return []PDFImage{}, nil
	}

	// Read extracted image files
	entries, err := os.ReadDir(tmpDir)
	if err != nil {
		return nil, fmt.Errorf("failed to read temp directory: %w", err)
	}

	if len(entries) == 0 {
		return []PDFImage{}, nil
	}

	var images []PDFImage
	for _, entry := range entries {
		if entry.IsDir() {
			continue // Skip directories
		}

		imagePath := filepath.Join(tmpDir, entry.Name())
		imageData, err := os.ReadFile(imagePath)
		if err != nil {
			continue // Skip unreadable files
		}

		// Parse image to get dimensions and format
		img, format, err := e.parseImage(imageData)
		if err != nil {
			continue // Skip unparseable images
		}

		// Get dimensions
		bounds := img.Bounds()
		width := bounds.Dx()
		height := bounds.Dy()
		pixels := width * height

		// Pre-filter: skip images that are too small BEFORE base64 encoding
		if width < MinImageWidth || height < MinImageHeight || pixels < MinImagePixels {
			continue // Skip images below minimum dimensions
		}

		// Pre-filter: skip images that are too large
		if width > MaxImageWidth || height > MaxImageHeight || pixels > MaxImagePixels {
			continue // Skip images above maximum dimensions
		}

		// Compress image if needed
		compressedData, err := e.compressImage(imageData, format, width, height)
		if err != nil {
			compressedData = imageData
		}

		// Now encode to base64 (only for images that passed filters)
		base64Data := base64.StdEncoding.EncodeToString(compressedData)

		// Assign the known page number (from the function parameter)
		images = append(images, PDFImage{
			Data:       compressedData,
			Format:     format,
			Width:      width,
			Height:     height,
			PageNumber: pageNumber,
			Base64:     base64Data,
		})
	}

	return images, nil
}

// parseImage parses image data to get format and dimensions
func (e *ImageExtractor) parseImage(data []byte) (image.Image, string, error) {
	img, format, err := image.Decode(bytes.NewReader(data))
	if err != nil {
		return nil, "", fmt.Errorf("failed to decode image: %w", err)
	}
	return img, format, nil
}

// compressImage compresses an image if it exceeds max dimensions
// For now, we'll just resize if needed (simple implementation)
func (e *ImageExtractor) compressImage(data []byte, _ string, width, height int) ([]byte, error) {
	// If image is already within limits, return as-is
	if width <= e.maxWidth && height <= e.maxHeight {
		return data, nil
	}

	// For now, we'll return the original data
	// A full implementation would resize the image using an image processing library
	// This is a placeholder - in production, you'd use something like:
	// - github.com/disintegration/imaging
	// - github.com/nfnt/resize
	// For now, we'll just return the original data and let the API handle it
	return data, nil
}

// extractPageNumber extracts page number from pdfcpu filename format
// Format: "page_X_img_Y.ext" or similar variations
func (e *ImageExtractor) extractPageNumber(filename string) int {
	// pdfcpu typically names files like: "page_1_img_0.jpg"
	// Try to extract page number
	parts := strings.Split(filename, "_")
	for i, part := range parts {
		if part == "page" && i+1 < len(parts) {
			var pageNum int
			if _, err := fmt.Sscanf(parts[i+1], "%d", &pageNum); err == nil {
				return pageNum
			}
		}
	}
	// If we can't extract, return 1 as default
	return 1
}

// GetImageBase64 returns base64-encoded image data for embedding API
func (img *PDFImage) GetImageBase64() string {
	if img.Base64 != "" {
		return img.Base64
	}
	// Encode if not already encoded
	img.Base64 = base64.StdEncoding.EncodeToString(img.Data)
	return img.Base64
}
