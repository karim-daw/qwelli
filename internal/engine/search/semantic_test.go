package search

import (
	"context"
	"crypto/sha256"
	"math"
	"os"
	"path/filepath"
	"testing"

	"github.com/karim-daw/qwelli/internal/db"
	"github.com/karim-daw/qwelli/internal/voyage"
)

// mockVoyageClient is a simple mock for testing search strategies
type mockVoyageClient struct {
	dimension int
}

func newMockClient(dimension int) voyage.ClientInterface {
	return &mockVoyageClient{dimension: dimension}
}

func (m *mockVoyageClient) Embed(text string) ([]float32, error) {
	return m.generateEmbedding(text), nil
}

func (m *mockVoyageClient) EmbedBatch(ctx context.Context, texts []string, progressCallback func(current, total int)) ([][]float32, error) {
	embeddings := make([][]float32, len(texts))
	for i, text := range texts {
		embeddings[i] = m.generateEmbedding(text)
		if progressCallback != nil {
			progressCallback(i+1, len(texts))
		}
	}
	return embeddings, nil
}

func (m *mockVoyageClient) EmbedMultimodal(ctx context.Context, inputs []voyage.MultimodalInput, progressCallback func(current, total int)) ([][]float32, error) {
	embeddings := make([][]float32, len(inputs))
	for i, input := range inputs {
		text := input.Text
		embeddings[i] = m.generateEmbedding(text)
		if progressCallback != nil {
			progressCallback(i+1, len(inputs))
		}
	}
	return embeddings, nil
}

func (m *mockVoyageClient) Rerank(ctx context.Context, query string, documents []string) ([]voyage.RerankResult, error) {
	return nil, nil
}

func (m *mockVoyageClient) RerankWithMetadata(ctx context.Context, query string, inputs []voyage.RerankInput) ([]voyage.RerankResult, error) {
	return nil, nil
}

func (m *mockVoyageClient) EmbeddingModel() string {
	return "mock-model"
}

func (m *mockVoyageClient) RerankModel() string {
	return "mock-rerank"
}

func (m *mockVoyageClient) generateEmbedding(text string) []float32 {
	hash := sha256.Sum256([]byte(text))
	embedding := make([]float32, m.dimension)
	for i := 0; i < m.dimension; i++ {
		idx := i % len(hash)
		val := float64(hash[idx]) / 255.0
		embedding[i] = float32(val)
	}
	// Normalize
	var sum float64
	for _, v := range embedding {
		sum += float64(v) * float64(v)
	}
	norm := math.Sqrt(sum)
	if norm > 0 {
		for i := range embedding {
			embedding[i] /= float32(norm)
		}
	}
	return embedding
}

// TestSemanticSearch_BasicQuery tests basic semantic search functionality
func TestSemanticSearch_BasicQuery(t *testing.T) {
	mockClient := newMockClient(1024)
	strategy := NewSemanticSearchStrategy(mockClient)

	// Create test database and index
	testDB := createTestSemanticIndex(t, 1024)
	defer testDB.Close()

	// Search for "cats"
	results, err := strategy.Search("cats are wonderful pets", testDB, 5, "")
	if err != nil {
		t.Fatalf("Search failed: %v", err)
	}

	if len(results) == 0 {
		t.Error("Expected search results")
	}

	// Verify results are sorted by distance (lower is better)
	for i := 1; i < len(results); i++ {
		if results[i].Distance < results[i-1].Distance {
			t.Errorf("Results not sorted by distance: %f < %f at index %d", results[i].Distance, results[i-1].Distance, i)
		}
	}
}

// TestSemanticSearch_NoResults tests search with no matching results
func TestSemanticSearch_NoResults(t *testing.T) {
	mockClient := newMockClient(1024)
	strategy := NewSemanticSearchStrategy(mockClient)

	// Create empty database
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "test.db")
	testDB, err := db.OpenProjectDB(dbPath, 1024)
	if err != nil {
		t.Fatalf("Failed to create database: %v", err)
	}
	defer testDB.Close()

	// Search empty index
	results, err := strategy.Search("test query", testDB, 5, "")
	if err != nil {
		t.Fatalf("Search failed: %v", err)
	}

	if len(results) != 0 {
		t.Errorf("Expected no results, got %d", len(results))
	}
}

// TestSemanticSearch_TopKLimit tests that results respect the topK limit
func TestSemanticSearch_TopKLimit(t *testing.T) {
	mockClient := newMockClient(1024)
	strategy := NewSemanticSearchStrategy(mockClient)

	testDB := createTestSemanticIndex(t, 1024)
	defer testDB.Close()

	// Search with topK = 2
	results, err := strategy.Search("test query", testDB, 2, "")
	if err != nil {
		t.Fatalf("Search failed: %v", err)
	}

	if len(results) > 2 {
		t.Errorf("Expected at most 2 results, got %d", len(results))
	}
}

// TestSemanticSearch_ContentTypeFilter tests filtering by content type
func TestSemanticSearch_ContentTypeFilter(t *testing.T) {
	mockClient := newMockClient(1024)
	strategy := NewSemanticSearchStrategy(mockClient)

	testDB := createTestSemanticIndex(t, 1024)
	defer testDB.Close()

	// Search for text content only
	results, err := strategy.Search("test query", testDB, 10, "text")
	if err != nil {
		t.Fatalf("Search failed: %v", err)
	}

	// Verify all results are text content
	for _, result := range results {
		if result.ContentType != "text" {
			t.Errorf("Expected text content type, got %s", result.ContentType)
		}
	}
}

// TestSemanticSearch_DifferentDimensions tests search with different embedding dimensions
func TestSemanticSearch_DifferentDimensions(t *testing.T) {
	tests := []struct {
		name      string
		dimension int
	}{
		{"dimension 512", 512},
		{"dimension 1024", 1024},
		{"dimension 1536", 1536},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockClient := newMockClient(tt.dimension)
			strategy := NewSemanticSearchStrategy(mockClient)

			testDB := createTestSemanticIndex(t, tt.dimension)
			defer testDB.Close()

			results, err := strategy.Search("test query", testDB, 5, "")
			if err != nil {
				t.Fatalf("Search failed: %v", err)
			}

			// Should work with any dimension
			if len(results) == 0 {
				t.Error("Expected search results")
			}
		})
	}
}

// TestSemanticSearch_StrategyName tests that the strategy returns correct name
func TestSemanticSearch_StrategyName(t *testing.T) {
	mockClient := newMockClient(1024)
	strategy := NewSemanticSearchStrategy(mockClient)

	if strategy.Name() != "semantic" {
		t.Errorf("Expected strategy name 'semantic', got '%s'", strategy.Name())
	}
}

// createTestSemanticIndex creates a test database with sample indexed content
func createTestSemanticIndex(t *testing.T, dimension int) *db.ProjectDB {
	t.Helper()

	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "test.db")

	projectDB, err := db.OpenProjectDB(dbPath, dimension)
	if err != nil {
		t.Fatalf("Failed to create database: %v", err)
	}

	// Create mock client for embeddings
	mockClient := newMockClient(dimension)

	// Create sample chunks with embeddings
	chunks := []struct {
		content     string
		filePath    string
		contentType string
	}{
		{"Cats are wonderful pets. They are independent and loving.", "/test/cats.txt", "text"},
		{"Dogs are loyal companions. They love to play fetch.", "/test/dogs.txt", "text"},
		{"Birds can fly and sing beautiful songs.", "/test/birds.txt", "text"},
		{"Fish live in water and come in many colors.", "/test/fish.txt", "text"},
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

	// Rebuild HNSW index
	if err := projectDB.RebuildHNSWIndex(); err != nil {
		t.Fatalf("Failed to rebuild HNSW index: %v", err)
	}

	// Mark files as indexed
	for _, chunk := range chunks {
		stat, _ := os.Stat(tempDir)
		projectDB.InsertFile(db.File{
			FileID:   filepath.Base(chunk.filePath),
			Path:     chunk.filePath,
			FileHash: "test-hash",
			Size:     100,
			ModifiedAt: stat.ModTime(),
		})
	}

	return projectDB
}
