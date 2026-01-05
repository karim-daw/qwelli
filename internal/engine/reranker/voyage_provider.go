package reranker

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"sort"
	"time"
)

const (
	defaultVoyageRerankEndpoint = "https://api.voyageai.com/v1/rerank"
	defaultVoyageRerankModel    = "rerank-2"
	defaultTimeout              = 30 * time.Second
	maxRetries                  = 3
)

// VoyageProvider implements the Provider interface using Voyage AI's reranking API
type VoyageProvider struct {
	apiKey   string
	model    string
	endpoint string
	client   *http.Client
}

// NewVoyageProvider creates a new Voyage reranker provider
func NewVoyageProvider(apiKey string, model string, endpoint string) (*VoyageProvider, error) {
	if apiKey == "" {
		return nil, fmt.Errorf("Voyage API key is required for reranking")
	}

	if model == "" {
		model = defaultVoyageRerankModel
	}

	if endpoint == "" {
		endpoint = defaultVoyageRerankEndpoint
	}

	provider := &VoyageProvider{
		apiKey:   apiKey,
		model:    model,
		endpoint: endpoint,
		client: &http.Client{
			Timeout: defaultTimeout,
		},
	}

	log.Printf("🔄 Voyage reranker initialized (model: %s)", model)

	return provider, nil
}

// Name returns the name of this provider
func (p *VoyageProvider) Name() string {
	return "voyage"
}

// voyageRerankRequest represents the request to Voyage's rerank API
type voyageRerankRequest struct {
	Query     string   `json:"query"`
	Documents []string `json:"documents"`
	Model     string   `json:"model"`
	TopK      *int     `json:"top_k,omitempty"` // Optional: limit number of results
}

// voyageRerankResponse represents the response from Voyage's rerank API
type voyageRerankResponse struct {
	Object string `json:"object"`
	Data   []struct {
		Index         int     `json:"index"`
		RelevanceScore float64 `json:"relevance_score"`
	} `json:"data"`
	Model string `json:"model"`
	Usage struct {
		TotalTokens int `json:"total_tokens"`
	} `json:"usage"`
}

// Rerank reranks search results using Voyage AI's reranking API
func (p *VoyageProvider) Rerank(ctx context.Context, query string, results []SearchResult) ([]SearchResult, error) {
	if len(results) == 0 {
		log.Printf("🔄 Reranking skipped: no results to rerank")
		return results, nil
	}

	startTime := time.Now()
	log.Printf("🔄 Reranking %d results for query: %s", len(results), truncateString(query, 50))

	// Extract documents from search results
	documents := make([]string, len(results))
	for i, result := range results {
		documents[i] = result.Content
	}

	// Prepare request
	reqBody := voyageRerankRequest{
		Query:     query,
		Documents: documents,
		Model:     p.model,
	}

	// Try with retries
	var resp *voyageRerankResponse
	var err error
	for attempt := 0; attempt <= maxRetries; attempt++ {
		if attempt > 0 {
			delay := time.Duration(attempt*attempt) * time.Second
			log.Printf("  ⏳ Retry attempt %d/%d after %v", attempt, maxRetries, delay)
			time.Sleep(delay)
		}

		resp, err = p.makeRerankRequest(ctx, reqBody)
		if err == nil {
			break
		}

		if attempt == maxRetries {
			return nil, fmt.Errorf("rerank request failed after %d retries: %w", maxRetries, err)
		}

		if isTimeoutError(err) {
			log.Printf("  ⚠️  Request timed out, will retry...")
		} else {
			// For non-timeout errors, fail immediately
			return nil, fmt.Errorf("rerank request failed: %w", err)
		}
	}

	// Map relevance scores back to results
	rerankedResults := make([]SearchResult, 0, len(resp.Data))
	var topResultOldPosition int
	var topRelevanceScore float64

	for _, item := range resp.Data {
		if item.Index < 0 || item.Index >= len(results) {
			log.Printf("  ⚠️  Invalid index %d in rerank response (max: %d)", item.Index, len(results)-1)
			continue
		}

		result := results[item.Index]
		// Update the distance field with the relevance score
		// Note: Higher relevance score is better, unlike distance metrics
		// We use negative relevance score to maintain the "lower is better" convention
		result.Distance = -item.RelevanceScore
		rerankedResults = append(rerankedResults, result)

		// Track which original position became the top result
		if len(rerankedResults) == 1 || item.RelevanceScore > topRelevanceScore {
			topRelevanceScore = item.RelevanceScore
			topResultOldPosition = item.Index
		}
	}

	// Sort by relevance score (descending - higher is better, which means lower Distance)
	sort.Slice(rerankedResults, func(i, j int) bool {
		return rerankedResults[i].Distance < rerankedResults[j].Distance
	})

	elapsed := time.Since(startTime)
	log.Printf("  ✓ Reranking completed in %v (%d tokens used)", elapsed.Round(time.Millisecond), resp.Usage.TotalTokens)

	// Show improvement
	if len(rerankedResults) > 0 {
		if topResultOldPosition == 0 {
			log.Printf("  📊 Top result unchanged (relevance: %.3f)", topRelevanceScore)
		} else {
			log.Printf("  📊 Result #%d promoted to top (relevance: %.3f)", topResultOldPosition+1, topRelevanceScore)
		}
	}

	return rerankedResults, nil
}

// makeRerankRequest performs the actual HTTP request to Voyage's rerank API
func (p *VoyageProvider) makeRerankRequest(ctx context.Context, reqBody voyageRerankRequest) (*voyageRerankResponse, error) {
	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", p.endpoint, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+p.apiKey)

	resp, err := p.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API error (status %d): %s", resp.StatusCode, string(body))
	}

	var rerankResp voyageRerankResponse
	if err := json.Unmarshal(body, &rerankResp); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return &rerankResp, nil
}

// isTimeoutError checks if an error is a timeout error
func isTimeoutError(err error) bool {
	if err == nil {
		return false
	}
	// Check for context deadline exceeded
	if err == context.DeadlineExceeded {
		return true
	}
	// Check for http.Client timeout
	netErr, ok := err.(interface{ Timeout() bool })
	return ok && netErr.Timeout()
}

// truncateString truncates a string to the specified length with ellipsis
func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}
