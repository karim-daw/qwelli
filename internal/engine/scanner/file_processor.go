package scanner

import (
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"io/fs"
	"log"
	"path/filepath"
	"strings"

	"github.com/karim-daw/qwelli/internal/db"
	"github.com/karim-daw/qwelli/internal/engine/chunker"
	"github.com/karim-daw/qwelli/internal/engine/processor"
	"github.com/karim-daw/qwelli/internal/textutil"
)

// Helper functions

func ScanFolder(root string) ([]string, error) {
	var files []string
	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			name := d.Name()
			if strings.HasPrefix(name, ".") || name == "node_modules" || name == "vendor" {
				return filepath.SkipDir
			}
			return nil
		}
		if strings.HasPrefix(d.Name(), ".") || !IsTextFile(path) {
			return nil
		}
		files = append(files, path)
		return nil
	})
	return files, err
}

func IsTextFile(path string) bool {
	ext := strings.ToLower(filepath.Ext(path))
	textExts := map[string]bool{
		".txt": true, ".md": true, ".go": true, ".py": true, ".js": true, ".ts": true,
		".tsx": true, ".jsx": true, ".java": true, ".c": true, ".cpp": true, ".h": true,
		".rs": true, ".rb": true, ".php": true, ".cs": true, ".swift": true,
		".html": true, ".css": true, ".scss": true, ".yaml": true, ".yml": true,
		".toml": true, ".sh": true, ".proto": true, ".graphql": true,
		".pdf": true, // PDF support
	}
	return textExts[ext]
}

func GenerateFileID(path string) string {
	hash := md5.Sum([]byte(path))
	return hex.EncodeToString(hash[:])
}

func GenerateChunkID(fileID string, chunkIndex int) string {
	source := fmt.Sprintf("%s:chunk:%d", fileID, chunkIndex)
	hash := md5.Sum([]byte(source))
	return hex.EncodeToString(hash[:])
}

func GetFileTypeFromPath(path string) string {
	ext := strings.ToLower(filepath.Ext(path))
	if ext == "" {
		return "unknown"
	}
	return ext[1:]
}

// ProcessPDFFileMultimodal processes a PDF file with multimodal support (text + images)
func ProcessPDFFileMultimodal(file db.File) ([]db.Chunk, []string, error) {
	// Extract PDF text and metadata
	pdfProc := processor.NewPDFProcessor()
	pages, metadata, err := pdfProc.ExtractText(file.Path)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to extract PDF text: %w", err)
	}

	// Extract images
	imageExtractor := processor.NewImageExtractor(1024, 1024)
	images, err := imageExtractor.ExtractImages(file.Path)
	if err != nil {
		log.Printf("⚠️  Failed to extract images from %s: %v (continuing with text only)", filepath.Base(file.Path), err)
		images = []processor.PDFImage{}
	}

	// Create multimodal chunker
	pdfChunker := chunker.NewChunker(chunker.ChunkerConfig{
		ChunkSize:   300,
		OverlapSize: 10,
	})
	multimodalChunker := chunker.NewMultimodalChunker(pdfChunker, imageExtractor)

	// Chunk PDF with multimodal support
	multimodalChunks, err := multimodalChunker.ChunkPDF(pages, images, metadata, file.Path)
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
			ChunkID:     GenerateChunkID(file.FileID, i),
			FileID:      file.FileID,
			FilePath:    file.Path,
			FileType:    file.FileType,
			ChunkIndex:  mc.ChunkIndex,
			TotalChunks: mc.TotalChunks,
			Content:     mc.Content,
			PageNumbers: pageNumbers,
			ContentType: mc.ContentType,
			ImageData:   mc.ImageData, // Already base64 string as bytes
		}

		dbChunks = append(dbChunks, dbChunk)

		// For embedding, use text content for text chunks
		// Images will be handled separately in the embedding generation
		contents = append(contents, mc.Content)
	}

	return dbChunks, contents, nil
}

// ProcessPDFFile processes a PDF file and returns chunks
func ProcessPDFFile(file db.File) ([]db.Chunk, []string, error) {
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
		ChunkSize:   300,
		OverlapSize: 10,
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
			ChunkID:     GenerateChunkID(file.FileID, i),
			FileID:      file.FileID,
			FilePath:    file.Path,     // Denormalized
			FileType:    file.FileType, // Denormalized
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

// ProcessTextFile processes a text file and returns chunks
func ProcessTextFile(file db.File, content string) ([]db.Chunk, []string, error) {
	// Estimate tokens
	estimatedTokens := textutil.EstimateTokens(content)

	var dbChunks []db.Chunk
	var contents []string

	if estimatedTokens > 1000 {
		// Chunk large text files
		textChunker := chunker.NewChunker(chunker.ChunkerConfig{
			ChunkSize:   300,
			OverlapSize: 10,
		})

		chunks, err := textChunker.ChunkByTokens(content, nil)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to chunk text: %w", err)
		}

		for i, chunk := range chunks {
			dbChunk := db.Chunk{
				ChunkID:     GenerateChunkID(file.FileID, i),
				FileID:      file.FileID,
				FilePath:    file.Path,     // Denormalized
				FileType:    file.FileType, // Denormalized
				ChunkIndex:  i,
				TotalChunks: len(chunks),
				Content:     chunk.Content,
				PageNumbers: []int{}, // Text files don't have pages
			}

			dbChunks = append(dbChunks, dbChunk)
			contents = append(contents, chunk.Content)
		}
	} else {
		// Small files: keep as single chunk
		dbChunk := db.Chunk{
			ChunkID:     GenerateChunkID(file.FileID, 0),
			FileID:      file.FileID,
			FilePath:    file.Path,     // Denormalized
			FileType:    file.FileType, // Denormalized
			ChunkIndex:  0,
			TotalChunks: 1,
			Content:     content,
			PageNumbers: []int{},
		}

		dbChunks = append(dbChunks, dbChunk)
		contents = append(contents, content)
	}

	return dbChunks, contents, nil
}
