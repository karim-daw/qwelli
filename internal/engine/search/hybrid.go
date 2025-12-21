package search

import (
	"fmt"
	"sort"

	"github.com/karim-daw/qwelli/internal/db"
)

// HybridSearchStrategy performs hybrid search combining semantic and keyword search
// Results are weighted and merged
type HybridSearchStrategy struct {
	semanticStrategy *SemanticSearchStrategy
	keywordStrategy  *KeywordSearchStrategy
	semanticWeight   float64 // Weight for semantic results (0.0 to 1.0)
	keywordWeight    float64 // Weight for keyword results (0.0 to 1.0)
}

// HybridSearchConfig contains configuration for hybrid search
type HybridSearchConfig struct {
	SemanticWeight float64 // Weight for semantic results (default: 0.7)
	KeywordWeight  float64 // Weight for keyword results (default: 0.3)
}

// DefaultHybridConfig returns default hybrid search configuration
func DefaultHybridConfig() HybridSearchConfig {
	return HybridSearchConfig{
		SemanticWeight: 0.7,
		KeywordWeight:  0.3,
	}
}

// NewHybridSearchStrategy creates a new hybrid search strategy
func NewHybridSearchStrategy(semantic *SemanticSearchStrategy, keyword *KeywordSearchStrategy) *HybridSearchStrategy {
	config := DefaultHybridConfig()
	return &HybridSearchStrategy{
		semanticStrategy: semantic,
		keywordStrategy:  keyword,
		semanticWeight:   config.SemanticWeight,
		keywordWeight:    config.KeywordWeight,
	}
}

// NewHybridSearchStrategyWithConfig creates a hybrid search strategy with custom weights
func NewHybridSearchStrategyWithConfig(semantic *SemanticSearchStrategy, keyword *KeywordSearchStrategy, config HybridSearchConfig) *HybridSearchStrategy {
	return &HybridSearchStrategy{
		semanticStrategy: semantic,
		keywordStrategy:  keyword,
		semanticWeight:   config.SemanticWeight,
		keywordWeight:    config.KeywordWeight,
	}
}

// Name returns the strategy name
func (s *HybridSearchStrategy) Name() string {
	return "hybrid"
}

// Search performs hybrid search by combining semantic and keyword results
func (s *HybridSearchStrategy) Search(query string, projectDB *db.ProjectDB, topK int, contentType string) ([]db.SearchResult, error) {
	// Get semantic results
	semanticResults, err := s.semanticStrategy.Search(query, projectDB, topK*2, contentType)
	if err != nil {
		return nil, fmt.Errorf("semantic search failed: %w", err)
	}

	// Get keyword results
	keywordResults, err := s.keywordStrategy.Search(query, projectDB, topK*2, contentType)
	if err != nil {
		return nil, fmt.Errorf("keyword search failed: %w", err)
	}

	// Merge and rank results
	merged := s.mergeResults(semanticResults, keywordResults, topK)
	return merged, nil
}

// mergeResults merges semantic and keyword results with weighted scoring
func (s *HybridSearchStrategy) mergeResults(semantic, keyword []db.SearchResult, topK int) []db.SearchResult {
	// Create a map to track results by chunk ID
	resultMap := make(map[string]*db.SearchResult)

	// Process semantic results
	// Distance is already normalized (0.0 = perfect match, 1.0 = no match)
	// Convert to score (higher is better): score = 1.0 - distance
	for i := range semantic {
		chunkID := semantic[i].ChunkID
		score := (1.0 - semantic[i].Distance) * s.semanticWeight

		if existing, ok := resultMap[chunkID]; ok {
			// Already exists, add to score
			existing.Distance = existing.Distance - score // Lower distance = better
		} else {
			result := semantic[i]
			result.Distance = 1.0 - score // Store as distance (lower is better)
			resultMap[chunkID] = &result
		}
	}

	// Process keyword results
	// Distance in keyword results is already normalized (0.0 = perfect match, 1.0 = no match)
	// Convert to score (higher is better): score = 1.0 - distance
	for i := range keyword {
		chunkID := keyword[i].ChunkID
		score := (1.0 - keyword[i].Distance) * s.keywordWeight

		if existing, ok := resultMap[chunkID]; ok {
			// Already exists, add to score
			existing.Distance = existing.Distance - score // Lower distance = better
		} else {
			result := keyword[i]
			result.Distance = 1.0 - score // Store as distance (lower is better)
			resultMap[chunkID] = &result
		}
	}

	// Convert map to slice and sort by distance (lower is better)
	results := make([]db.SearchResult, 0, len(resultMap))
	for _, result := range resultMap {
		results = append(results, *result)
	}

	// Sort by distance (ascending - lower distance = better match)
	sort.Slice(results, func(i, j int) bool {
		return results[i].Distance < results[j].Distance
	})

	// Return topK results
	if len(results) > topK {
		return results[:topK]
	}
	return results
}
