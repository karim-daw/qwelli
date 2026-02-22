package chunker

import (
	"fmt"
	"log"
	"path/filepath"
	"sort"

	"github.com/karim-daw/qwelli/internal/engine/extraction"
)

// ChunkService provides a unified interface for all chunking operations
type ChunkService struct {
	config ChunkerConfig
}

// NewChunkService creates a service with given config
func NewChunkService(config ChunkerConfig) *ChunkService {
	return &ChunkService{
		config: config,
	}
}

// ChunkText chunks plain text content
func (s *ChunkService) ChunkText(text string) ([]Chunk, error) {
	strategy := &TextChunkStrategy{}
	return strategy.Chunkify(text, s.config, make(map[string]interface{}))
}

// ChunkPDF chunks PDF pages (text only)
func (s *ChunkService) ChunkPDF(pages []extraction.PDFPage, metadata *extraction.PDFMetadata, filePath string) ([]Chunk, error) {
	strategy := NewPDFChunkStrategy(metadata, filePath)
	return strategy.Chunkify(pages, s.config, make(map[string]interface{}))
}

// ChunkMultimodal chunks PDF with both text and images
func (s *ChunkService) ChunkMultimodal(pages []extraction.PDFPage, images []extraction.PDFImage, metadata *extraction.PDFMetadata, filePath string) ([]Chunk, error) {
	var allChunks []Chunk

	// First, get text chunks from PDF strategy
	pdfStrategy := NewPDFChunkStrategy(metadata, filePath)
	textChunks, err := pdfStrategy.Chunkify(pages, s.config, make(map[string]interface{}))
	if err != nil {
		return nil, fmt.Errorf("failed to chunk PDF text: %w", err)
	}

	// Debug: check if text chunks have page numbers
	for i, tc := range textChunks {
		if len(tc.PageNumbers) == 0 {
			log.Printf("⚠️  Text chunk %d from %s has NO page numbers", i, filepath.Base(filePath))
		}
	}

	// Text chunks are already in the unified Chunk format, just add them
	allChunks = append(allChunks, textChunks...)

	// Add image chunks (images should already be pre-filtered during extraction)
	// But we double-check here for safety
	skippedTooSmall := 0
	skippedTooLarge := 0
	for _, img := range images {
		pixelCount := img.Width * img.Height

		// Double-check size filters (should already be filtered during extraction)
		if img.Width < extraction.MinImageWidth || img.Height < extraction.MinImageHeight || pixelCount < extraction.MinImagePixels {
			skippedTooSmall++
			continue
		}

		if img.Width > extraction.MaxImageWidth || img.Height > extraction.MaxImageHeight || pixelCount > extraction.MaxImagePixels {
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
