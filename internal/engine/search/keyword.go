package search

import (
	"context"

	"github.com/karim-daw/qwelli/internal/db"
)

// KeywordSearchStrategy performs keyword search using full-text search (FTS)
// Uses DuckDB's LIKE queries with TF-IDF-like relevance scoring
type KeywordSearchStrategy struct{}

// NewKeywordSearchStrategy creates a new keyword search strategy
func NewKeywordSearchStrategy() *KeywordSearchStrategy {
	return &KeywordSearchStrategy{}
}

// Name returns the strategy name
func (s *KeywordSearchStrategy) Name() string {
	return "keyword"
}

// Search performs keyword search using full-text search
// Uses the database's SearchFTS method which implements:
// 1. Query tokenization (splitting into keywords, removing stop words)
// 2. LIKE-based matching (case-insensitive)
// 3. TF-IDF-like relevance scoring
// 4. Top-K result limiting
// Note: Content type filtering removed (multimodal mode searches all content)
func (s *KeywordSearchStrategy) Search(query string, projectDB *db.ProjectDB, topK int, contentType string) ([]db.SearchResult, error) {
	return projectDB.SearchFTS(context.Background(), query, topK, "") // Always search all content
}
