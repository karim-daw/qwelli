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
	if options.EnableMultimodal {
		return p.processMultimodal(file, options)
	}
	return p.processTextOnly(file, options)
}

// processMultimodal processes a PDF with multimodal support (text + images)
func (p *PDFProcessor) processMultimodal(file db.File, options ProcessOptions) ([]db.Chunk, []string, error) {
	// Extract PDF text and metadata
	pdfProc, err := processor.NewPDFProcessor(true)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create PDF processor: %w", err)
	}
	pdfResult, err := pdfProc.ExtractText(file.Path)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to extract PDF text: %w", err)
	}

	// Extract images
	imageExtractor, err := processor.NewImageExtractor(1024, 1024, false)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create image extractor: %w", err)
	}
	images, err := imageExtractor.ExtractImages(file.Path)
	if err != nil {
		log.Printf("⚠️  Failed to extract images from %s: %v (continuing with text only)", filepath.Base(file.Path), err)
		images = []processor.PDFImage{}
	}

	// Create multimodal chunker
	pdfChunker := chunker.NewChunker(chunker.ChunkerConfig{
		ChunkSize:   options.ChunkSize,
		OverlapSize: options.OverlapSize,
	})
	multimodalChunker := chunker.NewMultimodalChunker(pdfChunker, imageExtractor)

	// Chunk PDF with multimodal support
	multimodalChunks, err := multimodalChunker.ChunkPDF(pdfResult.Pages, images, pdfResult.Metadata, file.Path)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to chunk PDF: %w", err)
	}

	// Convert to db.Chunk format
	var dbChunks []db.Chunk
	var contents []string

	for i, mc := range multimodalChunks {
		pageNumbers := []int{}
		if len(mc.PageNumbers) > 0 {
			pageNumbers = mc.PageNumbers
		}

		dbChunk := db.Chunk{
			ChunkID:     scanner.GenerateChunkID(file.FileID, i),
			FileID:      file.FileID,
			FilePath:    file.Path,
			FileType:    file.FileType,
			ChunkIndex:  mc.ChunkIndex,
			TotalChunks: mc.TotalChunks,
			Content:     mc.Content,
			PageNumbers: pageNumbers,
			ContentType: mc.ContentType,
			ImageData:   mc.ImageData,
		}

		dbChunks = append(dbChunks, dbChunk)
		contents = append(contents, mc.Content)
	}

	return dbChunks, contents, nil
}

// processTextOnly processes a PDF with text only
func (p *PDFProcessor) processTextOnly(file db.File, options ProcessOptions) ([]db.Chunk, []string, error) {
	// Extract PDF text and metadata
	pdfProc, err := processor.NewPDFProcessor(true)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create PDF processor: %w", err)
	}
	pdfResult, err := pdfProc.ExtractText(file.Path)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to extract PDF text: %w", err)
	}

	// Check if PDF has no text
	hasText := false
	for _, page := range pdfResult.Pages {
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

	chunks, err := pdfChunker.ChunkPDFPages(pdfResult.Pages, nil, file.Path)
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
