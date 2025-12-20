package db

import (
	"path/filepath"
	"testing"
	"time"
)

func TestOpenProjectDB(t *testing.T) {
	t.Run("successful_open", func(t *testing.T) {
		tmpDir := t.TempDir()
		dbPath := filepath.Join(tmpDir, "test.db")

		pdb, err := OpenProjectDB(dbPath, 4)
		if err != nil {
			t.Fatalf("OpenProjectDB() error = %v", err)
		}
		defer pdb.Close()

		if pdb.Path != dbPath {
			t.Errorf("pdb.Path = %v, want %v", pdb.Path, dbPath)
		}
	})

	t.Run("empty_path_error", func(t *testing.T) {
		_, err := OpenProjectDB("", 4)
		if err == nil {
			t.Fatal("expected error for empty path")
		}
	})
}

func TestInsertAndGetDocument(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	pdb, err := OpenProjectDB(dbPath, 4)
	if err != nil {
		t.Fatalf("OpenProjectDB() error = %v", err)
	}
	defer pdb.Close()

	testDoc := Document{
		ID:           "test-doc-001",
		Path:         "/test/path/file.txt",
		FileType:     "txt",
		ModifiedAt:   time.Now(),
		Size:         1234,
		TextMetadata: `{"key": "value"}`,
		Content:      "This is test content.",
	}

	if err := pdb.InsertDocument(testDoc); err != nil {
		t.Fatalf("InsertDocument() error = %v", err)
	}

	retrieved, err := pdb.GetDocument(testDoc.ID)
	if err != nil {
		t.Fatalf("GetDocument() error = %v", err)
	}

	if retrieved.ID != testDoc.ID {
		t.Errorf("retrieved.ID = %v, want %v", retrieved.ID, testDoc.ID)
	}
	if retrieved.Content != testDoc.Content {
		t.Errorf("retrieved.Content = %v, want %v", retrieved.Content, testDoc.Content)
	}
}

func TestInsertEmbedding(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	pdb, err := OpenProjectDB(dbPath, 4)
	if err != nil {
		t.Fatalf("OpenProjectDB() error = %v", err)
	}
	defer pdb.Close()

	testDoc := Document{
		ID:           "embed-test-doc",
		Path:         "/test/embed.txt",
		FileType:     "txt",
		ModifiedAt:   time.Now(),
		Size:         100,
		TextMetadata: `{}`,
		Content:      "Test content",
	}

	if err := pdb.InsertDocument(testDoc); err != nil {
		t.Fatalf("InsertDocument() error = %v", err)
	}

	testEmb := Embedding{
		DocID:  testDoc.ID,
		Vector: []float32{0.1, 0.2, 0.3, 0.4},
	}

	if err := pdb.InsertEmbedding(testEmb); err != nil {
		t.Fatalf("InsertEmbedding() error = %v", err)
	}

	embeddings, err := pdb.LoadAllEmbeddings()
	if err != nil {
		t.Fatalf("LoadAllEmbeddings() error = %v", err)
	}

	if len(embeddings) != 1 {
		t.Fatalf("len(embeddings) = %d, want 1", len(embeddings))
	}

	if embeddings[0].DocID != testEmb.DocID {
		t.Errorf("DocID = %v, want %v", embeddings[0].DocID, testEmb.DocID)
	}
}

func TestBuildHNSWIndex(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	pdb, err := OpenProjectDB(dbPath, 4)
	if err != nil {
		t.Fatalf("OpenProjectDB() error = %v", err)
	}
	defer pdb.Close()

	docs := []Document{
		{ID: "doc-1", Path: "/1.txt", FileType: "txt", ModifiedAt: time.Now(), Size: 100, TextMetadata: `{}`, Content: "Content 1"},
		{ID: "doc-2", Path: "/2.txt", FileType: "txt", ModifiedAt: time.Now(), Size: 200, TextMetadata: `{}`, Content: "Content 2"},
	}

	embeddings := []Embedding{
		{DocID: "doc-1", Vector: []float32{1.0, 0.0, 0.0, 0.0}},
		{DocID: "doc-2", Vector: []float32{0.0, 1.0, 0.0, 0.0}},
	}

	for _, doc := range docs {
		pdb.InsertDocument(doc)
	}
	for _, emb := range embeddings {
		pdb.InsertEmbedding(emb)
	}

	if err := pdb.BuildHNSWIndex(); err != nil {
		t.Fatalf("BuildHNSWIndex() error = %v", err)
	}

	// Should be idempotent
	if err := pdb.BuildHNSWIndex(); err != nil {
		t.Fatalf("BuildHNSWIndex() second call error = %v", err)
	}
}

func TestSearchANN(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	pdb, err := OpenProjectDB(dbPath, 4)
	if err != nil {
		t.Fatalf("OpenProjectDB() error = %v", err)
	}
	defer pdb.Close()

	docs := []Document{
		{ID: "doc-1", Path: "/1.txt", FileType: "txt", ModifiedAt: time.Now(), Size: 100, TextMetadata: `{}`, Content: "Cats"},
		{ID: "doc-2", Path: "/2.txt", FileType: "txt", ModifiedAt: time.Now(), Size: 200, TextMetadata: `{}`, Content: "Dogs"},
		{ID: "doc-3", Path: "/3.txt", FileType: "txt", ModifiedAt: time.Now(), Size: 300, TextMetadata: `{}`, Content: "Birds"},
	}

	embeddings := []Embedding{
		{DocID: "doc-1", Vector: []float32{1.0, 0.0, 0.0, 0.0}},
		{DocID: "doc-2", Vector: []float32{0.0, 1.0, 0.0, 0.0}},
		{DocID: "doc-3", Vector: []float32{0.0, 0.0, 1.0, 0.0}},
	}

	for _, doc := range docs {
		pdb.InsertDocument(doc)
	}
	for _, emb := range embeddings {
		pdb.InsertEmbedding(emb)
	}
	pdb.BuildHNSWIndex()

	t.Run("finds_closest", func(t *testing.T) {
		results, err := pdb.SearchANN([]float32{0.9, 0.1, 0.0, 0.0}, 3)
		if err != nil {
			t.Fatalf("SearchANN() error = %v", err)
		}

		if len(results) != 3 {
			t.Fatalf("len(results) = %d, want 3", len(results))
		}

		if results[0].DocID != "doc-1" {
			t.Errorf("results[0].DocID = %v, want doc-1", results[0].DocID)
		}
	})

	t.Run("limit_k", func(t *testing.T) {
		results, err := pdb.SearchANN([]float32{0.5, 0.5, 0.5, 0.0}, 1)
		if err != nil {
			t.Fatalf("SearchANN() error = %v", err)
		}

		if len(results) != 1 {
			t.Fatalf("len(results) = %d, want 1", len(results))
		}
	})

	t.Run("exact_match_distance_zero", func(t *testing.T) {
		results, err := pdb.SearchANN([]float32{0.0, 1.0, 0.0, 0.0}, 1)
		if err != nil {
			t.Fatalf("SearchANN() error = %v", err)
		}

		if results[0].DocID != "doc-2" {
			t.Errorf("results[0].DocID = %v, want doc-2", results[0].DocID)
		}

		if results[0].Distance > 0.001 {
			t.Errorf("Distance = %v, expected ~0", results[0].Distance)
		}
	})
}

func TestSearchANN_EmptyDatabase(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	pdb, err := OpenProjectDB(dbPath, 4)
	if err != nil {
		t.Fatalf("OpenProjectDB() error = %v", err)
	}
	defer pdb.Close()

	pdb.BuildHNSWIndex()

	results, err := pdb.SearchANN([]float32{1.0, 0.0, 0.0, 0.0}, 3)
	if err != nil {
		t.Fatalf("SearchANN() error = %v", err)
	}

	if len(results) != 0 {
		t.Errorf("len(results) = %d, want 0", len(results))
	}
}
