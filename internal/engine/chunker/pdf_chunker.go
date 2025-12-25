package chunker

import (
	"path/filepath"
	"strings"
	"time"

	"github.com/karim-daw/qwelli/internal/engine/processor"
	"github.com/karim-daw/qwelli/internal/textutil"
)

// PDFChunkStrategy implements chunking for PDF pages
// This strategy processes PDF pages and preserves page number context
type PDFChunkStrategy struct {
	metadata *processor.PDFMetadata
	filePath string
}

// NewPDFChunkStrategy creates a new PDF chunking strategy
func NewPDFChunkStrategy(metadata *processor.PDFMetadata, filePath string) *PDFChunkStrategy {
	return &PDFChunkStrategy{
		metadata: metadata,
		filePath: filePath,
	}
}

func (s *PDFChunkStrategy) Chunkify(content interface{}, config ChunkerConfig, baseMetadata map[string]interface{}) ([]Chunk, error) {
	pages, ok := content.([]processor.PDFPage)
	if !ok {
		return nil, &StrategyError{Message: "PDFChunkStrategy requires []PDFPage content"}
	}

	var allChunks []Chunk

	// Process each page individually
	for _, page := range pages {
		// Skip empty pages
		if strings.TrimSpace(page.Text) == "" {
			continue
		}

		pageTokens := textutil.EstimateTokens(page.Text)

		// If page fits in one chunk, keep it as-is
		if pageTokens <= config.ChunkSize {
			chunk := Chunk{
				Content:     page.Text,
				ContentType: "text",
				PageNumbers: []int{page.PageNumber},
				ChunkIndex:  len(allChunks),
				TotalChunks: 0,   // Will be set later
				Metadata:    nil, // Will be set later
			}
			allChunks = append(allChunks, chunk)
		} else {
			// Page is too large, chunk it using text strategy
			textStrategy := &TextChunkStrategy{}
			// Merge page_number into baseMetadata if provided
			pageMetadata := make(map[string]interface{})
			for k, v := range baseMetadata {
				pageMetadata[k] = v
			}
			pageMetadata["page_number"] = page.PageNumber

			pageChunks, err := textStrategy.Chunkify(page.Text, config, pageMetadata)
			if err != nil {
				return nil, err
			}

			// Convert to chunks with page numbers - all from same page
			for _, c := range pageChunks {
				chunk := Chunk{
					Content:     c.Content,
					ContentType: "text",
					PageNumbers: []int{page.PageNumber},
					ChunkIndex:  len(allChunks),
					TotalChunks: 0,   // Will be set later
					Metadata:    nil, // Will be set later
				}
				allChunks = append(allChunks, chunk)
			}
		}
	}

	// If no text extracted, return single empty chunk
	if len(allChunks) == 0 {
		return []Chunk{
			{
				Content:     "",
				ContentType: "text",
				PageNumbers: []int{},
				ChunkIndex:  0,
				TotalChunks: 1,
				Metadata:    s.buildPDFMetadata([]int{}, 0, 1),
			},
		}, nil
	}

	// Now update all chunks with total count and metadata
	totalChunks := len(allChunks)
	for i := range allChunks {
		allChunks[i].ChunkIndex = i
		allChunks[i].TotalChunks = totalChunks
		// Build PDF metadata and merge with baseMetadata
		pdfMeta := s.buildPDFMetadata(allChunks[i].PageNumbers, i, totalChunks)
		// Merge baseMetadata into the final metadata (baseMetadata takes precedence for overlapping keys)
		for k, v := range baseMetadata {
			pdfMeta[k] = v
		}
		allChunks[i].Metadata = pdfMeta
	}

	return allChunks, nil
}

// buildPDFMetadata constructs the complete metadata map for a PDF chunk
// This is a helper function specific to PDF chunking
func (s *PDFChunkStrategy) buildPDFMetadata(pageNumbers []int, chunkIndex, totalChunks int) map[string]interface{} {
	metadata := map[string]interface{}{
		"file_path":    s.filePath,
		"file_name":    filepath.Base(s.filePath),
		"chunk_index":  chunkIndex,
		"total_chunks": totalChunks,
		"is_pdf":       true,
		"indexed_at":   time.Now().Format(time.RFC3339),
	}

	// Add optional metadata if available
	if s.metadata != nil {
		// Only add title if it's not empty
		if s.metadata.Title != "" {
			metadata["title"] = s.metadata.Title
		}

		// Add optional metadata
		if s.metadata.PageCount > 0 {
			metadata["page_count"] = s.metadata.PageCount
		}

		if s.metadata.Creator != "" {
			metadata["creator"] = s.metadata.Creator
		}

		if !s.metadata.CreationDate.IsZero() {
			metadata["creation_date"] = s.metadata.CreationDate.Format(time.RFC3339)
		}

		if !s.metadata.ModDate.IsZero() {
			metadata["modified_date"] = s.metadata.ModDate.Format(time.RFC3339)
		}
	}

	// Add page numbers
	if len(pageNumbers) > 0 {
		metadata["page_numbers"] = pageNumbers
	}

	return metadata
}
