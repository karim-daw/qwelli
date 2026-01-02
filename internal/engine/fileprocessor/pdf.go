package fileprocessor

import (
	"fmt"
	"log"
	"path/filepath"
	"strings"

	"github.com/karim-daw/qwelli/internal/db"
	"github.com/karim-daw/qwelli/internal/engine/chunker"
	"github.com/karim-daw/qwelli/internal/engine/processor"
	"github.com/karim-daw/qwelli/internal/engine/scanner"
)

// PDFProcessor processes PDF files
type PDFProcessor struct{}

// NewPDFProcessor creates a new PDF processor
func NewPDFProcessor() *PDFProcessor {
	return &PDFProcessor{}
}

// CanProcess returns true for PDF files
func (p *PDFProcessor) CanProcess(fileType string) bool {
	return strings.ToLower(fileType) == "pdf"
}

// Process processes a PDF file and returns chunks
func (p *PDFProcessor) Process(file db.File, options ProcessOptions) ([]db.Chunk, []string, error) {
	// If text-only mode is selected, use text-only processing (skip image extraction entirely)
	if options.ContentTypeMode == ContentTypeText {
		return p.processTextOnly(file, options)
	}

	// If multimodal is enabled and we want images or both, use multimodal processing
	if options.EnableMultimodal {
		return p.processMultimodal(file, options)
	}

	// Default to text-only if multimodal is disabled
	return p.processTextOnly(file, options)
}

// processMultimodal processes a PDF with multimodal support (text + images)
func (p *PDFProcessor) processMultimodal(file db.File, options ProcessOptions) ([]db.Chunk, []string, error) {
	// Extract PDF text and metadata
	pdfProc := processor.NewPDFProcessor()
	pages, metadata, err := pdfProc.ExtractText(file.Path)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to extract PDF text: %w", err)
	}

	// Only extract images if we need them (not in text-only mode)
	var images []processor.PDFImage
	var imageExtractor *processor.ImageExtractor
	if options.ContentTypeMode != ContentTypeText {
		imageExtractor = processor.NewImageExtractor(1024, 1024)
		// Extract images page-by-page to get correct page numbers
		for _, page := range pages {
			pageImages, err := imageExtractor.ExtractImagesByPage(file.Path, page.PageNumber)
			if err != nil {
				log.Printf("⚠️  Failed to extract images from page %d of %s: %v (continuing)",
					page.PageNumber, filepath.Base(file.Path), err)
				continue
			}
			images = append(images, pageImages...)
		}
	} else {
		// Text-only mode: skip image extraction entirely
		images = []processor.PDFImage{}
		imageExtractor = processor.NewImageExtractor(1024, 1024) // Still need it for chunker, but won't extract
	}

	// Create multimodal chunker
	pdfChunker := chunker.NewChunker(chunker.ChunkerConfig{
		ChunkSize:   options.ChunkSize,
		OverlapSize: options.OverlapSize,
	})
	multimodalChunker := chunker.NewMultimodalChunker(pdfChunker, imageExtractor)

	// Chunk PDF with multimodal support
	multimodalChunks, err := multimodalChunker.ChunkPDF(pages, images, metadata, file.Path)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to chunk PDF: %w", err)
	}

	// Convert to db.Chunk format and filter based on ContentTypeMode
	var dbChunks []db.Chunk
	var contents []string
	chunkIndex := 0

	for _, mc := range multimodalChunks {
		// Filter based on content type mode
		switch options.ContentTypeMode {
		case ContentTypeText:
			if mc.ContentType != "text" {
				continue // Skip images
			}
		case ContentTypeImages:
			if mc.ContentType != "image" {
				continue // Skip text
			}
		case ContentTypeBoth:
			// Include both, no filtering
		default:
			// Default to both if not specified
		}

		pageNumbers := []int{}
		if len(mc.PageNumbers) > 0 {
			pageNumbers = mc.PageNumbers
		}

		dbChunk := db.Chunk{
			ChunkID:     scanner.GenerateChunkID(file.FileID, chunkIndex),
			FileID:      file.FileID,
			FilePath:    file.Path,
			FileType:    file.FileType,
			ChunkIndex:  chunkIndex,
			TotalChunks: 0, // Will be updated after filtering
			Content:     mc.Content,
			PageNumbers: pageNumbers,
			ContentType: mc.ContentType,
			ImageData:   mc.ImageData,
		}

		dbChunks = append(dbChunks, dbChunk)
		contents = append(contents, mc.Content)
		chunkIndex++
	}

	// Update total chunks count
	for i := range dbChunks {
		dbChunks[i].TotalChunks = len(dbChunks)
	}

	return dbChunks, contents, nil
}

// processTextOnly processes a PDF with text only
func (p *PDFProcessor) processTextOnly(file db.File, options ProcessOptions) ([]db.Chunk, []string, error) {
	// Extract PDF text and metadata
	pdfProc := processor.NewPDFProcessor()
	pages, _, err := pdfProc.ExtractText(file.Path)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to extract PDF text: %w", err)
	}

	// Check if PDF has no text
	hasText := false
	for _, page := range pages {
		if strings.TrimSpace(page.Text) != "" {
			hasText = true
			break
		}
	}

	if !hasText {
		return nil, nil, fmt.Errorf("skipping image-only PDF")
	}

	// Chunk the PDF
	pdfChunker := chunker.NewChunker(chunker.ChunkerConfig{
		ChunkSize:   options.ChunkSize,
		OverlapSize: options.OverlapSize,
	})

	chunks, err := pdfChunker.ChunkPDFPages(pages, nil, file.Path)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to chunk PDF: %w", err)
	}

	// Convert to db.Chunk format
	var dbChunks []db.Chunk
	var contents []string

	for i, chunk := range chunks {
		dbChunk := db.Chunk{
			ChunkID:     scanner.GenerateChunkID(file.FileID, i),
			FileID:      file.FileID,
			FilePath:    file.Path,
			FileType:    file.FileType,
			ChunkIndex:  chunk.ChunkIndex,
			TotalChunks: chunk.TotalChunks,
			Content:     chunk.Content,
			PageNumbers: chunk.PageNumbers,
		}

		dbChunks = append(dbChunks, dbChunk)
		contents = append(contents, chunk.Content)
	}

	return dbChunks, contents, nil
}
