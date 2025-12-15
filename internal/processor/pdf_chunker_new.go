package processor

import (
	"strings"
)

// ChunkPDFPages chunks PDF pages while preserving page number context
// Strategy: Process each page individually, chunking only if page exceeds token limit
func (p *PDFChunker) ChunkPDFPagesNew(pages []PDFPage, metadata *PDFMetadata, filePath string) ([]PDFChunk, error) {
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
