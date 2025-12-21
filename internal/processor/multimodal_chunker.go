package processor

import (
	"fmt"
	"log"
	"sort"
)

// MultimodalChunk represents either a text chunk or an image chunk
type MultimodalChunk struct {
	Content     string
	ContentType string // "text" or "image"
	PageNumber  int
	ChunkIndex  int
	TotalChunks int
	Metadata    map[string]interface{}

	// Image-specific fields
	ImageData   []byte
	ImageFormat string
	ImageWidth  int
	ImageHeight int
}

// MultimodalChunker orchestrates text and image chunking from PDFs
type MultimodalChunker struct {
	pdfChunker     *PDFChunker
	imageExtractor *ImageExtractor
}

// NewMultimodalChunker creates a new multimodal chunker
func NewMultimodalChunker(pdfChunker *PDFChunker, imageExtractor *ImageExtractor) *MultimodalChunker {
	return &MultimodalChunker{
		pdfChunker:     pdfChunker,
		imageExtractor: imageExtractor,
	}
}

// ChunkPDF processes a PDF and returns both text and image chunks, sequenced by page
func (m *MultimodalChunker) ChunkPDF(pages []PDFPage, images []PDFImage, metadata *PDFMetadata, filePath string) ([]MultimodalChunk, error) {
	var allChunks []MultimodalChunk

	// First, get text chunks from PDFChunker
	textChunks, err := m.pdfChunker.ChunkPDFPages(pages, metadata, filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to chunk PDF text: %w", err)
	}

	// Convert text chunks to multimodal chunks
	for _, tc := range textChunks {
		pageNum := 1
		if len(tc.PageNumbers) > 0 {
			pageNum = tc.PageNumbers[0]
		}

		chunk := MultimodalChunk{
			Content:     tc.Content,
			ContentType: "text",
			PageNumber:  pageNum,
			ChunkIndex:  tc.ChunkIndex,
			TotalChunks: tc.TotalChunks,
			Metadata:    tc.Metadata,
		}
		allChunks = append(allChunks, chunk)
	}

	// Add image chunks (filter out tiny and huge images)
	skippedTooSmall := 0
	skippedTooLarge := 0
	for _, img := range images {
		// Filter out images that are too small (~1/3 of A4 page minimum)
		// Minimum: 500x350 pixels (175,000 pixels total)
		minWidth := 500
		minHeight := 350
		minPixels := 175_000
		pixelCount := img.Width * img.Height

		if img.Width < minWidth || img.Height < minHeight || pixelCount < minPixels {
			skippedTooSmall++
			continue
		}

		// Filter out images that are too large (max dimensions and pixel count)
		// Maximum: 4000x3000 pixels (12M pixels) - reasonable upper limit for search
		// Voyage API limit is 16M pixels, but we'll be more conservative
		maxWidth := 4000
		maxHeight := 3000
		maxPixels := 12_000_000 // 12 million pixels

		if img.Width > maxWidth || img.Height > maxHeight || pixelCount > maxPixels {
			skippedTooLarge++
			continue
		}

		// Store base64-encoded image data
		imageBase64Bytes := []byte(img.GetImageBase64())
		chunk := MultimodalChunk{
			Content:     fmt.Sprintf("Image from page %d (%dx%d, %s)", img.PageNumber, img.Width, img.Height, img.Format),
			ContentType: "image",
			PageNumber:  img.PageNumber,
			ImageData:   imageBase64Bytes, // Store base64 string as bytes
			ImageFormat: img.Format,
			ImageWidth:  img.Width,
			ImageHeight: img.Height,
			Metadata: map[string]interface{}{
				"image_format": img.Format,
				"image_width":  img.Width,
				"image_height": img.Height,
				"page_number":  img.PageNumber,
			},
		}
		allChunks = append(allChunks, chunk)
	}

	// Log summary of filtered images
	if skippedTooSmall > 0 || skippedTooLarge > 0 {
		log.Printf("📊 Image filtering: %d too small, %d too large, %d accepted", skippedTooSmall, skippedTooLarge, len(images)-skippedTooSmall-skippedTooLarge)
	}

	// Sort chunks by page number, then by type (text before images on same page)
	sort.Slice(allChunks, func(i, j int) bool {
		if allChunks[i].PageNumber != allChunks[j].PageNumber {
			return allChunks[i].PageNumber < allChunks[j].PageNumber
		}
		// On same page, text comes before images
		if allChunks[i].ContentType != allChunks[j].ContentType {
			return allChunks[i].ContentType == "text"
		}
		return allChunks[i].ChunkIndex < allChunks[j].ChunkIndex
	})

	// Update chunk indices and total counts
	totalChunks := len(allChunks)
	for i := range allChunks {
		allChunks[i].ChunkIndex = i
		allChunks[i].TotalChunks = totalChunks
	}

	return allChunks, nil
}
