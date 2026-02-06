package search

import (
	"github.com/karim-daw/qwelli/internal/db"
)

// SearchStrategy defines the interface for different search approaches
type SearchStrategy interface {
	Search(query string, projectDB *db.ProjectDB, topK int, contentType string) ([]db.SearchResult, error)
	Name() string
}
