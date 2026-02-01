package db

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"
)

// skipIfNoPostgres skips the test if DATABASE_URL is not set
func skipIfNoPostgres(t *testing.T) {
	if os.Getenv("DATABASE_URL") == "" {
		t.Skip("Skipping PostgreSQL test: DATABASE_URL not set")
	}
}

func TestOpenProjectDB(t *testing.T) {
	skipIfNoPostgres(t)
	t.Run("successful_open", func(t *testing.T) {
		tmpDir := t.TempDir()
		dbPath := filepath.Join(tmpDir, "test.db")

		pdb, err := OpenProjectDB(dbPath, 4)
		if err != nil {
			t.Fatalf("OpenProjectDB() error = %v", err)
		}
		defer pdb.Close()

		// DB struct no longer has Path field (using connection string now)
		if pdb == nil {
			t.Error("expected valid DB instance")
		}
	})

	t.Run("empty_path_error", func(t *testing.T) {
		_, err := OpenProjectDB("", 4)
		if err == nil {
			t.Fatal("expected error for empty path")
		}
	})
}

func TestInsertAndGetFile(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	pdb, err := OpenProjectDB(dbPath, 4)
	if err != nil {
		t.Fatalf("OpenProjectDB() error = %v", err)
	}
	defer pdb.Close()

	testFile := File{
		FileID:     "test-file-001",
		Path:       "/test/path/file.txt",
		FileType:   "txt",
		FileHash:   "abc123",
		ModifiedAt: time.Now(),
		Size:       1234,
		IndexedAt:  time.Now(),
	}

	if err := pdb.InsertFile(context.Background(), testFile); err != nil {
		t.Fatalf("InsertFile() error = %v", err)
	}

	retrieved, err := pdb.GetFile(context.Background(), testFile.FileID)
	if err != nil {
		t.Fatalf("GetFile() error = %v", err)
	}

	if retrieved.FileID != testFile.FileID {
		t.Errorf("retrieved.FileID = %v, want %v", retrieved.FileID, testFile.FileID)
	}
	if retrieved.Path != testFile.Path {
		t.Errorf("retrieved.Path = %v, want %v", retrieved.Path, testFile.Path)
	}
}

func TestInsertFileChunkAndEmbedding(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	pdb, err := OpenProjectDB(dbPath, 4)
	if err != nil {
		t.Fatalf("OpenProjectDB() error = %v", err)
	}
	defer pdb.Close()

	// Insert file
	testFile := File{
		FileID:     "embed-test-file",
		Path:       "/test/embed.txt",
		FileType:   "txt",
		FileHash:   "abc123",
		ModifiedAt: time.Now(),
		Size:       100,
		IndexedAt:  time.Now(),
	}

	if err := pdb.InsertFile(context.Background(), testFile); err != nil {
		t.Fatalf("InsertFile() error = %v", err)
	}

	// Insert chunk
	testChunk := Chunk{
		ChunkID:     "chunk-001",
		FileID:      testFile.FileID,
		FilePath:    testFile.Path,
		FileType:    testFile.FileType,
		ChunkIndex:  0,
		TotalChunks: 1,
		Content:     "Test content",
		PageNumbers: []int{},
	}

	chunkID, err := pdb.InsertChunk(context.Background(), testChunk)
	if err != nil {
		t.Fatalf("InsertChunk() error = %v", err)
	}

	// Insert embedding
	if err := pdb.InsertEmbedding(context.Background(), chunkID, []float32{0.1, 0.2, 0.3, 0.4}); err != nil {
		t.Fatalf("InsertEmbedding() error = %v", err)
	}

	// Verify chunk retrieval
	retrieved, err := pdb.GetChunk(context.Background(), testChunk.ChunkID)
	if err != nil {
		t.Fatalf("GetChunk() error = %v", err)
	}

	if retrieved.ChunkID != testChunk.ChunkID {
		t.Errorf("ChunkID = %v, want %v", retrieved.ChunkID, testChunk.ChunkID)
	}
	if retrieved.Content != testChunk.Content {
		t.Errorf("Content = %v, want %v", retrieved.Content, testChunk.Content)
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

	// Create files
	files := []File{
		{FileID: "file-1", Path: "/1.txt", FileType: "txt", FileHash: "hash1", ModifiedAt: time.Now(), Size: 100, IndexedAt: time.Now()},
		{FileID: "file-2", Path: "/2.txt", FileType: "txt", FileHash: "hash2", ModifiedAt: time.Now(), Size: 200, IndexedAt: time.Now()},
	}

	// Create chunks
	chunks := []Chunk{
		{ChunkID: "chunk-1", FileID: "file-1", FilePath: "/1.txt", FileType: "txt", ChunkIndex: 0, TotalChunks: 1, Content: "Content 1", PageNumbers: []int{}},
		{ChunkID: "chunk-2", FileID: "file-2", FilePath: "/2.txt", FileType: "txt", ChunkIndex: 0, TotalChunks: 1, Content: "Content 2", PageNumbers: []int{}},
	}

	embeddings := []Embedding{
		{ChunkID: "chunk-1", Vector: []float32{1.0, 0.0, 0.0, 0.0}},
		{ChunkID: "chunk-2", Vector: []float32{0.0, 1.0, 0.0, 0.0}},
	}

	for _, file := range files {
		pdb.InsertFile(context.Background(), file)
	}
	for _, chunk := range chunks {
		pdb.InsertChunk(context.Background(), chunk)
	}
	for _, emb := range embeddings {
		pdb.InsertEmbedding(context.Background(), emb.ChunkID, emb.Vector)
	}

	if err := pdb.BuildHNSWIndex(context.Background()); err != nil {
		t.Fatalf("BuildHNSWIndex() error = %v", err)
	}

	// Should be idempotent
	if err := pdb.BuildHNSWIndex(context.Background()); err != nil {
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

	// Create files
	files := []File{
		{FileID: "file-1", Path: "/1.txt", FileType: "txt", FileHash: "hash1", ModifiedAt: time.Now(), Size: 100, IndexedAt: time.Now()},
		{FileID: "file-2", Path: "/2.txt", FileType: "txt", FileHash: "hash2", ModifiedAt: time.Now(), Size: 200, IndexedAt: time.Now()},
		{FileID: "file-3", Path: "/3.txt", FileType: "txt", FileHash: "hash3", ModifiedAt: time.Now(), Size: 300, IndexedAt: time.Now()},
	}

	// Create chunks
	chunks := []Chunk{
		{ChunkID: "chunk-1", FileID: "file-1", FilePath: "/1.txt", FileType: "txt", ChunkIndex: 0, TotalChunks: 1, Content: "Cats", PageNumbers: []int{}},
		{ChunkID: "chunk-2", FileID: "file-2", FilePath: "/2.txt", FileType: "txt", ChunkIndex: 0, TotalChunks: 1, Content: "Dogs", PageNumbers: []int{}},
		{ChunkID: "chunk-3", FileID: "file-3", FilePath: "/3.txt", FileType: "txt", ChunkIndex: 0, TotalChunks: 1, Content: "Birds", PageNumbers: []int{}},
	}

	embeddings := []Embedding{
		{ChunkID: "chunk-1", Vector: []float32{1.0, 0.0, 0.0, 0.0}},
		{ChunkID: "chunk-2", Vector: []float32{0.0, 1.0, 0.0, 0.0}},
		{ChunkID: "chunk-3", Vector: []float32{0.0, 0.0, 1.0, 0.0}},
	}

	for _, file := range files {
		pdb.InsertFile(context.Background(), file)
	}
	for _, chunk := range chunks {
		pdb.InsertChunk(context.Background(), chunk)
	}
	for _, emb := range embeddings {
		pdb.InsertEmbedding(context.Background(), emb.ChunkID, emb.Vector)
	}
	pdb.BuildHNSWIndex(context.Background())

	t.Run("finds_closest", func(t *testing.T) {
		results, err := pdb.SearchANN(context.Background(), []float32{0.9, 0.1, 0.0, 0.0}, 3)
		if err != nil {
			t.Fatalf("SearchANN() error = %v", err)
		}

		if len(results) != 3 {
			t.Fatalf("len(results) = %d, want 3", len(results))
		}

		if results[0].ChunkID != "chunk-1" {
			t.Errorf("results[0].ChunkID = %v, want chunk-1", results[0].ChunkID)
		}
	})

	t.Run("limit_k", func(t *testing.T) {
		results, err := pdb.SearchANN(context.Background(), []float32{0.5, 0.5, 0.5, 0.0}, 1)
		if err != nil {
			t.Fatalf("SearchANN() error = %v", err)
		}

		if len(results) != 1 {
			t.Fatalf("len(results) = %d, want 1", len(results))
		}
	})

	t.Run("exact_match_distance_zero", func(t *testing.T) {
		results, err := pdb.SearchANN(context.Background(), []float32{0.0, 1.0, 0.0, 0.0}, 1)
		if err != nil {
			t.Fatalf("SearchANN() error = %v", err)
		}

		if results[0].ChunkID != "chunk-2" {
			t.Errorf("results[0].ChunkID = %v, want chunk-2", results[0].ChunkID)
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

	pdb.BuildHNSWIndex(context.Background())

	results, err := pdb.SearchANN(context.Background(), []float32{1.0, 0.0, 0.0, 0.0}, 3)
	if err != nil {
		t.Fatalf("SearchANN() error = %v", err)
	}

	if len(results) != 0 {
		t.Errorf("len(results) = %d, want 0", len(results))
	}
}

func TestDeleteFile_Cascade(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	pdb, err := OpenProjectDB(dbPath, 4)
	if err != nil {
		t.Fatalf("OpenProjectDB() error = %v", err)
	}
	defer pdb.Close()

	// Insert file with chunks and embeddings
	file := File{
		FileID:     "file-001",
		Path:       "/test.txt",
		FileType:   "txt",
		FileHash:   "hash123",
		ModifiedAt: time.Now(),
		Size:       100,
		IndexedAt:  time.Now(),
	}
	if err := pdb.InsertFile(context.Background(), file); err != nil {
		t.Fatalf("InsertFile() error = %v", err)
	}

	chunk := Chunk{
		ChunkID:     "chunk-001",
		FileID:      "file-001",
		FilePath:    "/test.txt",
		FileType:    "txt",
		ChunkIndex:  0,
		TotalChunks: 1,
		Content:     "Test content",
		PageNumbers: []int{},
	}
	chunkID, err := pdb.InsertChunk(context.Background(), chunk)
	if err != nil {
		t.Fatalf("InsertChunk() error = %v", err)
	}

	// Insert embedding
	if err := pdb.InsertEmbedding(context.Background(), chunkID, []float32{0.1, 0.2, 0.3, 0.4}); err != nil {
		t.Fatalf("InsertEmbedding() error = %v", err)
	}

	// Delete file (should cascade)
	err = pdb.DeleteFile(context.Background(), "file-001")
	if err != nil {
		t.Fatalf("DeleteFile() error = %v", err)
	}

	// Verify cascade deletion
	_, err = pdb.GetFile(context.Background(), "file-001")
	if err == nil {
		t.Error("File should be deleted")
	}

	_, err = pdb.GetChunk(context.Background(), "chunk-001")
	if err == nil {
		t.Error("Chunk should be deleted")
	}

	// Verify embedding deleted by trying to search
	results, _ := pdb.SearchANN(context.Background(), []float32{0.1, 0.2, 0.3, 0.4}, 1)
	if len(results) > 0 {
		t.Error("Embedding should be deleted")
	}
}

func TestGetAllFiles(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	pdb, err := OpenProjectDB(dbPath, 4)
	if err != nil {
		t.Fatalf("OpenProjectDB() error = %v", err)
	}
	defer pdb.Close()

	files := []File{
		{FileID: "f1", Path: "/1.txt", FileType: "txt", FileHash: "h1", ModifiedAt: time.Now(), Size: 100, IndexedAt: time.Now()},
		{FileID: "f2", Path: "/2.txt", FileType: "txt", FileHash: "h2", ModifiedAt: time.Now(), Size: 200, IndexedAt: time.Now()},
	}

	for _, f := range files {
		if err := pdb.InsertFile(context.Background(), f); err != nil {
			t.Fatalf("InsertFile() error = %v", err)
		}
	}

	allFiles, err := pdb.GetAllFiles(context.Background())
	if err != nil {
		t.Fatalf("GetAllFiles() error = %v", err)
	}

	if len(allFiles) != 2 {
		t.Errorf("len(allFiles) = %d, want 2", len(allFiles))
	}
}

func TestGetFileByPath(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	pdb, err := OpenProjectDB(dbPath, 4)
	if err != nil {
		t.Fatalf("OpenProjectDB() error = %v", err)
	}
	defer pdb.Close()

	testFile := File{
		FileID:     "test-file",
		Path:       "/test/path/file.txt",
		FileType:   "txt",
		FileHash:   "hash123",
		ModifiedAt: time.Now(),
		Size:       500,
		IndexedAt:  time.Now(),
	}

	if err := pdb.InsertFile(context.Background(), testFile); err != nil {
		t.Fatalf("InsertFile() error = %v", err)
	}

	retrieved, err := pdb.GetFileByPath(context.Background(), "/test/path/file.txt")
	if err != nil {
		t.Fatalf("GetFileByPath() error = %v", err)
	}

	if retrieved.FileID != testFile.FileID {
		t.Errorf("FileID = %v, want %v", retrieved.FileID, testFile.FileID)
	}
	if retrieved.Path != testFile.Path {
		t.Errorf("Path = %v, want %v", retrieved.Path, testFile.Path)
	}
}

func TestGetChunksForFile(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	pdb, err := OpenProjectDB(dbPath, 4)
	if err != nil {
		t.Fatalf("OpenProjectDB() error = %v", err)
	}
	defer pdb.Close()

	file := File{
		FileID:     "file-001",
		Path:       "/test.txt",
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
		{ChunkID: "c1", FileID: "file-001", FilePath: "/test.txt", FileType: "txt", ChunkIndex: 0, TotalChunks: 2, Content: "Chunk 1", PageNumbers: []int{}},
		{ChunkID: "c2", FileID: "file-001", FilePath: "/test.txt", FileType: "txt", ChunkIndex: 1, TotalChunks: 2, Content: "Chunk 2", PageNumbers: []int{}},
	}

	for _, c := range chunks {
		_, err = pdb.InsertChunk(context.Background(), c)
		if err != nil {
			t.Fatalf("InsertChunk() error = %v", err)
		}
	}

	fileChunks, err := pdb.GetChunksForFile(context.Background(), "file-001")
	if err != nil {
		t.Fatalf("GetChunksForFile() error = %v", err)
	}

	if len(fileChunks) != 2 {
		t.Errorf("len(fileChunks) = %d, want 2", len(fileChunks))
	}

	// Verify chunks are ordered by chunk_index
	if fileChunks[0].ChunkIndex != 0 {
		t.Errorf("First chunk ChunkIndex = %d, want 0", fileChunks[0].ChunkIndex)
	}
	if fileChunks[1].ChunkIndex != 1 {
		t.Errorf("Second chunk ChunkIndex = %d, want 1", fileChunks[1].ChunkIndex)
	}
}

func TestRebuildHNSWIndex(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	pdb, err := OpenProjectDB(dbPath, 4)
	if err != nil {
		t.Fatalf("OpenProjectDB() error = %v", err)
	}
	defer pdb.Close()

	// Insert embeddings
	file := File{
		FileID:     "file-1",
		Path:       "/1.txt",
		FileType:   "txt",
		FileHash:   "h1",
		ModifiedAt: time.Now(),
		Size:       100,
		IndexedAt:  time.Now(),
	}
	pdb.InsertFile(context.Background(), file)

	chunks := []Chunk{
		{ChunkID: "c1", FileID: "file-1", FilePath: "/1.txt", FileType: "txt", ChunkIndex: 0, TotalChunks: 2, Content: "Content 1", PageNumbers: []int{}},
		{ChunkID: "c2", FileID: "file-1", FilePath: "/1.txt", FileType: "txt", ChunkIndex: 1, TotalChunks: 2, Content: "Content 2", PageNumbers: []int{}},
	}

	embeddings := []Embedding{
		{ChunkID: "c1", Vector: []float32{1.0, 0.0, 0.0, 0.0}},
		{ChunkID: "c2", Vector: []float32{0.0, 1.0, 0.0, 0.0}},
	}

	for _, c := range chunks {
		pdb.InsertChunk(context.Background(), c)
	}
	for _, e := range embeddings {
		pdb.InsertEmbedding(context.Background(), e.ChunkID, e.Vector)
	}

	// Build initial index
	if err := pdb.BuildHNSWIndex(context.Background()); err != nil {
		t.Fatalf("BuildHNSWIndex() error = %v", err)
	}

	// Verify search works
	results, err := pdb.SearchANN(context.Background(), []float32{0.9, 0.1, 0.0, 0.0}, 2)
	if err != nil {
		t.Fatalf("SearchANN() error = %v", err)
	}
	if len(results) != 2 {
		t.Errorf("len(results) = %d, want 2", len(results))
	}

	// Rebuild index
	if err := pdb.RebuildHNSWIndex(context.Background()); err != nil {
		t.Fatalf("RebuildHNSWIndex() error = %v", err)
	}

	// Verify search still works after rebuild
	results, err = pdb.SearchANN(context.Background(), []float32{0.9, 0.1, 0.0, 0.0}, 2)
	if err != nil {
		t.Fatalf("SearchANN() after rebuild error = %v", err)
	}
	if len(results) != 2 {
		t.Errorf("len(results) after rebuild = %d, want 2", len(results))
	}
}

func TestRebuildHNSWIndex_EmptyTable(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	pdb, err := OpenProjectDB(dbPath, 4)
	if err != nil {
		t.Fatalf("OpenProjectDB() error = %v", err)
	}
	defer pdb.Close()

	// Should not error on empty table
	if err := pdb.RebuildHNSWIndex(context.Background()); err != nil {
		t.Fatalf("RebuildHNSWIndex() on empty table error = %v", err)
	}
}
