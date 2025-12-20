package processor

import (
	"path/filepath"
	"strings"
	"time"
)

// PDFChunk represents a chunk from a PDF with page tracking
type PDFChunk struct {
	Content     string
	PageNumbers []int
	ChunkIndex  int
	TotalChunks int
	Metadata    map[string]interface{}
}

// PDFChunker handles PDF-specific chunking with page number tracking
type PDFChunker struct {
	chunker *Chunker
}

// NewPDFChunker creates a new PDF chunker with the given configuration
func NewPDFChunker(config ChunkerConfig) *PDFChunker {
	return &PDFChunker{
		chunker: NewChunker(config),
	}
}

// ChunkPDFPages chunks PDF pages while preserving page number context
// Strategy: Process each page individually, chunking only if page exceeds token limit
func (p *PDFChunker) ChunkPDFPages(pages []PDFPage, metadata *PDFMetadata, filePath string) ([]PDFChunk, error) {
	var allChunks []PDFChunk

	// Process each page individually
	for _, page := range pages {
		// Skip empty pages
		if strings.TrimSpace(page.Text) == "" {
			continue
		}

		pageTokens := EstimateTokens(page.Text)

		// If page fits in one chunk, keep it as-is
		if pageTokens <= p.chunker.config.ChunkSize {
			chunk := PDFChunk{
				Content:     page.Text,
				PageNumbers: []int{page.PageNumber},
				ChunkIndex:  len(allChunks),
				TotalChunks: 0,   // Will be set later
				Metadata:    nil, // Will be set later
			}
			allChunks = append(allChunks, chunk)
		} else {
			// Page is too large, chunk it
			baseMetadata := map[string]interface{}{
				"page_number": page.PageNumber,
			}

			pageChunks, err := p.chunker.ChunkByTokens(page.Text, baseMetadata)
			if err != nil {
				return nil, err
			}

			// Convert to PDF chunks - all from same page
			for _, c := range pageChunks {
				chunk := PDFChunk{
					Content:     c.Content,
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
		return []PDFChunk{
			{
				Content:     "",
				PageNumbers: []int{},
				ChunkIndex:  0,
				TotalChunks: 1,
				Metadata:    p.buildMetadata(metadata, filePath, []int{}, 0, 1),
			},
		}, nil
	}

	// Now update all chunks with total count and metadata
	totalChunks := len(allChunks)
	for i := range allChunks {
		allChunks[i].ChunkIndex = i
		allChunks[i].TotalChunks = totalChunks
		allChunks[i].Metadata = p.buildMetadata(metadata, filePath, allChunks[i].PageNumbers, i, totalChunks)
	}

	return allChunks, nil
}

// buildMetadata constructs the complete metadata map for a PDF chunk
func (p *PDFChunker) buildMetadata(pdfMetadata *PDFMetadata, filePath string, pageNumbers []int, chunkIndex, totalChunks int) map[string]interface{} {
	metadata := map[string]interface{}{
		"file_path":    filePath,
		"file_name":    filepath.Base(filePath),
		"chunk_index":  chunkIndex,
		"total_chunks": totalChunks,
		"is_pdf":       true,
		"indexed_at":   time.Now().Format(time.RFC3339),
	}

	// Only add title if it's not empty
	if pdfMetadata.Title != "" {
		metadata["title"] = pdfMetadata.Title
	}

	// Add page numbers
	if len(pageNumbers) > 0 {
		metadata["page_numbers"] = pageNumbers
	}

	// Add optional metadata
	if pdfMetadata.PageCount > 0 {
		metadata["page_count"] = pdfMetadata.PageCount
	}

	if pdfMetadata.Creator != "" {
		metadata["creator"] = pdfMetadata.Creator
	}

	if !pdfMetadata.CreationDate.IsZero() {
		metadata["creation_date"] = pdfMetadata.CreationDate.Format(time.RFC3339)
	}

	if !pdfMetadata.ModDate.IsZero() {
		metadata["modified_date"] = pdfMetadata.ModDate.Format(time.RFC3339)
	}

	return metadata
}
