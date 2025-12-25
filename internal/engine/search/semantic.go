package search

import (
	"github.com/karim-daw/qwelli/internal/db"
	"github.com/karim-daw/qwelli/internal/engine/indexer"
)

// SemanticSearchStrategy performs semantic search using vector embeddings
type SemanticSearchStrategy struct {
	apiKey   string
	model    string
	endpoint string
}

// NewSemanticSearchStrategy creates a new semantic search strategy
func NewSemanticSearchStrategy() *SemanticSearchStrategy {
	return &SemanticSearchStrategy{}
}

// NewSemanticSearchStrategyWithConfig creates a semantic search strategy with config
func NewSemanticSearchStrategyWithConfig(apiKey, model, endpoint string) *SemanticSearchStrategy {
	return &SemanticSearchStrategy{
		apiKey:   apiKey,
		model:    model,
		endpoint: endpoint,
	}
}

// Name returns the strategy name
func (s *SemanticSearchStrategy) Name() string {
	return "semantic"
}

// Search performs semantic search using vector embeddings
func (s *SemanticSearchStrategy) Search(query string, projectDB *db.ProjectDB, topK int, contentType string) ([]db.SearchResult, error) {
	// Create embedder if not already created
	embedder, err := indexer.NewEmbedder(s.apiKey, s.model, s.endpoint)
	if err != nil {
		return nil, err
	}

	// Generate query embedding
	queryVec, err := embedder.Embed(query)
	if err != nil {
		return nil, err
	}

	// Perform ANN search
	if contentType != "" {
		return projectDB.SearchANNWithFilter(queryVec, topK, contentType)
	}
	return projectDB.SearchANN(queryVec, topK)
}
