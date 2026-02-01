package voyage

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"time"
)

const (
	// Timeouts (hardcoded - rarely need customization)
	EmbeddingTimeout = 180 * time.Second // Longer for large batch operations
	RerankTimeout    = 30 * time.Second  // Shorter for quick reranking

	// Retry settings
	MaxRetries = 3
)

// ClientConfig holds configuration for the Voyage client
// All fields come from environment variables via config.Config
type ClientConfig struct {
	APIKey string // Required: VOYAGE_API_KEY

	// Embedding settings (from .env)
	EmbeddingModel    string // VOYAGE_MODEL (default: voyage-multimodal-3.5)
	EmbeddingEndpoint string // VOYAGE_EMBEDDING_ENDPOINT (default: Voyage AI)

	// Rerank settings (from .env)
	RerankModel    string // VOYAGE_RERANK_MODEL (default: rerank-2)
	RerankEndpoint string // VOYAGE_RERANK_ENDPOINT (default: Voyage AI)
}

// Client is a unified client for Voyage AI APIs (embeddings and reranking)
type Client struct {
	apiKey string

	// Embedding settings
	embeddingModel    string
	embeddingEndpoint string
	embeddingClient   *http.Client

	// Rerank settings
	rerankModel    string
	rerankEndpoint string
	rerankClient   *http.Client
}

// NewClient creates a new Voyage AI client
func NewClient(cfg ClientConfig) (*Client, error) {
	if cfg.APIKey == "" {
		return nil, fmt.Errorf("Voyage API key is required")
	}

	// Validate required fields (should come from config.Config)
	if cfg.EmbeddingModel == "" {
		return nil, fmt.Errorf("embedding model is required")
	}
	if cfg.EmbeddingEndpoint == "" {
		return nil, fmt.Errorf("embedding endpoint is required")
	}
	if cfg.RerankModel == "" {
		return nil, fmt.Errorf("rerank model is required")
	}
	if cfg.RerankEndpoint == "" {
		return nil, fmt.Errorf("rerank endpoint is required")
	}

	client := &Client{
		apiKey:            cfg.APIKey,
		embeddingModel:    cfg.EmbeddingModel,
		embeddingEndpoint: cfg.EmbeddingEndpoint,
		embeddingClient: &http.Client{
			Timeout: EmbeddingTimeout, // Hardcoded timeout
		},
		rerankModel:    cfg.RerankModel,
		rerankEndpoint: cfg.RerankEndpoint,
		rerankClient: &http.Client{
			Timeout: RerankTimeout, // Hardcoded timeout
		},
	}

	log.Printf("Voyage client initialized (embedding: %s, rerank: %s)", cfg.EmbeddingModel, cfg.RerankModel)

	return client, nil
}

// EmbeddingModel returns the configured embedding model
func (c *Client) EmbeddingModel() string {
	return c.embeddingModel
}

// RerankModel returns the configured rerank model
func (c *Client) RerankModel() string {
	return c.rerankModel
}

// doRequest performs an HTTP request with retry logic
func (c *Client) doRequest(ctx context.Context, httpClient *http.Client, endpoint string, reqBody interface{}) ([]byte, error) {
	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	var resp *http.Response
	var lastErr error

	for attempt := 0; attempt <= MaxRetries; attempt++ {
		if attempt > 0 {
			waitTime := time.Duration(attempt*attempt) * time.Second // 1s, 4s, 9s
			log.Printf("  Retry attempt %d/%d after %v", attempt, MaxRetries, waitTime)
			time.Sleep(waitTime)
		}

		req, err := http.NewRequestWithContext(ctx, "POST", endpoint, bytes.NewBuffer(jsonData))
		if err != nil {
			return nil, fmt.Errorf("failed to create request: %w", err)
		}

		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+c.apiKey)

		resp, err = httpClient.Do(req)
		if err != nil {
			lastErr = err
			if isTimeoutError(err) && attempt < MaxRetries {
				log.Printf("  Request timed out, will retry...")
				continue
			}
			return nil, fmt.Errorf("request failed: %w", err)
		}

		// Success - break out of retry loop
		break
	}

	if resp == nil {
		return nil, fmt.Errorf("all retries failed: %w", lastErr)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API error (status %d): %s", resp.StatusCode, string(body))
	}

	return body, nil
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
	if netErr, ok := err.(interface{ Timeout() bool }); ok && netErr.Timeout() {
		return true
	}

	// Check error string as fallback
	errStr := err.Error()
	return containsAny(errStr, []string{
		"context deadline exceeded",
		"Client.Timeout exceeded",
		"timeout",
		"timed out",
	})
}

func containsAny(s string, substrs []string) bool {
	for _, substr := range substrs {
		if len(s) >= len(substr) {
			for i := 0; i <= len(s)-len(substr); i++ {
				if s[i:i+len(substr)] == substr {
					return true
				}
			}
		}
	}
	return false
}
