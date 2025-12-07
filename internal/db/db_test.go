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

		pdb, err := OpenProjectDB(dbPath)
		if err != nil {
			t.Fatalf("OpenProjectDB() error = %v, want nil", err)
		}
		defer pdb.Close()

		if pdb.Path != dbPath {
			t.Errorf("pdb.Path = %v, want %v", pdb.Path, dbPath)
		}

		if pdb.conn == nil {
			t.Error("pdb.conn is nil, expected a valid connection")
		}
	})

	t.Run("empty_path_error", func(t *testing.T) {
		_, err := OpenProjectDB("")
		if err == nil {
			t.Fatal("OpenProjectDB(\"\") expected error, got nil")
		}
	})
}

func TestInsertAndGetDocument(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	pdb, err := OpenProjectDB(dbPath)
	if err != nil {
		t.Fatalf("OpenProjectDB() error = %v", err)
	}
	defer pdb.Close()

	// Create test document
	testDoc := Document{
		ID:         "test-doc-001",
		Path:       "/test/path/file.txt",
		FileType:   "txt",
		ModifiedAt: time.Now().Truncate(time.Microsecond),
		Size:       1234,
		Metadata:   `{"key": "value"}`,
		Content:    "This is test content for the document.",
	}

	t.Run("insert_document", func(t *testing.T) {
		err := pdb.InsertDocument(testDoc)
		if err != nil {
			t.Fatalf("InsertDocument() error = %v", err)
		}
	})

	t.Run("get_document", func(t *testing.T) {
		retrieved, err := pdb.GetDocument(testDoc.ID)
		if err != nil {
			t.Fatalf("GetDocument() error = %v", err)
		}

		if retrieved.ID != testDoc.ID {
			t.Errorf("retrieved.ID = %v, want %v", retrieved.ID, testDoc.ID)
		}
		if retrieved.Path != testDoc.Path {
			t.Errorf("retrieved.Path = %v, want %v", retrieved.Path, testDoc.Path)
		}
		if retrieved.FileType != testDoc.FileType {
			t.Errorf("retrieved.FileType = %v, want %v", retrieved.FileType, testDoc.FileType)
		}
		if retrieved.Size != testDoc.Size {
			t.Errorf("retrieved.Size = %v, want %v", retrieved.Size, testDoc.Size)
		}
		if retrieved.Content != testDoc.Content {
			t.Errorf("retrieved.Content = %v, want %v", retrieved.Content, testDoc.Content)
		}
	})

	t.Run("get_nonexistent_document", func(t *testing.T) {
		_, err := pdb.GetDocument("nonexistent-id")
		if err == nil {
			t.Error("GetDocument() expected error for nonexistent ID, got nil")
		}
	})
}

func TestInsertEmbedding(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	pdb, err := OpenProjectDB(dbPath)
	if err != nil {
		t.Fatalf("OpenProjectDB() error = %v", err)
	}
	defer pdb.Close()

	// Insert a document first (foreign key constraint)
	testDoc := Document{
		ID:         "embed-test-doc",
		Path:       "/test/embed.txt",
		FileType:   "txt",
		ModifiedAt: time.Now(),
		Size:       100,
		Metadata:   `{}`,
		Content:    "Test content for embedding",
	}

	if err := pdb.InsertDocument(testDoc); err != nil {
		t.Fatalf("InsertDocument() error = %v", err)
	}

	// Create test embedding (4 dimensions for simplicity)
	testEmbedding := Embedding{
		DocID:  testDoc.ID,
		Vector: []float32{0.1, 0.2, 0.3, 0.4},
	}

	t.Run("insert_embedding", func(t *testing.T) {
		err := pdb.InsertEmbedding(testEmbedding)
		if err != nil {
			t.Fatalf("InsertEmbedding() error = %v", err)
		}
	})

	t.Run("load_all_embeddings", func(t *testing.T) {
		embeddings, err := pdb.LoadAllEmbeddings()
		if err != nil {
			t.Fatalf("LoadAllEmbeddings() error = %v", err)
		}

		if len(embeddings) != 1 {
			t.Fatalf("len(embeddings) = %d, want 1", len(embeddings))
		}

		emb := embeddings[0]
		if emb.DocID != testEmbedding.DocID {
			t.Errorf("emb.DocID = %v, want %v", emb.DocID, testEmbedding.DocID)
		}

		if len(emb.Vector) != len(testEmbedding.Vector) {
			t.Fatalf("len(emb.Vector) = %d, want %d", len(emb.Vector), len(testEmbedding.Vector))
		}

		for i, v := range emb.Vector {
			if v != testEmbedding.Vector[i] {
				t.Errorf("emb.Vector[%d] = %v, want %v", i, v, testEmbedding.Vector[i])
			}
		}
	})
}

func TestInsertEmbedding_Replace(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	pdb, err := OpenProjectDB(dbPath)
	if err != nil {
		t.Fatalf("OpenProjectDB() error = %v", err)
	}
	defer pdb.Close()

	// Insert a document
	testDoc := Document{
		ID:         "replace-test-doc",
		Path:       "/test/replace.txt",
		FileType:   "txt",
		ModifiedAt: time.Now(),
		Size:       100,
		Metadata:   `{}`,
		Content:    "Test content",
	}

	if err := pdb.InsertDocument(testDoc); err != nil {
		t.Fatalf("InsertDocument() error = %v", err)
	}

	// Insert first embedding
	emb1 := Embedding{
		DocID:  testDoc.ID,
		Vector: []float32{0.1, 0.2, 0.3, 0.4},
	}
	if err := pdb.InsertEmbedding(emb1); err != nil {
		t.Fatalf("InsertEmbedding() first insert error = %v", err)
	}

	// Replace with new embedding
	emb2 := Embedding{
		DocID:  testDoc.ID,
		Vector: []float32{0.5, 0.6, 0.7, 0.8},
	}
	if err := pdb.InsertEmbedding(emb2); err != nil {
		t.Fatalf("InsertEmbedding() replace error = %v", err)
	}

	// Verify only one embedding exists and it's the updated one
	embeddings, err := pdb.LoadAllEmbeddings()
	if err != nil {
		t.Fatalf("LoadAllEmbeddings() error = %v", err)
	}

	if len(embeddings) != 1 {
		t.Fatalf("len(embeddings) = %d, want 1 (after replace)", len(embeddings))
	}

	// Check that the vector was replaced
	for i, v := range embeddings[0].Vector {
		if v != emb2.Vector[i] {
			t.Errorf("embeddings[0].Vector[%d] = %v, want %v", i, v, emb2.Vector[i])
		}
	}
}

func TestMultipleDocumentsAndEmbeddings(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	pdb, err := OpenProjectDB(dbPath)
	if err != nil {
		t.Fatalf("OpenProjectDB() error = %v", err)
	}
	defer pdb.Close()

	// Insert multiple documents
	docs := []Document{
		{ID: "doc-1", Path: "/path/1.txt", FileType: "txt", ModifiedAt: time.Now(), Size: 100, Metadata: `{}`, Content: "Content 1"},
		{ID: "doc-2", Path: "/path/2.txt", FileType: "txt", ModifiedAt: time.Now(), Size: 200, Metadata: `{}`, Content: "Content 2"},
		{ID: "doc-3", Path: "/path/3.txt", FileType: "txt", ModifiedAt: time.Now(), Size: 300, Metadata: `{}`, Content: "Content 3"},
	}

	embeddings := []Embedding{
		{DocID: "doc-1", Vector: []float32{1.0, 0.0, 0.0, 0.0}},
		{DocID: "doc-2", Vector: []float32{0.0, 1.0, 0.0, 0.0}},
		{DocID: "doc-3", Vector: []float32{0.0, 0.0, 1.0, 0.0}},
	}

	// Insert all
	for _, doc := range docs {
		if err := pdb.InsertDocument(doc); err != nil {
			t.Fatalf("InsertDocument() error = %v", err)
		}
	}

	for _, emb := range embeddings {
		if err := pdb.InsertEmbedding(emb); err != nil {
			t.Fatalf("InsertEmbedding() error = %v", err)
		}
	}

	// Verify all embeddings can be loaded
	loaded, err := pdb.LoadAllEmbeddings()
	if err != nil {
		t.Fatalf("LoadAllEmbeddings() error = %v", err)
	}

	if len(loaded) != len(embeddings) {
		t.Errorf("len(loaded) = %d, want %d", len(loaded), len(embeddings))
	}
}

func TestBuildHNSWIndex(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	pdb, err := OpenProjectDB(dbPath)
	if err != nil {
		t.Fatalf("OpenProjectDB() error = %v", err)
	}
	defer pdb.Close()

	// Insert documents with embeddings
	docs := []Document{
		{ID: "hnsw-doc-1", Path: "/path/1.txt", FileType: "txt", ModifiedAt: time.Now(), Size: 100, Metadata: `{}`, Content: "Content 1"},
		{ID: "hnsw-doc-2", Path: "/path/2.txt", FileType: "txt", ModifiedAt: time.Now(), Size: 200, Metadata: `{}`, Content: "Content 2"},
	}

	embeddings := []Embedding{
		{DocID: "hnsw-doc-1", Vector: []float32{1.0, 0.0, 0.0, 0.0}},
		{DocID: "hnsw-doc-2", Vector: []float32{0.0, 1.0, 0.0, 0.0}},
	}

	for _, doc := range docs {
		if err := pdb.InsertDocument(doc); err != nil {
			t.Fatalf("InsertDocument() error = %v", err)
		}
	}

	for _, emb := range embeddings {
		if err := pdb.InsertEmbedding(emb); err != nil {
			t.Fatalf("InsertEmbedding() error = %v", err)
		}
	}

	t.Run("build_hnsw_index", func(t *testing.T) {
		err := pdb.BuildHNSWIndex()
		if err != nil {
			t.Fatalf("BuildHNSWIndex() error = %v", err)
		}
	})

	t.Run("build_hnsw_index_idempotent", func(t *testing.T) {
		// Should not error when called again (CREATE INDEX IF NOT EXISTS)
		err := pdb.BuildHNSWIndex()
		if err != nil {
			t.Fatalf("BuildHNSWIndex() second call error = %v", err)
		}
	})
}

func TestSearchANN(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	pdb, err := OpenProjectDB(dbPath)
	if err != nil {
		t.Fatalf("OpenProjectDB() error = %v", err)
	}
	defer pdb.Close()

	// Insert documents with distinct embeddings for easy verification
	// Using orthogonal vectors for predictable search results
	docs := []Document{
		{ID: "search-doc-1", Path: "/path/1.txt", FileType: "txt", ModifiedAt: time.Now(), Size: 100, Metadata: `{}`, Content: "Document about cats"},
		{ID: "search-doc-2", Path: "/path/2.txt", FileType: "txt", ModifiedAt: time.Now(), Size: 200, Metadata: `{}`, Content: "Document about dogs"},
		{ID: "search-doc-3", Path: "/path/3.txt", FileType: "txt", ModifiedAt: time.Now(), Size: 300, Metadata: `{}`, Content: "Document about birds"},
	}

	embeddings := []Embedding{
		{DocID: "search-doc-1", Vector: []float32{1.0, 0.0, 0.0, 0.0}},  // Points along x-axis
		{DocID: "search-doc-2", Vector: []float32{0.0, 1.0, 0.0, 0.0}},  // Points along y-axis
		{DocID: "search-doc-3", Vector: []float32{0.0, 0.0, 1.0, 0.0}},  // Points along z-axis
	}

	for _, doc := range docs {
		if err := pdb.InsertDocument(doc); err != nil {
			t.Fatalf("InsertDocument() error = %v", err)
		}
	}

	for _, emb := range embeddings {
		if err := pdb.InsertEmbedding(emb); err != nil {
			t.Fatalf("InsertEmbedding() error = %v", err)
		}
	}

	// Build index before searching
	if err := pdb.BuildHNSWIndex(); err != nil {
		t.Fatalf("BuildHNSWIndex() error = %v", err)
	}

	t.Run("search_finds_closest_vector", func(t *testing.T) {
		// Search with a vector close to doc-1
		queryVector := []float32{0.9, 0.1, 0.0, 0.0}

		results, err := pdb.SearchANN(queryVector, 3)
		if err != nil {
			t.Fatalf("SearchANN() error = %v", err)
		}

		if len(results) != 3 {
			t.Fatalf("len(results) = %d, want 3", len(results))
		}

		// First result should be search-doc-1 (closest to query)
		if results[0].DocID != "search-doc-1" {
			t.Errorf("results[0].DocID = %v, want search-doc-1", results[0].DocID)
		}
	})

	t.Run("search_limit_k", func(t *testing.T) {
		queryVector := []float32{0.5, 0.5, 0.5, 0.0}

		results, err := pdb.SearchANN(queryVector, 1)
		if err != nil {
			t.Fatalf("SearchANN() error = %v", err)
		}

		if len(results) != 1 {
			t.Fatalf("len(results) = %d, want 1", len(results))
		}
	})

	t.Run("search_returns_distances", func(t *testing.T) {
		// Query exactly matching doc-2
		queryVector := []float32{0.0, 1.0, 0.0, 0.0}

		results, err := pdb.SearchANN(queryVector, 3)
		if err != nil {
			t.Fatalf("SearchANN() error = %v", err)
		}

		// First result should be doc-2 with distance 0 (or very close)
		if results[0].DocID != "search-doc-2" {
			t.Errorf("results[0].DocID = %v, want search-doc-2", results[0].DocID)
		}

		// Distance should be 0 for exact match (cosine distance)
		if results[0].Distance > 0.001 {
			t.Errorf("results[0].Distance = %v, expected close to 0 for exact match", results[0].Distance)
		}
	})

	t.Run("search_empty_result", func(t *testing.T) {
		// Request 0 results
		queryVector := []float32{1.0, 0.0, 0.0, 0.0}

		results, err := pdb.SearchANN(queryVector, 0)
		if err != nil {
			t.Fatalf("SearchANN() error = %v", err)
		}

		if len(results) != 0 {
			t.Errorf("len(results) = %d, want 0", len(results))
		}
	})
}

func TestSearchANN_EmptyDatabase(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	pdb, err := OpenProjectDB(dbPath)
	if err != nil {
		t.Fatalf("OpenProjectDB() error = %v", err)
	}
	defer pdb.Close()

	// Build index on empty table (should not error)
	if err := pdb.BuildHNSWIndex(); err != nil {
		t.Fatalf("BuildHNSWIndex() on empty table error = %v", err)
	}

	// Search on empty database
	queryVector := []float32{1.0, 0.0, 0.0, 0.0}
	results, err := pdb.SearchANN(queryVector, 3)
	if err != nil {
		t.Fatalf("SearchANN() on empty database error = %v", err)
	}

	if len(results) != 0 {
		t.Errorf("len(results) = %d, want 0 for empty database", len(results))
	}
}
