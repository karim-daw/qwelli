package chunker

import (
	"log"
	"path/filepath"
	"strings"
	"time"

	"github.com/karim-daw/qwelli/internal/engine/extraction"
	"github.com/karim-daw/qwelli/internal/textutil"
)

// PDFChunkStrategy implements chunking for PDF pages
// This strategy processes PDF pages and preserves page number context
type PDFChunkStrategy struct {
	metadata *extraction.PDFMetadata
	filePath string
}

// NewPDFChunkStrategy creates a new PDF chunking strategy
func NewPDFChunkStrategy(metadata *extraction.PDFMetadata, filePath string) *PDFChunkStrategy {
	return &PDFChunkStrategy{
		metadata: metadata,
		filePath: filePath,
	}
}

// MinTokensForTextPage is the minimum tokens a page needs to be considered text-worthy.
// Pages with fewer tokens are assumed to be image-heavy and their sparse text is skipped.
const MinTokensForTextPage = 30

func (s *PDFChunkStrategy) Chunkify(content interface{}, config ChunkerConfig, baseMetadata map[string]interface{}) ([]Chunk, error) {
	pages, ok := content.([]extraction.PDFPage)
	if !ok {
		return nil, &StrategyError{Message: "PDFChunkStrategy requires []PDFPage content"}
	}

	var allChunks []Chunk
	skippedSparsePages := 0

	// Process each page individually
	for _, page := range pages {
		trimmedText := strings.TrimSpace(page.Text)

		// Skip empty pages
		if trimmedText == "" {
			continue
		}

		// Skip text-sparse pages (likely image-only pages with captions/headers)
		// These pages typically have scattered text that doesn't add search value
		pageTokens := textutil.EstimateTokens(trimmedText)
		if pageTokens < MinTokensForTextPage {
			skippedSparsePages++
			continue
		}

		// Debug: check if page has valid page number
		if page.PageNumber <= 0 {
			log.Printf("⚠️  PDF chunker: Page has invalid PageNumber: %d (should be >= 1)", page.PageNumber)
		}

		// If page fits in one chunk, keep it as-is (pageTokens already calculated above)
		if pageTokens <= config.ChunkSize {
			chunk := Chunk{
				Content:     trimmedText,
				ContentType: "text",
				PageNumbers: []int{page.PageNumber},
				ChunkIndex:  len(allChunks),
				TotalChunks: 0,   // Will be set later
				Metadata:    nil, // Will be set later
			}
			if page.PageNumber <= 0 {
				log.Printf("⚠️  PDF chunker: Created chunk with invalid PageNumber: %d", page.PageNumber)
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

	// Log skipped sparse pages
	if skippedSparsePages > 0 {
		log.Printf("  Skipped %d text-sparse pages (< %d tokens)", skippedSparsePages, MinTokensForTextPage)
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
