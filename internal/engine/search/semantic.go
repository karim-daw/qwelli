package search

import (
	"context"

	"github.com/karim-daw/qwelli/internal/db"
	"github.com/karim-daw/qwelli/internal/engine/embeddings"
	"github.com/karim-daw/qwelli/internal/voyage"
)

// SemanticSearchStrategy performs semantic search using vector embeddings
type SemanticSearchStrategy struct {
	client *voyage.Client
}

// NewSemanticSearchStrategy creates a semantic search strategy with a Voyage client
func NewSemanticSearchStrategy(client *voyage.Client) *SemanticSearchStrategy {
	return &SemanticSearchStrategy{client: client}
}

// Name returns the strategy name
func (s *SemanticSearchStrategy) Name() string {
	return "semantic"
}

// Search performs semantic search using vector embeddings
func (s *SemanticSearchStrategy) Search(query string, projectDB *db.ProjectDB, topK int, contentType string) ([]db.SearchResult, error) {
	embedder, err := embeddings.NewEmbedder(s.client)
	if err != nil {
		return nil, err
	}

	// Generate query embedding
	queryVec, err := embedder.Embed(query)
	if err != nil {
		return nil, err
	}

	// Perform ANN search
	ctx := context.Background()
	if contentType != "" {
		return projectDB.SearchANNWithFilter(ctx, queryVec, topK, contentType)
	}
	return projectDB.SearchANN(ctx, queryVec, topK)
}
