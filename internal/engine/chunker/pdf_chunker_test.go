package chunker

import (
	"testing"
	"time"

	"github.com/karim-daw/qwelli/internal/engine/processor"
)

func TestNewPDFChunker(t *testing.T) {
	config := ChunkerConfig{
		ChunkSize:   1000,
		OverlapSize: 150,
	}

	chunker := NewChunker(config)

	if chunker == nil {
		t.Fatal("NewChunker returned nil")
	}

	if chunker.config.ChunkSize != 1000 {
		t.Errorf("Expected ChunkSize 1000, got %d", chunker.config.ChunkSize)
	}

	if chunker.config.OverlapSize != 150 {
		t.Errorf("Expected OverlapSize 150, got %d", chunker.config.OverlapSize)
	}
}

func TestChunkPDFPages_SmallPDF(t *testing.T) {
	chunker := NewChunker(ChunkerConfig{
		ChunkSize:   1000,
		OverlapSize: 150,
	})

	// Create a small PDF with 2 pages
	pages := []processor.PDFPage{
		{PageNumber: 1, Text: "This is the first page. It has some content."},
		{PageNumber: 2, Text: "This is the second page. It has different content."},
	}

	metadata := &processor.PDFMetadata{
		Title:     "Test Document",
		PageCount: 2,
	}

	chunks, err := chunker.ChunkPDFPages(pages, metadata, "/path/to/test.pdf")
	if err != nil {
		t.Fatalf("ChunkPDFPages failed: %v", err)
	}

	// Small PDF with 2 pages results in 2 chunks (one per page)
	if len(chunks) != 2 {
		t.Errorf("Expected 2 chunks, got %d", len(chunks))
	}

	// Verify first chunk
	chunk := chunks[0]
	if chunk.ChunkIndex != 0 {
		t.Errorf("Expected chunk_index 0, got %d", chunk.ChunkIndex)
	}

	if chunk.TotalChunks != 2 {
		t.Errorf("Expected total_chunks 2, got %d", chunk.TotalChunks)
	}

	// Each chunk should have 1 page number
	if len(chunk.PageNumbers) != 1 {
		t.Errorf("Expected 1 page number in first chunk, got %d", len(chunk.PageNumbers))
	}

	if chunk.PageNumbers[0] != 1 {
		t.Errorf("Expected first chunk to be page 1, got %d", chunk.PageNumbers[0])
	}

	// Verify metadata fields
	if chunk.Metadata["title"] != "Test Document" {
		t.Errorf("Expected title 'Test Document', got %v", chunk.Metadata["title"])
	}

	if chunk.Metadata["page_count"] != 2 {
		t.Errorf("Expected page_count 2, got %v", chunk.Metadata["page_count"])
	}

	if chunk.Metadata["is_pdf"] != true {
		t.Errorf("Expected is_pdf true, got %v", chunk.Metadata["is_pdf"])
	}
}

func TestChunkPDFPages_LargePDF(t *testing.T) {
	chunker := NewChunker(ChunkerConfig{
		ChunkSize:   100, // Small chunk size to force multiple chunks
		OverlapSize: 20,
	})

	// Create a larger PDF with multiple pages
	pages := []processor.PDFPage{
		{PageNumber: 1, Text: "This is page one with a lot of text. It contains multiple sentences to ensure chunking happens. We want to test the page tracking."},
		{PageNumber: 2, Text: "This is page two with more content. It also has multiple sentences. The chunker should track which pages each chunk comes from."},
		{PageNumber: 3, Text: "This is page three. It has additional content. The page numbers should be correctly mapped to chunks."},
	}

	metadata := &processor.PDFMetadata{
		Title:        "Large Test Document",
		PageCount:    3,
		Creator:      "Test Creator",
		CreationDate: time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC),
	}

	chunks, err := chunker.ChunkPDFPages(pages, metadata, "/path/to/large.pdf")
	if err != nil {
		t.Fatalf("ChunkPDFPages failed: %v", err)
	}

	// Should create multiple chunks
	if len(chunks) < 2 {
		t.Errorf("Expected multiple chunks, got %d", len(chunks))
	}

	// Verify chunk indices
	for i, chunk := range chunks {
		if chunk.ChunkIndex != i {
			t.Errorf("Chunk %d has incorrect index: %d", i, chunk.ChunkIndex)
		}

		if chunk.TotalChunks != len(chunks) {
			t.Errorf("Chunk %d has incorrect total_chunks: %d", i, chunk.TotalChunks)
		}
	}

	// Verify metadata presence
	for i, chunk := range chunks {
		if chunk.Metadata["title"] != "Large Test Document" {
			t.Errorf("Chunk %d has incorrect title", i)
		}

		if chunk.Metadata["creator"] != "Test Creator" {
			t.Errorf("Chunk %d has incorrect creator", i)
		}

		if chunk.Metadata["page_count"] != 3 {
			t.Errorf("Chunk %d has incorrect page_count", i)
		}

		// Check that page numbers are present
		if pageNums, ok := chunk.Metadata["page_numbers"].([]int); ok {
			if len(pageNums) == 0 {
				t.Errorf("Chunk %d has no page numbers", i)
			}
		}
	}

	t.Logf("Created %d chunks from 3-page PDF", len(chunks))
	for i, chunk := range chunks {
		t.Logf("Chunk %d spans pages: %v", i, chunk.PageNumbers)
	}
}

func TestChunkPDFPages_EmptyPDF(t *testing.T) {
	chunker := NewChunker(ChunkerConfig{
		ChunkSize:   1000,
		OverlapSize: 150,
	})

	// PDF with no text
	pages := []processor.PDFPage{
		{PageNumber: 1, Text: ""},
		{PageNumber: 2, Text: ""},
	}

	metadata := &processor.PDFMetadata{
		Title:     "Empty Document",
		PageCount: 2,
	}

	chunks, err := chunker.ChunkPDFPages(pages, metadata, "/path/to/empty.pdf")
	if err != nil {
		t.Fatalf("ChunkPDFPages failed: %v", err)
	}

	// Should still create one chunk
	if len(chunks) != 1 {
		t.Errorf("Expected 1 chunk for empty PDF, got %d", len(chunks))
	}

	if chunks[0].Content != "" {
		t.Errorf("Expected empty content, got: %s", chunks[0].Content)
	}
}

func TestChunkPDFPages_PageNumberTracking(t *testing.T) {
	chunker := NewChunker(ChunkerConfig{
		ChunkSize:   50, // Very small to test page tracking
		OverlapSize: 10,
	})

	pages := []processor.PDFPage{
		{PageNumber: 1, Text: "Page one text goes here."},
		{PageNumber: 2, Text: "Page two text goes here."},
		{PageNumber: 3, Text: "Page three text goes here."},
	}

	metadata := &processor.PDFMetadata{
		Title:     "Page Tracking Test",
		PageCount: 3,
	}

	chunks, err := chunker.ChunkPDFPages(pages, metadata, "/path/to/test.pdf")
	if err != nil {
		t.Fatalf("ChunkPDFPages failed: %v", err)
	}

	// Verify all page numbers are accounted for
	allPageNumbers := make(map[int]bool)
	for _, chunk := range chunks {
		for _, pageNum := range chunk.PageNumbers {
			allPageNumbers[pageNum] = true

			// Page numbers should be in valid range
			if pageNum < 1 || pageNum > 3 {
				t.Errorf("Invalid page number: %d", pageNum)
			}
		}
	}

	// All pages should appear in at least one chunk
	if len(allPageNumbers) != 3 {
		t.Errorf("Expected all 3 pages to be covered, got %d pages", len(allPageNumbers))
	}
}

func TestChunkPDFPages_MetadataStructure(t *testing.T) {
	chunker := NewChunker(ChunkerConfig{
		ChunkSize:   1000,
		OverlapSize: 150,
	})

	pages := []processor.PDFPage{
		{PageNumber: 1, Text: "Test content."},
	}

	creationDate := time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC)
	modDate := time.Date(2024, 2, 20, 14, 20, 0, 0, time.UTC)

	metadata := &processor.PDFMetadata{
		Title:        "Metadata Test",
		Creator:      "Test Creator",
		CreationDate: creationDate,
		ModDate:      modDate,
		PageCount:    1,
	}

	chunks, err := chunker.ChunkPDFPages(pages, metadata, "/path/to/metadata_test.pdf")
	if err != nil {
		t.Fatalf("ChunkPDFPages failed: %v", err)
	}

	if len(chunks) == 0 {
		t.Fatal("No chunks created")
	}

	chunk := chunks[0]

	// Verify all expected metadata fields
	expectedFields := []string{
		"file_path", "file_name", "title", "chunk_index",
		"total_chunks", "is_pdf", "indexed_at", "page_count",
		"creator", "creation_date", "modified_date", "page_numbers",
	}

	for _, field := range expectedFields {
		if _, ok := chunk.Metadata[field]; !ok {
			t.Errorf("Missing metadata field: %s", field)
		}
	}

	// Verify specific values
	if chunk.Metadata["file_name"] != "metadata_test.pdf" {
		t.Errorf("Incorrect file_name: %v", chunk.Metadata["file_name"])
	}

	if chunk.Metadata["file_path"] != "/path/to/metadata_test.pdf" {
		t.Errorf("Incorrect file_path: %v", chunk.Metadata["file_path"])
	}
}
