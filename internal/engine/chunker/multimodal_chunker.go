package chunker

import (
	"fmt"
	"log"
	"sort"

	"github.com/karim-daw/qwelli/internal/engine/processor"
)

// MultimodalChunker orchestrates text and image chunking from PDFs
// This chunker combines text chunks from PDF pages with image chunks extracted from PDFs
// Future: Can be extended to handle standalone image files
type MultimodalChunker struct {
	pdfChunker     *Chunker
	imageExtractor *processor.ImageExtractor
}

// NewMultimodalChunker creates a new multimodal chunker
func NewMultimodalChunker(pdfChunker *Chunker, imageExtractor *processor.ImageExtractor) *MultimodalChunker {
	return &MultimodalChunker{
		pdfChunker:     pdfChunker,
		imageExtractor: imageExtractor,
	}
}

// ChunkPDF processes a PDF and returns both text and image chunks, sequenced by page
// Returns unified Chunk types that can represent both text and image content
func (m *MultimodalChunker) ChunkPDF(pages []processor.PDFPage, images []processor.PDFImage, metadata *processor.PDFMetadata, filePath string) ([]Chunk, error) {
	var allChunks []Chunk

	// First, get text chunks from PDFChunker
	// Use the PDFChunkStrategy constructor
	pdfStrategy := NewPDFChunkStrategy(metadata, filePath)
	// Pass empty map instead of nil to allow PDFChunkStrategy to merge any additional metadata
	textChunks, err := pdfStrategy.Chunkify(pages, m.pdfChunker.config, make(map[string]interface{}))
	if err != nil {
		return nil, fmt.Errorf("failed to chunk PDF text: %w", err)
	}

	// Text chunks are already in the unified Chunk format, just add them
	allChunks = append(allChunks, textChunks...)

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
		imageBase64Bytes := []byte(img.Base64)
		chunk := Chunk{
			Content:     fmt.Sprintf("Image from page %d (%dx%d, %s)", img.PageNumber, img.Width, img.Height, img.Format),
			ContentType: "image",
			PageNumbers: []int{img.PageNumber},
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
		pageI := 0
		pageJ := 0
		if len(allChunks[i].PageNumbers) > 0 {
			pageI = allChunks[i].PageNumbers[0]
		}
		if len(allChunks[j].PageNumbers) > 0 {
			pageJ = allChunks[j].PageNumbers[0]
		}

		if pageI != pageJ {
			return pageI < pageJ
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
