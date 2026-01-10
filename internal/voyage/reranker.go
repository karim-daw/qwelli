package voyage

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"sort"
	"time"
)

// Rerank reranks documents based on their relevance to a query
// Returns the documents sorted by relevance score (highest first)
func (c *Client) Rerank(ctx context.Context, query string, documents []string) ([]RerankResult, error) {
	if len(documents) == 0 {
		log.Printf("Reranking skipped: no documents to rerank")
		return []RerankResult{}, nil
	}

	startTime := time.Now()
	log.Printf("Reranking %d documents for query: %s", len(documents), truncateString(query, 50))

	reqBody := rerankRequest{
		Query:     query,
		Documents: documents,
		Model:     c.rerankModel,
	}

	respBody, err := c.doRequest(ctx, c.rerankClient, c.rerankEndpoint, reqBody)
	if err != nil {
		return nil, fmt.Errorf("rerank request failed: %w", err)
	}

	var resp rerankResponse
	if err := json.Unmarshal(respBody, &resp); err != nil {
		return nil, fmt.Errorf("failed to parse rerank response: %w", err)
	}

	// Build results with relevance scores
	results := make([]RerankResult, 0, len(resp.Data))
	var topResultOldPosition int
	var topRelevanceScore float64

	for _, item := range resp.Data {
		if item.Index < 0 || item.Index >= len(documents) {
			log.Printf("  Invalid index %d in rerank response (max: %d)", item.Index, len(documents)-1)
			continue
		}

		results = append(results, RerankResult{
			Index:          item.Index,
			RelevanceScore: item.RelevanceScore,
			Content:        documents[item.Index],
		})

		// Track which original position became the top result
		if len(results) == 1 || item.RelevanceScore > topRelevanceScore {
			topRelevanceScore = item.RelevanceScore
			topResultOldPosition = item.Index
		}
	}

	// Sort by relevance score (descending - higher is better)
	sort.Slice(results, func(i, j int) bool {
		return results[i].RelevanceScore > results[j].RelevanceScore
	})

	elapsed := time.Since(startTime)
	log.Printf("  Reranking completed in %v (%d tokens used)", elapsed.Round(time.Millisecond), resp.Usage.TotalTokens)

	// Show improvement
	if len(results) > 0 {
		if topResultOldPosition == 0 {
			log.Printf("  Top result unchanged (relevance: %.3f)", topRelevanceScore)
		} else {
			log.Printf("  Result #%d promoted to top (relevance: %.3f)", topResultOldPosition+1, topRelevanceScore)
		}
	}

	return results, nil
}

// RerankWithMetadata reranks documents while preserving associated metadata
func (c *Client) RerankWithMetadata(ctx context.Context, query string, inputs []RerankInput) ([]RerankResult, error) {
	if len(inputs) == 0 {
		log.Printf("Reranking skipped: no inputs to rerank")
		return []RerankResult{}, nil
	}

	// Extract just the content for the API call
	documents := make([]string, len(inputs))
	for i, input := range inputs {
		documents[i] = input.Content
	}

	results, err := c.Rerank(ctx, query, documents)
	if err != nil {
		return nil, err
	}

	// Attach metadata back to results
	for i := range results {
		if results[i].Index >= 0 && results[i].Index < len(inputs) {
			results[i].Metadata = inputs[results[i].Index].Metadata
		}
	}

	return results, nil
}

// truncateString truncates a string to the specified length with ellipsis
func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}
