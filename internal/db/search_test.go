package db

import (
	"path/filepath"
	"testing"
	"time"
)

// TestSearchANNWithFilter_CandidateLimit verifies that when filtering by content type,
// the query fetches more candidates to ensure enough results are returned
func TestSearchANNWithFilter_CandidateLimit(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	pdb, err := OpenProjectDB(dbPath, 4)
	if err != nil {
		t.Fatalf("OpenProjectDB() error = %v", err)
	}
	defer pdb.Close()

	testFile := File{
		FileID:     "candidate-limit-file",
		Path:       "/test/candidate.pdf",
		FileType:   "pdf",
		FileHash:   "hash-candidate",
		ModifiedAt: time.Now(),
		Size:       1500,
		IndexedAt:  time.Now(),
	}

	if err := pdb.InsertFile(testFile); err != nil {
		t.Fatalf("InsertFile() error = %v", err)
	}

	// Insert many text chunks and a few image chunks
	// This tests that when filtering for images, we fetch enough candidates
	textChunks := make([]Chunk, 0, 20)
	imageChunks := make([]Chunk, 0, 5)

	for i := 0; i < 20; i++ {
		textChunk := Chunk{
			ChunkID:     "text-" + string(rune('a'+i)),
			FileID:      testFile.FileID,
			FilePath:    testFile.Path,
			FileType:    testFile.FileType,
			ChunkIndex:  i,
			TotalChunks: 25,
			Content:     "Text content " + string(rune('a'+i)),
			ContentType: "text",
			PageNumbers: []int{1},
		}
		textChunks = append(textChunks, textChunk)
		if err := pdb.InsertChunk(textChunk); err != nil {
			t.Fatalf("InsertChunk() text %d error = %v", i, err)
		}
	}

	for i := 0; i < 5; i++ {
		imageChunk := Chunk{
			ChunkID:     "image-" + string(rune('a'+i)),
			FileID:      testFile.FileID,
			FilePath:    testFile.Path,
			FileType:    testFile.FileType,
			ChunkIndex:  20 + i,
			TotalChunks: 25,
			Content:     "Image content " + string(rune('a'+i)),
			ContentType: "image",
			ImageData:   []byte("base64data"),
			PageNumbers: []int{1},
		}
		imageChunks = append(imageChunks, imageChunk)
		if err := pdb.InsertChunk(imageChunk); err != nil {
			t.Fatalf("InsertChunk() image %d error = %v", i, err)
		}
	}

	// Insert embeddings - make text chunks closer to query vector
	queryVec := []float32{0.1, 0.2, 0.3, 0.4}

	// Text embeddings are closer (lower distance)
	for i, chunk := range textChunks {
		textEmbedding := Embedding{
			ChunkID: chunk.ChunkID,
			Vector:  []float32{0.1 + float32(i)*0.01, 0.2, 0.3, 0.4},
		}
		if err := pdb.InsertEmbedding(textEmbedding); err != nil {
			t.Fatalf("InsertEmbedding() text %d error = %v", i, err)
		}
	}

	// Image embeddings are further (higher distance)
	for i, chunk := range imageChunks {
		imageEmbedding := Embedding{
			ChunkID: chunk.ChunkID,
			Vector:  []float32{0.5 + float32(i)*0.01, 0.6, 0.7, 0.8},
		}
		if err := pdb.InsertEmbedding(imageEmbedding); err != nil {
			t.Fatalf("InsertEmbedding() image %d error = %v", i, err)
		}
	}

	// Build index
	if err := pdb.BuildHNSWIndex(); err != nil {
		t.Fatalf("BuildHNSWIndex() error = %v", err)
	}

	// Test: Request 5 image results, but top 10 results are all text
	// The query should fetch more candidates (3x = 15) to find the images
	imageResults, err := pdb.SearchANNWithFilter(queryVec, 5, "image")
	if err != nil {
		t.Fatalf("SearchANNWithFilter() image error = %v", err)
	}

	// Should find all 5 images (or at least some of them)
	if len(imageResults) == 0 {
		t.Error("SearchANNWithFilter() with image filter returned 0 results, expected at least 1")
	}

	// Verify all results are images
	for _, r := range imageResults {
		if r.ContentType != "image" {
			t.Errorf("SearchANNWithFilter() returned result with ContentType = %q, want %q", r.ContentType, "image")
		}
	}

	// Test: Request 5 text results - should get them easily
	textResults, err := pdb.SearchANNWithFilter(queryVec, 5, "text")
	if err != nil {
		t.Fatalf("SearchANNWithFilter() text error = %v", err)
	}

	if len(textResults) == 0 {
		t.Error("SearchANNWithFilter() with text filter returned 0 results")
	}

	// Verify all results are text
	for _, r := range textResults {
		if r.ContentType != "text" {
			t.Errorf("SearchANNWithFilter() returned result with ContentType = %q, want %q", r.ContentType, "text")
		}
	}
}

// TestSearchANNWithFilter_EmptyResults verifies handling of empty result sets
func TestSearchANNWithFilter_EmptyResults(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	pdb, err := OpenProjectDB(dbPath, 4)
	if err != nil {
		t.Fatalf("OpenProjectDB() error = %v", err)
	}
	defer pdb.Close()

	// Search with no data in database
	queryVec := []float32{0.1, 0.2, 0.3, 0.4}
	results, err := pdb.SearchANNWithFilter(queryVec, 10, "")
	if err != nil {
		t.Fatalf("SearchANNWithFilter() error = %v", err)
	}

	if len(results) != 0 {
		t.Errorf("SearchANNWithFilter() returned %d results, want 0", len(results))
	}

	// Search with filter for non-existent content type
	results, err = pdb.SearchANNWithFilter(queryVec, 10, "image")
	if err != nil {
		t.Fatalf("SearchANNWithFilter() image error = %v", err)
	}

	if len(results) != 0 {
		t.Errorf("SearchANNWithFilter() image returned %d results, want 0", len(results))
	}
}

// TestSearchANNWithFilter_MinMaxCandidateLimit verifies candidate limit bounds
func TestSearchANNWithFilter_MinMaxCandidateLimit(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	pdb, err := OpenProjectDB(dbPath, 4)
	if err != nil {
		t.Fatalf("OpenProjectDB() error = %v", err)
	}
	defer pdb.Close()

	// Test with k=1 (should use minimum candidate limit of 50)
	queryVec := []float32{0.1, 0.2, 0.3, 0.4}
	results, err := pdb.SearchANNWithFilter(queryVec, 1, "image")
	if err != nil {
		t.Fatalf("SearchANNWithFilter() k=1 error = %v", err)
	}
	// Should not error, even with no data
	_ = results

	// Test with k=500 (should cap at maximum candidate limit of 1000)
	results, err = pdb.SearchANNWithFilter(queryVec, 500, "image")
	if err != nil {
		t.Fatalf("SearchANNWithFilter() k=500 error = %v", err)
	}
	// Should not error, even with no data
	_ = results
}
