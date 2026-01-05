package reranker

import (
	"context"
)

// SearchResult represents a search result to be reranked
// This mirrors the engine.SearchResult structure to avoid circular dependencies
type SearchResult struct {
	FilePath     string
	FileName     string
	Distance     float64 // Will be replaced with relevance score after reranking
	Content      string
	TextMetadata map[string]interface{}
	ImageData    []byte
	PageNumbers  []int
}

// Provider defines the interface for reranking search results
type Provider interface {
	// Rerank takes a query and a list of search results and returns them reranked
	// with updated relevance scores based on the reranker model
	Rerank(ctx context.Context, query string, results []SearchResult) ([]SearchResult, error)

	// Name returns the name of the reranker provider
	Name() string
}
