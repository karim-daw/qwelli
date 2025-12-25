package search

import (
	"github.com/karim-daw/qwelli/internal/db"
)

// SearchStrategy defines the interface for different search strategies
// This allows for extensible search methods:
// - SemanticSearchStrategy: vector similarity search (current)
// - KeywordSearchStrategy: full-text search (FTS)
// - HybridSearchStrategy: weighted combination of semantic + keyword
type SearchStrategy interface {
	// Search performs a search and returns results
	// query: the search query string
	// projectDB: the database connection
	// topK: number of results to return
	// contentType: optional filter ("text", "image", or "")
	Search(query string, projectDB *db.ProjectDB, topK int, contentType string) ([]db.SearchResult, error)

	// Name returns the strategy name for identification
	Name() string
}

// SearchOptions contains options for search strategies
type SearchOptions struct {
	ContentType string // Optional: "text", "image", or ""
	TopK        int    // Number of results
}

// DefaultSearchOptions returns default search options
func DefaultSearchOptions() SearchOptions {
	return SearchOptions{
		ContentType: "",
		TopK:        5,
	}
}

// Registry manages search strategies
type Registry struct {
	strategies      map[string]SearchStrategy
	defaultStrategy string
}

// NewRegistry creates a new search strategy registry with default strategies
func NewRegistry() *Registry {
	semantic := NewSemanticSearchStrategy()
	registry := &Registry{
		strategies:      make(map[string]SearchStrategy),
		defaultStrategy: "semantic",
	}
	registry.Register("semantic", semantic)
	return registry
}

// Register adds a search strategy to the registry
func (r *Registry) Register(name string, strategy SearchStrategy) {
	r.strategies[name] = strategy
}

// GetStrategy returns a strategy by name, or default if not found
func (r *Registry) GetStrategy(name string) SearchStrategy {
	if strategy, ok := r.strategies[name]; ok {
		return strategy
	}
	return r.strategies[r.defaultStrategy]
}

// SetDefault sets the default strategy
func (r *Registry) SetDefault(name string) {
	if _, ok := r.strategies[name]; ok {
		r.defaultStrategy = name
	}
}

// DefaultRegistry is the default global registry
var DefaultRegistry = NewRegistry()

// GetStrategy returns a strategy from the default registry
func GetStrategy(name string) SearchStrategy {
	return DefaultRegistry.GetStrategy(name)
}
