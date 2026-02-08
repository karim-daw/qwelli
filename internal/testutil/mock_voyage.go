package testutil

import (
	"context"
	"crypto/sha256"
	"encoding/binary"
	"math"

	"github.com/karim-daw/qwelli/internal/voyage"
)

// MockVoyageClient is a mock implementation of the Voyage client for testing
type MockVoyageClient struct {
	dimension      int
	embeddingModel string
	rerankModel    string

	// Optional error injection
	EmbedError  error
	RerankError error

	// Call tracking
	EmbedCallCount          int
	EmbedBatchCallCount     int
	EmbedMultimodalCallCount int
	RerankCallCount         int
	LastEmbedInputs         []string
	LastMultimodalInputs    []voyage.MultimodalInput
	LastRerankQuery         string
	LastRerankDocuments     []string
}

// NewMockVoyageClient creates a new mock Voyage client
func NewMockVoyageClient(dimension int) *MockVoyageClient {
	return &MockVoyageClient{
		dimension:      dimension,
		embeddingModel: "mock-voyage-model",
		rerankModel:    "mock-rerank-model",
	}
}

// NewMockVoyageClientWithModels creates a mock client with specific model names
func NewMockVoyageClientWithModels(dimension int, embeddingModel, rerankModel string) *MockVoyageClient {
	return &MockVoyageClient{
		dimension:      dimension,
		embeddingModel: embeddingModel,
		rerankModel:    rerankModel,
	}
}

// EmbeddingModel returns the configured embedding model name
func (m *MockVoyageClient) EmbeddingModel() string {
	return m.embeddingModel
}

// RerankModel returns the configured rerank model name
func (m *MockVoyageClient) RerankModel() string {
	return m.rerankModel
}

// Embed generates a deterministic embedding for a single text
func (m *MockVoyageClient) Embed(text string) ([]float32, error) {
	m.EmbedCallCount++
	m.LastEmbedInputs = []string{text}

	if m.EmbedError != nil {
		return nil, m.EmbedError
	}

	return m.generateDeterministicEmbedding(text), nil
}

// EmbedBatch generates deterministic embeddings for multiple texts
func (m *MockVoyageClient) EmbedBatch(ctx context.Context, texts []string, progressCallback func(current, total int)) ([][]float32, error) {
	m.EmbedBatchCallCount++
	m.LastEmbedInputs = texts

	if m.EmbedError != nil {
		return nil, m.EmbedError
	}

	embeddings := make([][]float32, len(texts))
	for i, text := range texts {
		// Check for cancellation
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}

		embeddings[i] = m.generateDeterministicEmbedding(text)

		if progressCallback != nil {
			progressCallback(i+1, len(texts))
		}
	}

	return embeddings, nil
}

// EmbedMultimodal generates deterministic embeddings for multimodal inputs
func (m *MockVoyageClient) EmbedMultimodal(ctx context.Context, inputs []voyage.MultimodalInput, progressCallback func(current, total int)) ([][]float32, error) {
	m.EmbedMultimodalCallCount++
	m.LastMultimodalInputs = inputs

	if m.EmbedError != nil {
		return nil, m.EmbedError
	}

	embeddings := make([][]float32, len(inputs))
	for i, input := range inputs {
		// Check for cancellation
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}

		// Generate embedding based on input type
		switch input.Type {
		case "text":
			embeddings[i] = m.generateDeterministicEmbedding(input.Text)
		case "image":
			// Generate embedding from image base64
			embeddings[i] = m.generateDeterministicEmbedding(input.ImageBase64)
		default:
			embeddings[i] = m.generateDeterministicEmbedding("unknown")
		}

		if progressCallback != nil {
			progressCallback(i+1, len(inputs))
		}
	}

	return embeddings, nil
}

// Rerank returns deterministic rerank results
func (m *MockVoyageClient) Rerank(ctx context.Context, query string, documents []string) ([]voyage.RerankResult, error) {
	m.RerankCallCount++
	m.LastRerankQuery = query
	m.LastRerankDocuments = documents

	if m.RerankError != nil {
		return nil, m.RerankError
	}

	if len(documents) == 0 {
		return []voyage.RerankResult{}, nil
	}

	// Generate deterministic relevance scores based on query and document content
	results := make([]voyage.RerankResult, len(documents))
	for i, doc := range documents {
		score := m.calculateRelevanceScore(query, doc)
		results[i] = voyage.RerankResult{
			Index:          i,
			RelevanceScore: score,
			Content:        doc,
		}
	}

	// Sort by relevance score (descending)
	// Use a simple bubble sort to keep it simple
	for i := 0; i < len(results); i++ {
		for j := i + 1; j < len(results); j++ {
			if results[j].RelevanceScore > results[i].RelevanceScore {
				results[i], results[j] = results[j], results[i]
			}
		}
	}

	return results, nil
}

// RerankWithMetadata returns deterministic rerank results with metadata
func (m *MockVoyageClient) RerankWithMetadata(ctx context.Context, query string, inputs []voyage.RerankInput) ([]voyage.RerankResult, error) {
	if len(inputs) == 0 {
		return []voyage.RerankResult{}, nil
	}

	// Extract documents
	documents := make([]string, len(inputs))
	for i, input := range inputs {
		documents[i] = input.Content
	}

	// Call Rerank
	results, err := m.Rerank(ctx, query, documents)
	if err != nil {
		return nil, err
	}

	// Attach metadata
	for i := range results {
		if results[i].Index >= 0 && results[i].Index < len(inputs) {
			results[i].Metadata = inputs[results[i].Index].Metadata
		}
	}

	return results, nil
}

// generateDeterministicEmbedding generates a deterministic embedding vector from text
// The same text will always produce the same embedding
func (m *MockVoyageClient) generateDeterministicEmbedding(text string) []float32 {
	// Use SHA256 hash to generate deterministic values
	hash := sha256.Sum256([]byte(text))

	embedding := make([]float32, m.dimension)

	// Use hash bytes to seed pseudo-random values
	for i := 0; i < m.dimension; i++ {
		// Get 4 bytes from hash, cycling through if needed
		offset := (i * 4) % len(hash)
		var bytes [4]byte
		for j := 0; j < 4; j++ {
			bytes[j] = hash[(offset+j)%len(hash)]
		}

		// Convert to uint32, then to float in range [-1, 1]
		val := binary.BigEndian.Uint32(bytes[:])
		// Normalize to [-1, 1]
		embedding[i] = (float32(val)/float32(math.MaxUint32))*2.0 - 1.0
	}

	// Normalize the vector to unit length (L2 norm = 1)
	var sumSquares float32
	for _, v := range embedding {
		sumSquares += v * v
	}
	magnitude := float32(math.Sqrt(float64(sumSquares)))

	if magnitude > 0 {
		for i := range embedding {
			embedding[i] /= magnitude
		}
	}

	return embedding
}

// calculateRelevanceScore calculates a deterministic relevance score between query and document
// Score is between 0 and 1, with higher scores indicating more relevance
func (m *MockVoyageClient) calculateRelevanceScore(query, document string) float64 {
	// Simple deterministic scoring based on string similarity
	// In reality, this would use semantic similarity, but for testing we use a simple hash-based approach

	queryHash := sha256.Sum256([]byte(query))
	docHash := sha256.Sum256([]byte(document))

	// Calculate how many bytes are the same (pseudo-similarity)
	similarity := 0
	for i := 0; i < len(queryHash); i++ {
		if queryHash[i] == docHash[i] {
			similarity++
		}
	}

	// Normalize to 0-1 range and add some variation based on document length
	baseScore := float64(similarity) / float64(len(queryHash))

	// Add variation based on document content to ensure different scores
	docVal := binary.BigEndian.Uint32(docHash[:4])
	variation := float64(docVal%100) / 1000.0 // Small variation 0-0.1

	score := baseScore + variation
	if score > 1.0 {
		score = 1.0
	}

	return score
}

// SetEmbedError sets an error to be returned by Embed methods
func (m *MockVoyageClient) SetEmbedError(err error) {
	m.EmbedError = err
}

// SetRerankError sets an error to be returned by Rerank methods
func (m *MockVoyageClient) SetRerankError(err error) {
	m.RerankError = err
}

// Reset resets all call counters and tracked inputs
func (m *MockVoyageClient) Reset() {
	m.EmbedCallCount = 0
	m.EmbedBatchCallCount = 0
	m.EmbedMultimodalCallCount = 0
	m.RerankCallCount = 0
	m.LastEmbedInputs = nil
	m.LastMultimodalInputs = nil
	m.LastRerankQuery = ""
	m.LastRerankDocuments = nil
	m.EmbedError = nil
	m.RerankError = nil
}

// GetDimension returns the configured embedding dimension
func (m *MockVoyageClient) GetDimension() int {
	return m.dimension
}

// Verify that MockVoyageClient implements voyage.ClientInterface at compile time
var _ voyage.ClientInterface = (*MockVoyageClient)(nil)
