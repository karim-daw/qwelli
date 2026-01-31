package db

import (
	"context"
	"path/filepath"
	"testing"
	"time"
)

// TestInsertChunk_TextChunk verifies text chunks can be inserted without removed fields
func TestInsertChunk_TextChunk(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	pdb, err := OpenProjectDB(dbPath, 1024)
	if err != nil {
		t.Fatalf("OpenProjectDB() error = %v", err)
	}
	defer pdb.Close()

	// Insert file
	testFile := File{
		FileID:     "text-test-file",
		Path:       "/test/text.txt",
		FileType:   "txt",
		FileHash:   "hash789",
		ModifiedAt: time.Now(),
		Size:       1000,
		IndexedAt:  time.Now(),
	}

	if err := pdb.InsertFile(context.Background(), testFile); err != nil {
		t.Fatalf("InsertFile() error = %v", err)
	}

	// Insert text chunk (no StartToken, EndToken, or SourceChunkID)
	textChunk := Chunk{
		ChunkID:     "text-chunk-001",
		FileID:      testFile.FileID,
		FilePath:    testFile.Path,
		FileType:    testFile.FileType,
		ChunkIndex:  0,
		TotalChunks: 1,
		Content:     "This is a text chunk without removed fields",
		ContentType: "text",
		PageNumbers: []int{},
		ImageData:   nil,
	}

	_, err = pdb.InsertChunk(context.Background(), textChunk)
	if err != nil {
		t.Fatalf("InsertChunk() error = %v", err)
	}

	// Retrieve and verify
	retrieved, err := pdb.GetChunk(context.Background(), textChunk.ChunkID)
	if err != nil {
		t.Fatalf("GetChunk() error = %v", err)
	}

	if retrieved.Content != textChunk.Content {
		t.Errorf("Content = %q, want %q", retrieved.Content, textChunk.Content)
	}
	if retrieved.ContentType != "text" {
		t.Errorf("ContentType = %q, want %q", retrieved.ContentType, "text")
	}
	if len(retrieved.ImageData) > 0 {
		t.Error("ImageData should be empty for text chunks")
	}
}

// TestInsertChunk_DefaultContentType verifies default content type is set
func TestInsertChunk_DefaultContentType(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	pdb, err := OpenProjectDB(dbPath, 1024)
	if err != nil {
		t.Fatalf("OpenProjectDB() error = %v", err)
	}
	defer pdb.Close()

	testFile := File{
		FileID:     "default-ct-file",
		Path:       "/test/default.txt",
		FileType:   "txt",
		FileHash:   "hash999",
		ModifiedAt: time.Now(),
		Size:       500,
		IndexedAt:  time.Now(),
	}

	if err := pdb.InsertFile(context.Background(), testFile); err != nil {
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
		Content:     "Content without ContentType",
		ContentType: "", // Empty - should default to "text"
		PageNumbers: []int{},
	}

	_, err = pdb.InsertChunk(context.Background(), chunk)
	if err != nil {
		t.Fatalf("InsertChunk() error = %v", err)
	}

	retrieved, err := pdb.GetChunk(context.Background(), chunk.ChunkID)
	if err != nil {
		t.Fatalf("GetChunk() error = %v", err)
	}

	if retrieved.ContentType != "text" {
		t.Errorf("ContentType = %q, want %q", retrieved.ContentType, "text")
	}
}

// TestGetChunksForFile_MixedContentTypes verifies GetChunksForFile works with mixed content
func TestGetChunksForFile_MixedContentTypes(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	pdb, err := OpenProjectDB(dbPath, 1024)
	if err != nil {
		t.Fatalf("OpenProjectDB() error = %v", err)
	}
	defer pdb.Close()

	testFile := File{
		FileID:     "mixed-file",
		Path:       "/test/mixed.pdf",
		FileType:   "pdf",
		FileHash:   "hash-mixed",
		ModifiedAt: time.Now(),
		Size:       2000,
		IndexedAt:  time.Now(),
	}

	if err := pdb.InsertFile(context.Background(), testFile); err != nil {
		t.Fatalf("InsertFile() error = %v", err)
	}

	// Insert mixed chunks
	chunks := []Chunk{
		{
			ChunkID:     "mixed-1",
			FileID:      testFile.FileID,
			FilePath:    testFile.Path,
			FileType:    testFile.FileType,
			ChunkIndex:  0,
			TotalChunks: 3,
			Content:     "Text chunk 1",
			ContentType: "text",
			PageNumbers: []int{1},
		},
		{
			ChunkID:     "mixed-2",
			FileID:      testFile.FileID,
			FilePath:    testFile.Path,
			FileType:    testFile.FileType,
			ChunkIndex:  1,
			TotalChunks: 3,
			Content:     "Image from page 1",
			ContentType: "image",
			ImageData:   []byte("iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAYAAAAfFcSJAAAADUlEQVR42mNk+M9QDwADhgGAWjR9awAAAABJRU5ErkJggg=="),
			PageNumbers: []int{1},
		},
		{
			ChunkID:     "mixed-3",
			FileID:      testFile.FileID,
			FilePath:    testFile.Path,
			FileType:    testFile.FileType,
			ChunkIndex:  2,
			TotalChunks: 3,
			Content:     "Text chunk 2",
			ContentType: "text",
			PageNumbers: []int{2},
		},
	}

	for _, c := range chunks {
		_, err = pdb.InsertChunk(context.Background(), c)
		if err != nil {
			t.Fatalf("InsertChunk() error = %v", err)
		}
	}

	// Get all chunks for file
	fileChunks, err := pdb.GetChunksForFile(context.Background(), testFile.FileID)
	if err != nil {
		t.Fatalf("GetChunksForFile() error = %v", err)
	}

	if len(fileChunks) != 3 {
		t.Errorf("len(fileChunks) = %d, want 3", len(fileChunks))
	}

	// Verify content types
	if fileChunks[0].ContentType != "text" {
		t.Errorf("fileChunks[0].ContentType = %q, want %q", fileChunks[0].ContentType, "text")
	}
	if fileChunks[1].ContentType != "image" {
		t.Errorf("fileChunks[1].ContentType = %q, want %q", fileChunks[1].ContentType, "image")
	}
	if fileChunks[2].ContentType != "text" {
		t.Errorf("fileChunks[2].ContentType = %q, want %q", fileChunks[2].ContentType, "text")
	}

	// Verify image data is preserved
	if len(fileChunks[1].ImageData) == 0 {
		t.Error("fileChunks[1].ImageData should not be empty")
	}
}

// TestSearchANNWithFilter_ContentTypeFilter verifies content type filtering in search
func TestSearchANNWithFilter_ContentTypeFilter(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	pdb, err := OpenProjectDB(dbPath, 4)
	if err != nil {
		t.Fatalf("OpenProjectDB() error = %v", err)
	}
	defer pdb.Close()

	testFile := File{
		FileID:     "search-filter-file",
		Path:       "/test/search.pdf",
		FileType:   "pdf",
		FileHash:   "hash-search",
		ModifiedAt: time.Now(),
		Size:       1500,
		IndexedAt:  time.Now(),
	}

	if err := pdb.InsertFile(context.Background(), testFile); err != nil {
		t.Fatalf("InsertFile() error = %v", err)
	}

	// Insert text and image chunks
	textChunk := Chunk{
		ChunkID:     "search-text",
		FileID:      testFile.FileID,
		FilePath:    testFile.Path,
		FileType:    testFile.FileType,
		ChunkIndex:  0,
		TotalChunks: 2,
		Content:     "Text content for search",
		ContentType: "text",
		PageNumbers: []int{1},
	}

	imageChunk := Chunk{
		ChunkID:     "search-image",
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

	_, err = pdb.InsertChunk(context.Background(), textChunk)
	if err != nil {
		t.Fatalf("InsertChunk() text error = %v", err)
	}
	_, err = pdb.InsertChunk(context.Background(), imageChunk)
	if err != nil {
		t.Fatalf("InsertChunk() image error = %v", err)
	}

	// Insert embeddings
	if err := pdb.InsertEmbedding(context.Background(), textChunk.ChunkID, []float32{0.1, 0.2, 0.3, 0.4}); err != nil {
		t.Fatalf("InsertEmbedding() text error = %v", err)
	}
	if err := pdb.InsertEmbedding(context.Background(), imageChunk.ChunkID, []float32{0.5, 0.6, 0.7, 0.8}); err != nil {
		t.Fatalf("InsertEmbedding() image error = %v", err)
	}

	// Test filtering by text only
	queryVec := []float32{0.1, 0.2, 0.3, 0.4}
	textResults, err := pdb.SearchANNWithFilter(context.Background(), queryVec, 10, "text")
	if err != nil {
		t.Fatalf("SearchANNWithFilter() text error = %v", err)
	}

	if len(textResults) == 0 {
		t.Error("Expected at least one text result")
	}
	for _, r := range textResults {
		if r.ContentType != "text" {
			t.Errorf("Result ContentType = %q, want %q", r.ContentType, "text")
		}
	}

	// Test filtering by image only
	imageResults, err := pdb.SearchANNWithFilter(context.Background(), queryVec, 10, "image")
	if err != nil {
		t.Fatalf("SearchANNWithFilter() image error = %v", err)
	}

	if len(imageResults) == 0 {
		t.Error("Expected at least one image result")
	}
	for _, r := range imageResults {
		if r.ContentType != "image" {
			t.Errorf("Result ContentType = %q, want %q", r.ContentType, "image")
		}
	}

	// Test no filter (should return both)
	allResults, err := pdb.SearchANNWithFilter(context.Background(), queryVec, 10, "")
	if err != nil {
		t.Fatalf("SearchANNWithFilter() all error = %v", err)
	}

	if len(allResults) < 2 {
		t.Errorf("Expected at least 2 results without filter, got %d", len(allResults))
	}
}

// TestInsertChunk_EmptyPageNumbers verifies empty page numbers array works
func TestInsertChunk_EmptyPageNumbers(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	pdb, err := OpenProjectDB(dbPath, 1024)
	if err != nil {
		t.Fatalf("OpenProjectDB() error = %v", err)
	}
	defer pdb.Close()

	testFile := File{
		FileID:     "empty-pages-file",
		Path:       "/test/empty.txt",
		FileType:   "txt",
		FileHash:   "hash-empty",
		ModifiedAt: time.Now(),
		Size:       100,
		IndexedAt:  time.Now(),
	}

	if err := pdb.InsertFile(context.Background(), testFile); err != nil {
		t.Fatalf("InsertFile() error = %v", err)
	}

	chunk := Chunk{
		ChunkID:     "empty-pages-chunk",
		FileID:      testFile.FileID,
		FilePath:    testFile.Path,
		FileType:    testFile.FileType,
		ChunkIndex:  0,
		TotalChunks: 1,
		Content:     "Content with empty page numbers",
		ContentType: "text",
		PageNumbers: []int{}, // Empty array
	}

	_, err = pdb.InsertChunk(context.Background(), chunk)
	if err != nil {
		t.Fatalf("InsertChunk() error = %v", err)
	}

	retrieved, err := pdb.GetChunk(context.Background(), chunk.ChunkID)
	if err != nil {
		t.Fatalf("GetChunk() error = %v", err)
	}

	if len(retrieved.PageNumbers) != 0 {
		t.Errorf("PageNumbers = %v, want empty array", retrieved.PageNumbers)
	}
}

// TestGetChunk_NonExistent verifies error handling for non-existent chunks
func TestGetChunk_NonExistent(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	pdb, err := OpenProjectDB(dbPath, 1024)
	if err != nil {
		t.Fatalf("OpenProjectDB() error = %v", err)
	}
	defer pdb.Close()

	_, err = pdb.GetChunk(context.Background(), "non-existent-chunk-id")
	if err == nil {
		t.Error("Expected error for non-existent chunk, got nil")
	}
}
