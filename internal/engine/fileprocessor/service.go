package fileprocessor

import (
	"fmt"
	"log"
	"path/filepath"
	"strings"

	"github.com/karim-daw/qwelli/internal/db"
	"github.com/karim-daw/qwelli/internal/engine/chunker"
	"github.com/karim-daw/qwelli/internal/engine/processor"
	"github.com/karim-daw/qwelli/internal/textutil"
)

// ProcessingConfig holds configuration for file processing
type ProcessingConfig struct {
	ChunkerConfig    chunker.ChunkerConfig
	EnableMultimodal bool
	ContentTypeMode  ContentTypeMode
}

// DefaultProcessingConfig returns default processing configuration
func DefaultProcessingConfig() ProcessingConfig {
	return ProcessingConfig{
		ChunkerConfig:    chunker.DefaultConfig,
		EnableMultimodal: true,
		ContentTypeMode:  ContentTypeBoth,
	}
}

// FileProcessingService provides unified file processing functionality
// It holds reusable dependencies to avoid repeated allocations
type FileProcessingService struct {
	chunkService   *chunker.ChunkService
	pdfProcessor   *processor.PDFProcessor
	imageExtractor *processor.ImageExtractor
	config         ProcessingConfig
}

// SetContentTypeMode updates the content type mode
func (s *FileProcessingService) SetContentTypeMode(mode ContentTypeMode) {
	s.config.ContentTypeMode = mode
}

// NewFileProcessingService creates a new file processing service with given config
func NewFileProcessingService(config ProcessingConfig) *FileProcessingService {
	return &FileProcessingService{
		chunkService:   chunker.NewChunkService(config.ChunkerConfig),
		pdfProcessor:   processor.NewPDFProcessor(),
		imageExtractor: processor.NewImageExtractor(1024, 1024),
		config:         config,
	}
}

// CanProcess returns true if this service can process the given file type
func (s *FileProcessingService) CanProcess(fileType string) bool {
	return IsSupported(fileType)
}

// Process processes a file and returns chunks based on file type
// For text files, content must be provided in options.FileContent
func (s *FileProcessingService) Process(file db.File, options ProcessOptions) ([]db.Chunk, []string, error) {
	if IsPDFFile(file.FileType) {
		return s.processPDF(file, options)
	}

	if IsTextFile(file.FileType) {
		return s.processText(file, options)
	}

	return nil, nil, fmt.Errorf("unsupported file type: %s", file.FileType)
}

// ProcessText is a convenience method for processing text files
func (s *FileProcessingService) ProcessText(file db.File, content string) ([]db.Chunk, []string, error) {
	options := ProcessOptions{
		FileContent:      content,
		ChunkSize:        s.config.ChunkerConfig.ChunkSize,
		OverlapSize:      s.config.ChunkerConfig.OverlapSize,
		EnableMultimodal: false, // Text files don't use multimodal
		ContentTypeMode:  ContentTypeText,
	}
	return s.processText(file, options)
}

// ProcessPDF is a convenience method for processing PDF files
func (s *FileProcessingService) ProcessPDF(file db.File) ([]db.Chunk, []string, error) {
	options := ProcessOptions{
		ChunkSize:        s.config.ChunkerConfig.ChunkSize,
		OverlapSize:      s.config.ChunkerConfig.OverlapSize,
		EnableMultimodal: s.config.EnableMultimodal,
		ContentTypeMode:  s.config.ContentTypeMode,
	}
	return s.processPDF(file, options)
}

// processText handles text file processing
func (s *FileProcessingService) processText(file db.File, options ProcessOptions) ([]db.Chunk, []string, error) {
	if options.FileContent == "" {
		return nil, nil, fmt.Errorf("text processing requires file content in ProcessOptions")
	}

	content := options.FileContent
	estimatedTokens := textutil.EstimateTokens(content)

	var dbChunks []db.Chunk
	var contents []string

	if estimatedTokens > 1000 {
		// Chunk large text files
		chunks, err := s.chunkService.ChunkText(content)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to chunk text: %w", err)
		}

		// Convert to db.Chunk format
		dbChunks = chunker.ToDBChunks(chunks, file)

		// Extract contents for embedding
		for _, chunk := range chunks {
			contents = append(contents, chunk.Content)
		}
	} else {
		// Small files: keep as single chunk
		chunk := chunker.Chunk{
			Content:     content,
			ContentType: "text",
			ChunkIndex:  0,
			TotalChunks: 1,
			Metadata:    make(map[string]interface{}),
		}
		dbChunks = chunker.ToDBChunks([]chunker.Chunk{chunk}, file)
		contents = []string{content}
	}

	return dbChunks, contents, nil
}

// processPDF handles PDF file processing
func (s *FileProcessingService) processPDF(file db.File, options ProcessOptions) ([]db.Chunk, []string, error) {
	// If text-only mode is selected, use text-only processing
	if options.ContentTypeMode == ContentTypeText {
		return s.processPDFTextOnly(file, options)
	}

	// If multimodal is enabled and we want images or both, use multimodal processing
	if options.EnableMultimodal {
		return s.processPDFMultimodal(file, options)
	}

	// Default to text-only if multimodal is disabled
	return s.processPDFTextOnly(file, options)
}

// processPDFMultimodal processes a PDF with multimodal support (text + images)
func (s *FileProcessingService) processPDFMultimodal(file db.File, options ProcessOptions) ([]db.Chunk, []string, error) {
	// Extract PDF text and metadata
	pages, metadata, err := s.pdfProcessor.ExtractText(file.Path)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to extract PDF text: %w", err)
	}

	// Only extract images if we need them (not in text-only mode)
	var images []processor.PDFImage
	if options.ContentTypeMode != ContentTypeText {
		// Extract images page-by-page to get correct page numbers
		for _, page := range pages {
			pageImages, err := s.imageExtractor.ExtractImagesByPage(file.Path, page.PageNumber)
			if err != nil {
				log.Printf("⚠️  Failed to extract images from page %d of %s: %v (continuing)",
					page.PageNumber, filepath.Base(file.Path), err)
				continue
			}
			images = append(images, pageImages...)
		}
	} else {
		images = []processor.PDFImage{}
	}

	// Chunk PDF with multimodal support
	multimodalChunks, err := s.chunkService.ChunkMultimodal(pages, images, metadata, file.Path)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to chunk PDF: %w", err)
	}

	// Filter chunks based on ContentTypeMode
	filteredChunks := s.filterChunksByContentType(multimodalChunks, options.ContentTypeMode)

	// Convert to db.Chunk format
	dbChunks := chunker.ToDBChunks(filteredChunks, file)

	// Extract contents for embedding
	var contents []string
	for _, chunk := range filteredChunks {
		contents = append(contents, chunk.Content)
	}

	return dbChunks, contents, nil
}

// processPDFTextOnly processes a PDF with text only
func (s *FileProcessingService) processPDFTextOnly(file db.File, options ProcessOptions) ([]db.Chunk, []string, error) {
	// Extract PDF text and metadata
	pages, _, err := s.pdfProcessor.ExtractText(file.Path)
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
	chunks, err := s.chunkService.ChunkPDF(pages, nil, file.Path)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to chunk PDF: %w", err)
	}

	// Convert to db.Chunk format
	dbChunks := chunker.ToDBChunks(chunks, file)

	// Extract contents for embedding
	var contents []string
	for _, chunk := range chunks {
		contents = append(contents, chunk.Content)
	}

	return dbChunks, contents, nil
}

// filterChunksByContentType filters chunks based on content type mode
func (s *FileProcessingService) filterChunksByContentType(chunks []chunker.Chunk, mode ContentTypeMode) []chunker.Chunk {
	if mode == ContentTypeBoth {
		return chunks // No filtering
	}

	var filtered []chunker.Chunk
	for _, chunk := range chunks {
		switch mode {
		case ContentTypeText:
			if chunk.ContentType == "text" {
				filtered = append(filtered, chunk)
			}
		case ContentTypeImages:
			if chunk.ContentType == "image" {
				filtered = append(filtered, chunk)
			}
		}
	}
	return filtered
}
