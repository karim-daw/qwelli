package fileprocessor

import (
	"fmt"
	"strings"

	"github.com/karim-daw/qwelli/internal/db"
	"github.com/karim-daw/qwelli/internal/engine/chunker"
	"github.com/karim-daw/qwelli/internal/engine/scanner"
	"github.com/karim-daw/qwelli/internal/textutil"
)

// TextProcessor processes text files
type TextProcessor struct{}

// NewTextProcessor creates a new text processor
func NewTextProcessor() *TextProcessor {
	return &TextProcessor{}
}

// CanProcess returns true for text file types
func (p *TextProcessor) CanProcess(fileType string) bool {
	textTypes := map[string]bool{
		"txt": true, "md": true, "go": true, "py": true, "js": true, "ts": true,
		"tsx": true, "jsx": true, "java": true, "c": true, "cpp": true, "h": true,
		"rs": true, "rb": true, "php": true, "cs": true, "swift": true,
		"html": true, "css": true, "scss": true, "yaml": true, "yml": true,
		"toml": true, "sh": true, "proto": true, "graphql": true,
	}
	return textTypes[strings.ToLower(fileType)]
}

// Process processes a text file and returns chunks
func (p *TextProcessor) Process(file db.File, options ProcessOptions) ([]db.Chunk, []string, error) {
	if options.FileContent == "" {
		return nil, nil, fmt.Errorf("text processor requires file content in ProcessOptions")
	}
	return ProcessTextFile(file, options.FileContent, options)
}

// ProcessTextFile processes a text file with content and returns chunks
func ProcessTextFile(file db.File, content string, options ProcessOptions) ([]db.Chunk, []string, error) {
	// Estimate tokens
	estimatedTokens := textutil.EstimateTokens(content)

	var dbChunks []db.Chunk
	var contents []string

	if estimatedTokens > 1000 {
		// Chunk large text files
		textChunker := chunker.NewChunker(chunker.ChunkerConfig{
			ChunkSize:   options.ChunkSize,
			OverlapSize: options.OverlapSize,
		})

		chunks, err := textChunker.ChunkByTokens(content, nil)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to chunk text: %w", err)
		}

		for i, chunk := range chunks {
			dbChunk := db.Chunk{
				ChunkID:     scanner.GenerateChunkID(file.FileID, i),
				FileID:      file.FileID,
				FilePath:    file.Path,
				FileType:    file.FileType,
				ChunkIndex:  i,
				TotalChunks: len(chunks),
				Content:     chunk.Content,
				PageNumbers: []int{},
			}

			dbChunks = append(dbChunks, dbChunk)
			contents = append(contents, chunk.Content)
		}
	} else {
		// Small files: keep as single chunk
		dbChunk := db.Chunk{
			ChunkID:     scanner.GenerateChunkID(file.FileID, 0),
			FileID:      file.FileID,
			FilePath:    file.Path,
			FileType:    file.FileType,
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
