package search

import (
	"context"
	"log"

	"github.com/karim-daw/qwelli/internal/engine"
	"github.com/karim-daw/qwelli/internal/voyage"
)

// RerankOptions controls reranking behavior
type RerankOptions struct {
	Enabled bool // Whether to apply reranking
	Verbose bool // Whether to show detailed logs
}

// ApplyReranking reranks search results using Voyage AI for better relevance
// Returns: (reorderedResults, wasReranked)
// On error, returns original results with wasReranked=false
func ApplyReranking(
	ctx context.Context,
	client *voyage.Client,
	query string,
	results []engine.SearchResult,
	opts RerankOptions,
) ([]engine.SearchResult, bool) {
	// Return early if disabled or no results
	if !opts.Enabled || len(results) == 0 {
		return results, false
	}

	if opts.Verbose {
		log.Printf("🔄 Reranking %d documents...", len(results))
	}

	// Extract content from results
	documents := make([]string, len(results))
	for i, r := range results {
		documents[i] = r.Content
	}

	// Call Voyage rerank API
	reranked, err := client.Rerank(ctx, query, documents)
	if err != nil {
		log.Printf("⚠️  Reranking failed: %v (using original results)", err)
		return results, false
	}

	// Reorder results based on relevance scores
	reordered := make([]engine.SearchResult, 0, len(reranked))
	for _, item := range reranked {
		if item.Index >= 0 && item.Index < len(results) {
			result := results[item.Index]
			// Update distance with negative relevance score
			// (negative because lower distance = better match in vector search)
			result.Distance = -item.RelevanceScore
			reordered = append(reordered, result)
		}
	}

	// Log improvement if verbose
	if opts.Verbose && len(reranked) > 0 {
		logRerankImprovement(reranked)
	}

	return reordered, true
}

// logRerankImprovement shows which result was promoted to top
func logRerankImprovement(reranked []voyage.RerankResult) {
	if len(reranked) == 0 {
		return
	}

	topResult := reranked[0]
	if topResult.Index == 0 {
		log.Printf("  ✓ Top result unchanged (relevance: %.3f)", topResult.RelevanceScore)
	} else {
		log.Printf("  ↑ Result #%d promoted to top (relevance: %.3f)",
			topResult.Index+1, topResult.RelevanceScore)
	}
}
