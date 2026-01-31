package db

import (
	"context"
	"path/filepath"
	"testing"
	"time"
)

func TestSearchFTS(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test_fts.db")

	pdb, err := OpenProjectDB(dbPath, 4)
	if err != nil {
		t.Fatalf("OpenProjectDB() error = %v", err)
	}
	defer pdb.Close()

	// Insert test data
	file := File{
		FileID:     "file-1",
		Path:       "/test1.txt",
		FileType:   "txt",
		FileHash:   "hash1",
		ModifiedAt: time.Now(),
		Size:       100,
		IndexedAt:  time.Now(),
	}
	if err := pdb.InsertFile(context.Background(), file); err != nil {
		t.Fatalf("InsertFile() error = %v", err)
	}

	chunks := []Chunk{
		{
			ChunkID:     "c1",
			FileID:      "file-1",
			FilePath:    "/test1.txt",
			FileType:    "txt",
			ChunkIndex:  0,
			TotalChunks: 2,
			Content:     "This is a test document about machine learning and artificial intelligence.",
			PageNumbers: []int{},
			ContentType: "text",
		},
		{
			ChunkID:     "c2",
			FileID:      "file-1",
			FilePath:    "/test1.txt",
			FileType:    "txt",
			ChunkIndex:  1,
			TotalChunks: 2,
			Content:     "Another chunk discussing neural networks and deep learning algorithms.",
			PageNumbers: []int{},
			ContentType: "text",
		},
		{
			ChunkID:     "c3",
			FileID:      "file-1",
			FilePath:    "/test1.txt",
			FileType:    "txt",
			ChunkIndex:  2,
			TotalChunks: 2,
			Content:     "Random text about cooking recipes and food preparation.",
			PageNumbers: []int{},
			ContentType: "text",
		},
	}

	for _, c := range chunks {
		_, err = pdb.InsertChunk(context.Background(), c)
		if err != nil {
			t.Fatalf("InsertChunk() error = %v", err)
		}
	}

	// Test 1: Search for "machine learning"
	results, err := pdb.SearchFTS(context.Background(), "machine learning", 5, "")
	if err != nil {
		t.Fatalf("SearchFTS() error = %v", err)
	}

	if len(results) == 0 {
		t.Error("SearchFTS() returned 0 results, expected at least 1")
	}

	// First result should contain "machine learning"
	found := false
	for _, r := range results {
		if r.ChunkID == "c1" {
			found = true
			break
		}
	}
	if !found {
		t.Error("SearchFTS() did not return expected chunk with 'machine learning'")
	}

	// Test 2: Search for "neural networks"
	results, err = pdb.SearchFTS(context.Background(), "neural networks", 5, "")
	if err != nil {
		t.Fatalf("SearchFTS() neural networks error = %v", err)
	}

	if len(results) == 0 {
		t.Error("SearchFTS() neural networks returned 0 results")
	}

	// Test 3: Search with content type filter
	results, err = pdb.SearchFTS(context.Background(), "learning", 5, "text")
	if err != nil {
		t.Fatalf("SearchFTS() with filter error = %v", err)
	}

	// All results should be text type
	for _, r := range results {
		if r.ContentType != "text" {
			t.Errorf("SearchFTS() returned result with ContentType = %q, want %q", r.ContentType, "text")
		}
	}

	// Test 4: Empty query
	results, err = pdb.SearchFTS(context.Background(), "", 5, "")
	if err != nil {
		t.Fatalf("SearchFTS() empty query error = %v", err)
	}
	if len(results) != 0 {
		t.Errorf("SearchFTS() empty query returned %d results, want 0", len(results))
	}

	// Test 5: Query with only stop words
	results, err = pdb.SearchFTS(context.Background(), "the and or", 5, "")
	if err != nil {
		t.Fatalf("SearchFTS() stop words error = %v", err)
	}
	if len(results) != 0 {
		t.Errorf("SearchFTS() stop words returned %d results, want 0", len(results))
	}
}

func TestTokenizeQuery(t *testing.T) {
	// This tests the tokenizeQuery function indirectly through SearchFTS
	// We can't test it directly as it's not exported, but we can verify behavior

	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test_tokenize.db")

	pdb, err := OpenProjectDB(dbPath, 4)
	if err != nil {
		t.Fatalf("OpenProjectDB() error = %v", err)
	}
	defer pdb.Close()

	file := File{
		FileID:     "file-1",
		Path:       "/test.txt",
		FileType:   "txt",
		FileHash:   "hash1",
		ModifiedAt: time.Now(),
		Size:       100,
		IndexedAt:  time.Now(),
	}
	pdb.InsertFile(context.Background(), file)

	chunk := Chunk{
		ChunkID:     "c1",
		FileID:      "file-1",
		FilePath:    "/test.txt",
		FileType:    "txt",
		ChunkIndex:  0,
		TotalChunks: 1,
		Content:     "Python programming language tutorial",
		PageNumbers: []int{},
		ContentType: "text",
	}
	pdb.InsertChunk(context.Background(), chunk)

	// Test that punctuation is handled
	results, err := pdb.SearchFTS(context.Background(), "python, programming!", 5, "")
	if err != nil {
		t.Fatalf("SearchFTS() with punctuation error = %v", err)
	}
	if len(results) == 0 {
		t.Error("SearchFTS() with punctuation returned 0 results, expected 1")
	}

	// Test case insensitivity
	results, err = pdb.SearchFTS(context.Background(), "PYTHON PROGRAMMING", 5, "")
	if err != nil {
		t.Fatalf("SearchFTS() uppercase error = %v", err)
	}
	if len(results) == 0 {
		t.Error("SearchFTS() uppercase returned 0 results, expected 1")
	}
}
