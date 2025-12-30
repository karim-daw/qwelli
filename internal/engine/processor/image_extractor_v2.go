package processor

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"image/jpeg"
	"os"

	"github.com/unidoc/unipdf/v4/extractor"
	"github.com/unidoc/unipdf/v4/model"
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

// IImageExtractor defines the interface for image extraction
type IImageExtractor interface {
	ExtractImages(pdfPath string) ([]PDFImage, error)
	ExtractImagesFromPage(pdfPath string, pageNum int) ([]PDFImage, error)
}

// ImageExtractor handles extraction of images from PDFs using UniPDF
type ImageExtractor struct {
	maxWidth  int
	maxHeight int
	verbose   bool
}

// Compile-time check that ImageExtractor implements IImageExtractor
var _ IImageExtractor = (*ImageExtractor)(nil)

// NewImageExtractor creates a new image extractor
// maxWidth and maxHeight specify maximum dimensions for image compression (0 = no limit)
func NewImageExtractor(maxWidth, maxHeight int, verbose bool) (*ImageExtractor, error) {
	// Check if license initialization failed
	if licenseInitErr != nil {
		return nil, licenseInitErr
	}

	if maxWidth <= 0 {
		maxWidth = 1024
	}
	if maxHeight <= 0 {
		maxHeight = 1024
	}
	return &ImageExtractor{
		maxWidth:  maxWidth,
		maxHeight: maxHeight,
		verbose:   verbose,
	}, nil
}

// ExtractImages extracts all images from a PDF file
func (e *ImageExtractor) ExtractImages(pdfPath string) ([]PDFImage, error) {
	f, err := os.Open(pdfPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open PDF file: %w", err)
	}
	defer f.Close()

	pdfReader, err := model.NewPdfReader(f)
	if err != nil {
		return nil, fmt.Errorf("failed to create PDF reader: %w", err)
	}

	numPages, err := pdfReader.GetNumPages()
	if err != nil {
		return nil, fmt.Errorf("failed to get page count: %w", err)
	}

	if e.verbose {
		fmt.Printf("Extracting images from %d pages in: %s\n", numPages, pdfPath)
	}

	var allImages []PDFImage
	for i := 1; i <= numPages; i++ {
		images, err := e.extractImagesFromPageReader(pdfReader, i)
		if err != nil {
			// Log error but continue with other pages
			if e.verbose {
				fmt.Printf("Warning: failed to extract images from page %d: %v\n", i, err)
			}
			continue
		}
		allImages = append(allImages, images...)
	}

	return allImages, nil
}

// ExtractImagesFromPage extracts images from a specific page
func (e *ImageExtractor) ExtractImagesFromPage(pdfPath string, pageNum int) ([]PDFImage, error) {
	f, err := os.Open(pdfPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open PDF file: %w", err)
	}
	defer f.Close()

	pdfReader, err := model.NewPdfReader(f)
	if err != nil {
		return nil, fmt.Errorf("failed to create PDF reader: %w", err)
	}

	return e.extractImagesFromPageReader(pdfReader, pageNum)
}

// extractImagesFromPageReader extracts images from a specific page using a reader
func (e *ImageExtractor) extractImagesFromPageReader(pdfReader *model.PdfReader, pageNum int) ([]PDFImage, error) {
	page, err := pdfReader.GetPage(pageNum)
	if err != nil {
		return nil, fmt.Errorf("failed to get page %d: %w", pageNum, err)
	}

	// Extract images using UniPDF extractor
	ex, err := extractor.New(page)
	if err != nil {
		return nil, fmt.Errorf("failed to create extractor for page %d: %w", pageNum, err)
	}

	pdfImages, err := ex.ExtractPageImages(nil)
	if err != nil {
		return nil, fmt.Errorf("failed to extract images from page %d: %w", pageNum, err)
	}

	if e.verbose {
		fmt.Printf("Page %d: found %d images\n", pageNum, len(pdfImages.Images))
	}

	var images []PDFImage
	for imgIdx, img := range pdfImages.Images {
		pdfImage, err := e.processImage(img.Image, pageNum, imgIdx)
		if err != nil {
			if e.verbose {
				fmt.Printf("Warning: failed to process image %d on page %d: %v\n", imgIdx, pageNum, err)
			}
			continue
		}
		images = append(images, pdfImage)
	}

	return images, nil
}

// processImage converts a UniPDF image to our PDFImage format
func (e *ImageExtractor) processImage(img *model.Image, pageNum, imgIdx int) (PDFImage, error) {
	// Get image data
	imgData, err := img.ToGoImage()
	if err != nil {
		return PDFImage{}, fmt.Errorf("failed to convert to Go image: %w", err)
	}

	// Get dimensions
	bounds := imgData.Bounds()
	width := bounds.Dx()
	height := bounds.Dy()

	// Determine format from ColorSpace or filter
	format := "jpeg" // Default to JPEG

	// Encode image to bytes
	var buf bytes.Buffer

	jpeg.Encode(&buf, imgData, &jpeg.Options{Quality: 85})

	// Encode to base64
	base64Data := base64.StdEncoding.EncodeToString(buf.Bytes())

	return PDFImage{
		Data:       buf.Bytes(),
		Format:     format,
		Width:      width,
		Height:     height,
		PageNumber: pageNum,
		Base64:     base64Data,
	}, nil
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

// GetImageDataURL returns a data URL for HTML embedding
func (img *PDFImage) GetImageDataURL() string {
	mimeType := "image/jpeg"
	if img.Format == "png" {
		mimeType = "image/png"
	}
	return fmt.Sprintf("data:%s;base64,%s", mimeType, img.GetImageBase64())
}
