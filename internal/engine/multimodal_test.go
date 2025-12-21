package engine

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"
	"time"

	"github.com/karim-daw/qwelli/internal/db"
	"github.com/karim-daw/qwelli/internal/indexer"
	"github.com/karim-daw/qwelli/internal/processor"
)

func TestProcessPDFFileMultimodal_TextAndImages(t *testing.T) {
	// Create a mock embedder
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			Inputs []interface{} `json:"inputs"`
		}
		json.NewDecoder(r.Body).Decode(&req)

		embeddings := make([][]float64, len(req.Inputs))
		for i := range embeddings {
			embeddings[i] = make([]float64, 1024)
		}

		response := struct {
			Embeddings [][]float64 `json:"embeddings"`
		}{
			Embeddings: embeddings,
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	embedder, err := indexer.NewEmbedder("voyage", "test-key", "voyage-multimodal-3", server.URL)
	if err != nil {
		t.Fatalf("NewEmbedder() error = %v", err)
	}

	// Note: This test would require actual PDF processing
	// For now, we'll test the multimodal chunk creation logic
	tmpDir := t.TempDir()
	testFilePath := filepath.Join(tmpDir, "test.pdf")

	pdfChunker := processor.NewPDFChunker(processor.ChunkerConfig{
		ChunkSize:   300,
		OverlapSize: 10,
	})
	imageExtractor := processor.NewImageExtractor(1024, 1024)
	multimodalChunker := processor.NewMultimodalChunker(pdfChunker, imageExtractor)

	pages := []processor.PDFPage{
		{PageNumber: 1, Text: "Page one text content."},
		{PageNumber: 2, Text: "Page two text content."},
	}

	images := []processor.PDFImage{
		{
			Data:       []byte("image1"),
			Format:     "jpeg",
			Width:      800,
			Height:     600,
			PageNumber: 1,
			Base64:     "aW1hZ2Ux",
		},
	}

	metadata := &processor.PDFMetadata{
		PageCount: 2,
		Title:     "Test PDF",
	}

	multimodalChunks, err := multimodalChunker.ChunkPDF(pages, images, metadata, testFilePath)
	if err != nil {
		t.Fatalf("ChunkPDF() error = %v", err)
	}

	// Verify we have both text and image chunks
	textCount := 0
	imageCount := 0
	for _, chunk := range multimodalChunks {
		switch chunk.ContentType {
		case "text":
			textCount++
		case "image":
			imageCount++
		}
	}

	if textCount == 0 {
		t.Error("processPDFFileMultimodal() returned no text chunks")
	}
	if imageCount == 0 {
		t.Error("processPDFFileMultimodal() returned no image chunks")
	}

	// Verify embedder can handle multimodal
	if !embedder.IsMultimodal() {
		t.Fatal("Embedder should support multimodal for Voyage")
	}
}

func TestEngine_SearchWithContentTypeFilter(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	// Create test database with a fixed dimension (Voyage multimodal-3 uses 1024)
	dimension := 1024
	projectDB, err := db.OpenProjectDB(dbPath, dimension)
	if err != nil {
		t.Fatalf("OpenProjectDB() error = %v", err)
	}
	defer projectDB.Close()

	// Insert test file
	testFile := db.File{
		FileID:     "search-test-file",
		Path:       "/test/search.pdf",
		FileType:   "pdf",
		FileHash:   "hash789",
		ModifiedAt: time.Now(),
		Size:       2000,
		IndexedAt:  time.Now(),
	}

	if err := projectDB.InsertFile(testFile); err != nil {
		t.Fatalf("InsertFile() error = %v", err)
	}

	// Create test embeddings
	textVector := make([]float32, dimension)
	imageVector := make([]float32, dimension)
	for i := range textVector {
		textVector[i] = 0.1
	}
	for i := range imageVector {
		imageVector[i] = 0.2
	}

	// Insert text chunk
	textChunk := db.Chunk{
		ChunkID:     "search-text-1",
		FileID:      testFile.FileID,
		FilePath:    testFile.Path,
		FileType:    testFile.FileType,
		ChunkIndex:  0,
		TotalChunks: 2,
		Content:     "Searchable text",
		ContentType: "text",
		PageNumbers: []int{1},
	}

	if err := projectDB.InsertChunk(textChunk); err != nil {
		t.Fatalf("InsertChunk() text error = %v", err)
	}

	if err := projectDB.InsertEmbedding(db.Embedding{
		ChunkID: textChunk.ChunkID,
		Vector:  textVector,
	}); err != nil {
		t.Fatalf("InsertEmbedding() text error = %v", err)
	}

	// Insert image chunk
	imageChunk := db.Chunk{
		ChunkID:     "search-image-1",
		FileID:      testFile.FileID,
		FilePath:    testFile.Path,
		FileType:    testFile.FileType,
		ChunkIndex:  1,
		TotalChunks: 2,
		Content:     "Image content",
		ContentType: "image",
		ImageData:   []byte("base64data"),
		PageNumbers: []int{1},
	}

	if err := projectDB.InsertChunk(imageChunk); err != nil {
		t.Fatalf("InsertChunk() image error = %v", err)
	}

	if err := projectDB.InsertEmbedding(db.Embedding{
		ChunkID: imageChunk.ChunkID,
		Vector:  imageVector,
	}); err != nil {
		t.Fatalf("InsertEmbedding() image error = %v", err)
	}

	// Build index
	if err := projectDB.BuildHNSWIndex(); err != nil {
		t.Fatalf("BuildHNSWIndex() error = %v", err)
	}

	// Test search with text filter
	queryVector := make([]float32, dimension)
	for i := range queryVector {
		queryVector[i] = 0.15
	}

	textResults, err := projectDB.SearchANNWithFilter(queryVector, 10, "text")
	if err != nil {
		t.Fatalf("SearchANNWithFilter() text error = %v", err)
	}

	if len(textResults) != 1 {
		t.Errorf("SearchANNWithFilter() text returned %d results, want 1", len(textResults))
	}

	if len(textResults) > 0 && textResults[0].ContentType != "text" {
		t.Errorf("SearchANNWithFilter() text returned result with ContentType = %q", textResults[0].ContentType)
	}

	// Test search with image filter
	imageResults, err := projectDB.SearchANNWithFilter(queryVector, 10, "image")
	if err != nil {
		t.Fatalf("SearchANNWithFilter() image error = %v", err)
	}

	if len(imageResults) != 1 {
		t.Errorf("SearchANNWithFilter() image returned %d results, want 1", len(imageResults))
	}

	if len(imageResults) > 0 && imageResults[0].ContentType != "image" {
		t.Errorf("SearchANNWithFilter() image returned result with ContentType = %q", imageResults[0].ContentType)
	}
}
