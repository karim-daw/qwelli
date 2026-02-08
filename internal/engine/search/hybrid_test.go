package search

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/karim-daw/qwelli/internal/db"
)

// TestHybridSearch_CombinesResults tests that hybrid search combines semantic and keyword results
func TestHybridSearch_CombinesResults(t *testing.T) {
	mockClient := newMockClient(1024)
	semantic := NewSemanticSearchStrategy(mockClient)
	keyword := NewKeywordSearchStrategy()
	hybrid := NewHybridSearchStrategy(semantic, keyword)

	testDB := createTestHybridIndex(t)
	defer testDB.Close()

	// Search with query that should match both semantically and by keyword
	results, err := hybrid.Search("cats pets", testDB, 5, "")
	if err != nil {
		t.Fatalf("Hybrid search failed: %v", err)
	}

	if len(results) == 0 {
		t.Error("Expected hybrid search results")
	}

	// Verify results have required fields
	for _, result := range results {
		if result.FilePath == "" {
			t.Error("Expected FilePath to be set")
		}
		if result.Content == "" {
			t.Error("Expected Content to be set")
		}
	}
}

// TestHybridSearch_Deduplication tests that hybrid search removes duplicates
func TestHybridSearch_Deduplication(t *testing.T) {
	mockClient := newMockClient(1024)
	semantic := NewSemanticSearchStrategy(mockClient)
	keyword := NewKeywordSearchStrategy()
	hybrid := NewHybridSearchStrategy(semantic, keyword)

	testDB := createTestHybridIndex(t)
	defer testDB.Close()

	// Search with query that will likely return same chunks from both strategies
	results, err := hybrid.Search("cats", testDB, 10, "")
	if err != nil {
		t.Fatalf("Hybrid search failed: %v", err)
	}

	// Verify no duplicate chunk IDs
	seen := make(map[string]bool)
	for _, result := range results {
		if seen[result.ChunkID] {
			t.Errorf("Duplicate chunk ID found: %s", result.ChunkID)
		}
		seen[result.ChunkID] = true
	}
}

// TestHybridSearch_TopKLimit tests that results respect the topK limit
func TestHybridSearch_TopKLimit(t *testing.T) {
	mockClient := newMockClient(1024)
	semantic := NewSemanticSearchStrategy(mockClient)
	keyword := NewKeywordSearchStrategy()
	hybrid := NewHybridSearchStrategy(semantic, keyword)

	testDB := createTestHybridIndex(t)
	defer testDB.Close()

	// Search with topK = 2
	results, err := hybrid.Search("test query", testDB, 2, "")
	if err != nil {
		t.Fatalf("Hybrid search failed: %v", err)
	}

	if len(results) > 2 {
		t.Errorf("Expected at most 2 results, got %d", len(results))
	}
}

// TestHybridSearch_EmptySemanticResults tests when semantic search returns nothing
func TestHybridSearch_EmptySemanticResults(t *testing.T) {
	mockClient := newMockClient(1024)
	semantic := NewSemanticSearchStrategy(mockClient)
	keyword := NewKeywordSearchStrategy()
	hybrid := NewHybridSearchStrategy(semantic, keyword)

	testDB := createTestHybridIndex(t)
	defer testDB.Close()

	// This should still work even if one strategy has no results
	results, err := hybrid.Search("test", testDB, 5, "")
	if err != nil {
		t.Fatalf("Hybrid search failed: %v", err)
	}

	// May or may not have results, but should not error
	t.Logf("Got %d results", len(results))
}

// TestHybridSearch_ContentTypeFilter tests content type filtering
func TestHybridSearch_ContentTypeFilter(t *testing.T) {
	mockClient := newMockClient(1024)
	semantic := NewSemanticSearchStrategy(mockClient)
	keyword := NewKeywordSearchStrategy()
	hybrid := NewHybridSearchStrategy(semantic, keyword)

	testDB := createTestHybridIndex(t)
	defer testDB.Close()

	// Search with content type filter
	results, err := hybrid.Search("test", testDB, 10, "text")
	if err != nil {
		t.Fatalf("Hybrid search failed: %v", err)
	}

	// Verify all results are text content
	for _, result := range results {
		if result.ContentType != "text" {
			t.Errorf("Expected text content type, got %s", result.ContentType)
		}
	}
}

// TestHybridSearch_CustomWeights tests hybrid search with custom weights
func TestHybridSearch_CustomWeights(t *testing.T) {
	mockClient := newMockClient(1024)
	semantic := NewSemanticSearchStrategy(mockClient)
	keyword := NewKeywordSearchStrategy()

	// Create hybrid with custom weights
	config := HybridSearchConfig{
		SemanticWeight: 0.9,
		KeywordWeight:  0.1,
	}
	hybrid := NewHybridSearchStrategyWithConfig(semantic, keyword, config)

	testDB := createTestHybridIndex(t)
	defer testDB.Close()

	results, err := hybrid.Search("cats", testDB, 5, "")
	if err != nil {
		t.Fatalf("Hybrid search failed: %v", err)
	}

	if len(results) == 0 {
		t.Error("Expected hybrid search results with custom weights")
	}
}

// TestHybridSearch_DefaultWeights tests that default weights are used correctly
func TestHybridSearch_DefaultWeights(t *testing.T) {
	mockClient := newMockClient(1024)
	semantic := NewSemanticSearchStrategy(mockClient)
	keyword := NewKeywordSearchStrategy()
	hybrid := NewHybridSearchStrategy(semantic, keyword)

	// Verify default weights
	if hybrid.semanticWeight != 0.7 {
		t.Errorf("Expected default semantic weight 0.7, got %f", hybrid.semanticWeight)
	}
	if hybrid.keywordWeight != 0.3 {
		t.Errorf("Expected default keyword weight 0.3, got %f", hybrid.keywordWeight)
	}
}

// TestHybridSearch_StrategyName tests that the strategy returns correct name
func TestHybridSearch_StrategyName(t *testing.T) {
	mockClient := newMockClient(1024)
	semantic := NewSemanticSearchStrategy(mockClient)
	keyword := NewKeywordSearchStrategy()
	hybrid := NewHybridSearchStrategy(semantic, keyword)

	if hybrid.Name() != "hybrid" {
		t.Errorf("Expected strategy name 'hybrid', got '%s'", hybrid.Name())
	}
}

// TestHybridSearch_EmptyIndex tests search on empty index
func TestHybridSearch_EmptyIndex(t *testing.T) {
	mockClient := newMockClient(1024)
	semantic := NewSemanticSearchStrategy(mockClient)
	keyword := NewKeywordSearchStrategy()
	hybrid := NewHybridSearchStrategy(semantic, keyword)

	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "test.db")
	testDB, err := db.OpenProjectDB(dbPath, 1024)
	if err != nil {
		t.Fatalf("Failed to create database: %v", err)
	}
	defer testDB.Close()

	results, err := hybrid.Search("test", testDB, 10, "")
	if err != nil {
		t.Fatalf("Hybrid search failed: %v", err)
	}

	if len(results) != 0 {
		t.Errorf("Expected no results from empty index, got %d", len(results))
	}
}

// TestHybridSearch_SortedByScore tests that results are sorted by combined score
func TestHybridSearch_SortedByScore(t *testing.T) {
	mockClient := newMockClient(1024)
	semantic := NewSemanticSearchStrategy(mockClient)
	keyword := NewKeywordSearchStrategy()
	hybrid := NewHybridSearchStrategy(semantic, keyword)

	testDB := createTestHybridIndex(t)
	defer testDB.Close()

	results, err := hybrid.Search("cats pets animals", testDB, 5, "")
	if err != nil {
		t.Fatalf("Hybrid search failed: %v", err)
	}

	// Verify results are sorted by distance (lower is better)
	for i := 1; i < len(results); i++ {
		if results[i].Distance < results[i-1].Distance {
			t.Errorf("Results not sorted by distance: %f < %f at index %d",
				results[i].Distance, results[i-1].Distance, i)
		}
	}
}

// createTestHybridIndex creates a test database with sample indexed content
func createTestHybridIndex(t *testing.T) *db.ProjectDB {
	t.Helper()

	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "test.db")

	projectDB, err := db.OpenProjectDB(dbPath, 1024)
	if err != nil {
		t.Fatalf("Failed to create database: %v", err)
	}

	// Create mock client for embeddings
	mockClient := newMockClient(1024)

	// Create sample chunks with varied content for testing both strategies
	chunks := []struct {
		content     string
		filePath    string
		contentType string
	}{
		{"Cats are wonderful pets. They are independent and loving animals that enjoy human company.", "/test/cats.txt", "text"},
		{"Dogs are loyal companions. They love to play fetch and go for walks with their owners.", "/test/dogs.txt", "text"},
		{"Birds can fly and sing beautiful songs. Many people keep them as pets in cages.", "/test/birds.txt", "text"},
		{"Fish live in water and come in many beautiful colors and patterns. They are quiet pets.", "/test/fish.txt", "text"},
		{"Hamsters are small pets that love to run on wheels. They are active at night.", "/test/hamsters.txt", "text"},
	}

	for _, chunk := range chunks {
		// Generate embedding
		embedding, err := mockClient.Embed(chunk.content)
		if err != nil {
			t.Fatalf("Failed to generate embedding: %v", err)
		}

		// Create chunk ID
		chunkID := filepath.Base(chunk.filePath) + "_0"

		// Insert chunk
		err = projectDB.InsertChunk(db.Chunk{
			ChunkID:     chunkID,
			FileID:      filepath.Base(chunk.filePath),
			FilePath:    chunk.filePath,
			ChunkIndex:  0,
			TotalChunks: 1,
			Content:     chunk.content,
			ContentType: chunk.contentType,
		})
		if err != nil {
			t.Fatalf("Failed to insert chunk: %v", err)
		}

		// Insert embedding
		err = projectDB.InsertEmbedding(db.Embedding{
			ChunkID: chunkID,
			Vector:  embedding,
		})
		if err != nil {
			t.Fatalf("Failed to insert embedding: %v", err)
		}
	}

	// Rebuild HNSW index for semantic search
	if err := projectDB.RebuildHNSWIndex(); err != nil {
		t.Fatalf("Failed to rebuild HNSW index: %v", err)
	}

	// Mark files as indexed
	for _, chunk := range chunks {
		stat, _ := os.Stat(tempDir)
		projectDB.InsertFile(db.File{
			FileID:     filepath.Base(chunk.filePath),
			Path:       chunk.filePath,
			FileHash:   "test-hash",
			Size:       100,
			ModifiedAt: stat.ModTime(),
		})
	}

	return projectDB
}
