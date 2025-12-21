package db

import (
	"path/filepath"
	"testing"
	"time"
)

func TestInsertChunk_WithImageData(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	pdb, err := OpenProjectDB(dbPath, 1024) // Voyage uses 1024 dimensions
	if err != nil {
		t.Fatalf("OpenProjectDB() error = %v", err)
	}
	defer pdb.Close()

	// Insert file
	testFile := File{
		FileID:     "image-test-file",
		Path:       "/test/image.pdf",
		FileType:   "pdf",
		FileHash:   "hash123",
		ModifiedAt: time.Now(),
		Size:       5000,
		IndexedAt:  time.Now(),
	}

	if err := pdb.InsertFile(testFile); err != nil {
		t.Fatalf("InsertFile() error = %v", err)
	}

	// Insert text chunk
	textChunk := Chunk{
		ChunkID:     "text-chunk-001",
		FileID:      testFile.FileID,
		FilePath:    testFile.Path,
		FileType:    testFile.FileType,
		ChunkIndex:  0,
		TotalChunks: 2,
		Content:     "This is text content",
		ContentType: "text",
		PageNumbers: []int{1},
	}

	if err := pdb.InsertChunk(textChunk); err != nil {
		t.Fatalf("InsertChunk() text chunk error = %v", err)
	}

	// Insert image chunk
	imageBase64 := "iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAYAAAAfFcSJAAAADUlEQVR42mNk+M9QDwADhgGAWjR9awAAAABJRU5ErkJggg=="
	imageChunk := Chunk{
		ChunkID:     "image-chunk-001",
		FileID:      testFile.FileID,
		FilePath:    testFile.Path,
		FileType:    testFile.FileType,
		ChunkIndex:  1,
		TotalChunks: 2,
		Content:     "Image from page 1 (800x600, jpeg)",
		ContentType: "image",
		ImageData:   []byte(imageBase64),
		PageNumbers: []int{1},
	}

	if err := pdb.InsertChunk(imageChunk); err != nil {
		t.Fatalf("InsertChunk() image chunk error = %v", err)
	}

	// Retrieve text chunk
	retrievedText, err := pdb.GetChunk(textChunk.ChunkID)
	if err != nil {
		t.Fatalf("GetChunk() text chunk error = %v", err)
	}

	if retrievedText.ContentType != "text" {
		t.Errorf("GetChunk() text chunk ContentType = %q, want %q", retrievedText.ContentType, "text")
	}
	if retrievedText.Content != textChunk.Content {
		t.Errorf("GetChunk() text chunk Content = %q, want %q", retrievedText.Content, textChunk.Content)
	}

	// Retrieve image chunk
	retrievedImage, err := pdb.GetChunk(imageChunk.ChunkID)
	if err != nil {
		t.Fatalf("GetChunk() image chunk error = %v", err)
	}

	if retrievedImage.ContentType != "image" {
		t.Errorf("GetChunk() image chunk ContentType = %q, want %q", retrievedImage.ContentType, "image")
	}
	if len(retrievedImage.ImageData) == 0 {
		t.Error("GetChunk() image chunk ImageData is empty")
	}
	if string(retrievedImage.ImageData) != imageBase64 {
		t.Errorf("GetChunk() image chunk ImageData = %q, want %q", string(retrievedImage.ImageData), imageBase64)
	}
}

func TestGetChunksByType(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	pdb, err := OpenProjectDB(dbPath, 1024)
	if err != nil {
		t.Fatalf("OpenProjectDB() error = %v", err)
	}
	defer pdb.Close()

	// Insert file
	testFile := File{
		FileID:     "filter-test-file",
		Path:       "/test/filter.pdf",
		FileType:   "pdf",
		FileHash:   "hash456",
		ModifiedAt: time.Now(),
		Size:       3000,
		IndexedAt:  time.Now(),
	}

	if err := pdb.InsertFile(testFile); err != nil {
		t.Fatalf("InsertFile() error = %v", err)
	}

	// Insert multiple chunks of different types
	chunks := []Chunk{
		{
			ChunkID:     "text-1",
			FileID:      testFile.FileID,
			FilePath:    testFile.Path,
			FileType:    testFile.FileType,
			ChunkIndex:  0,
			TotalChunks: 4,
			Content:     "Text chunk 1",
			ContentType: "text",
			PageNumbers: []int{1},
		},
		{
			ChunkID:     "image-1",
			FileID:      testFile.FileID,
			FilePath:    testFile.Path,
			FileType:    testFile.FileType,
			ChunkIndex:  1,
			TotalChunks: 4,
			Content:     "Image chunk 1",
			ContentType: "image",
			ImageData:   []byte("base64image1"),
			PageNumbers: []int{1},
		},
		{
			ChunkID:     "text-2",
			FileID:      testFile.FileID,
			FilePath:    testFile.Path,
			FileType:    testFile.FileType,
			ChunkIndex:  2,
			TotalChunks: 4,
			Content:     "Text chunk 2",
			ContentType: "text",
			PageNumbers: []int{2},
		},
		{
			ChunkID:     "image-2",
			FileID:      testFile.FileID,
			FilePath:    testFile.Path,
			FileType:    testFile.FileType,
			ChunkIndex:  3,
			TotalChunks: 4,
			Content:     "Image chunk 2",
			ContentType: "image",
			ImageData:   []byte("base64image2"),
			PageNumbers: []int{2},
		},
	}

	for _, chunk := range chunks {
		if err := pdb.InsertChunk(chunk); err != nil {
			t.Fatalf("InsertChunk() error = %v", err)
		}
	}

	// Test filtering by text
	textChunks, err := pdb.GetChunksByType("text", testFile.FileID)
	if err != nil {
		t.Fatalf("GetChunksByType() text error = %v", err)
	}

	if len(textChunks) != 2 {
		t.Errorf("GetChunksByType() text returned %d chunks, want 2", len(textChunks))
	}

	for _, chunk := range textChunks {
		if chunk.ContentType != "text" {
			t.Errorf("GetChunksByType() text returned chunk with ContentType = %q", chunk.ContentType)
		}
	}

	// Test filtering by image
	imageChunks, err := pdb.GetChunksByType("image", testFile.FileID)
	if err != nil {
		t.Fatalf("GetChunksByType() image error = %v", err)
	}

	if len(imageChunks) != 2 {
		t.Errorf("GetChunksByType() image returned %d chunks, want 2", len(imageChunks))
	}

	for _, chunk := range imageChunks {
		if chunk.ContentType != "image" {
			t.Errorf("GetChunksByType() image returned chunk with ContentType = %q", chunk.ContentType)
		}
		if len(chunk.ImageData) == 0 {
			t.Error("GetChunksByType() image returned chunk with empty ImageData")
		}
	}
}

func TestSearchANNWithFilter(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	pdb, err := OpenProjectDB(dbPath, 1024)
	if err != nil {
		t.Fatalf("OpenProjectDB() error = %v", err)
	}
	defer pdb.Close()

	// Insert file
	testFile := File{
		FileID:     "search-test-file",
		Path:       "/test/search.pdf",
		FileType:   "pdf",
		FileHash:   "hash789",
		ModifiedAt: time.Now(),
		Size:       2000,
		IndexedAt:  time.Now(),
	}

	if err := pdb.InsertFile(testFile); err != nil {
		t.Fatalf("InsertFile() error = %v", err)
	}

	// Create test embeddings
	textVector := make([]float32, 1024)
	imageVector := make([]float32, 1024)
	for i := range textVector {
		textVector[i] = 0.1
	}
	for i := range imageVector {
		imageVector[i] = 0.2
	}

	// Insert text chunk with embedding
	textChunk := Chunk{
		ChunkID:     "search-text-1",
		FileID:      testFile.FileID,
		FilePath:    testFile.Path,
		FileType:    testFile.FileType,
		ChunkIndex:  0,
		TotalChunks: 2,
		Content:     "Searchable text content",
		ContentType: "text",
		PageNumbers: []int{1},
	}

	if err := pdb.InsertChunk(textChunk); err != nil {
		t.Fatalf("InsertChunk() text error = %v", err)
	}

	if err := pdb.InsertEmbedding(Embedding{
		ChunkID: textChunk.ChunkID,
		Vector:  textVector,
	}); err != nil {
		t.Fatalf("InsertEmbedding() text error = %v", err)
	}

	// Insert image chunk with embedding
	imageChunk := Chunk{
		ChunkID:     "search-image-1",
		FileID:      testFile.FileID,
		FilePath:    testFile.Path,
		FileType:    testFile.FileType,
		ChunkIndex:  1,
		TotalChunks: 2,
		Content:     "Image content",
		ContentType: "image",
		ImageData:   []byte("base64imagedata"),
		PageNumbers: []int{1},
	}

	if err := pdb.InsertChunk(imageChunk); err != nil {
		t.Fatalf("InsertChunk() image error = %v", err)
	}

	if err := pdb.InsertEmbedding(Embedding{
		ChunkID: imageChunk.ChunkID,
		Vector:  imageVector,
	}); err != nil {
		t.Fatalf("InsertEmbedding() image error = %v", err)
	}

	// Build HNSW index
	if err := pdb.BuildHNSWIndex(); err != nil {
		t.Fatalf("BuildHNSWIndex() error = %v", err)
	}

	// Test search without filter (should return both)
	queryVector := make([]float32, 1024)
	for i := range queryVector {
		queryVector[i] = 0.15 // Between text and image vectors
	}

	allResults, err := pdb.SearchANN(queryVector, 10)
	if err != nil {
		t.Fatalf("SearchANN() error = %v", err)
	}

	if len(allResults) != 2 {
		t.Errorf("SearchANN() returned %d results, want 2", len(allResults))
	}

	// Test search with text filter
	textResults, err := pdb.SearchANNWithFilter(queryVector, 10, "text")
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
	imageResults, err := pdb.SearchANNWithFilter(queryVector, 10, "image")
	if err != nil {
		t.Fatalf("SearchANNWithFilter() image error = %v", err)
	}

	if len(imageResults) != 1 {
		t.Errorf("SearchANNWithFilter() image returned %d results, want 1", len(imageResults))
	}

	if len(imageResults) > 0 {
		if imageResults[0].ContentType != "image" {
			t.Errorf("SearchANNWithFilter() image returned result with ContentType = %q", imageResults[0].ContentType)
		}
		if len(imageResults[0].ImageData) == 0 {
			t.Error("SearchANNWithFilter() image returned result with empty ImageData")
		}
	}
}

func TestChunk_DefaultContentType(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	pdb, err := OpenProjectDB(dbPath, 1024)
	if err != nil {
		t.Fatalf("OpenProjectDB() error = %v", err)
	}
	defer pdb.Close()

	// Insert file
	testFile := File{
		FileID:     "default-type-file",
		Path:       "/test/default.txt",
		FileType:   "txt",
		FileHash:   "hash999",
		ModifiedAt: time.Now(),
		Size:       100,
		IndexedAt:  time.Now(),
	}

	if err := pdb.InsertFile(testFile); err != nil {
		t.Fatalf("InsertFile() error = %v", err)
	}

	// Insert chunk without ContentType (should default to "text")
	chunk := Chunk{
		ChunkID:     "default-chunk",
		FileID:      testFile.FileID,
		FilePath:    testFile.Path,
		FileType:    testFile.FileType,
		ChunkIndex:  0,
		TotalChunks: 1,
		Content:     "Test content",
		// ContentType not set - should default to "text"
		PageNumbers: []int{},
	}

	if err := pdb.InsertChunk(chunk); err != nil {
		t.Fatalf("InsertChunk() error = %v", err)
	}

	// Retrieve and verify default
	retrieved, err := pdb.GetChunk(chunk.ChunkID)
	if err != nil {
		t.Fatalf("GetChunk() error = %v", err)
	}

	if retrieved.ContentType != "text" {
		t.Errorf("GetChunk() ContentType = %q, want %q (default)", retrieved.ContentType, "text")
	}
}
