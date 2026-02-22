package voyage

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"time"
)

const (
	// Default endpoints
	DefaultEmbeddingEndpoint = "https://api.voyageai.com/v1/multimodalembeddings"
	DefaultRerankEndpoint    = "https://api.voyageai.com/v1/rerank"

	// Default models
	DefaultEmbeddingModel = "voyage-multimodal-3"
	DefaultRerankModel    = "rerank-2"

	// Default timeouts
	DefaultEmbeddingTimeout = 180 * time.Second // Longer for large batch operations
	DefaultRerankTimeout    = 30 * time.Second  // Shorter for quick reranking

	// Concurrency settings
	DefaultMaxConcurrentBatches = 5

	// Retry settings
	MaxRetries = 3
)

// ClientConfig holds configuration for the Voyage client
type ClientConfig struct {
	APIKey string

	// Embedding settings
	EmbeddingModel    string
	EmbeddingEndpoint string
	EmbeddingTimeout  time.Duration

	// Concurrency settings
	MaxConcurrentBatches int // Max parallel embedding API calls (default: 5)

	// Rerank settings
	RerankModel    string
	RerankEndpoint string
	RerankTimeout  time.Duration
}

// Client is a unified client for Voyage AI APIs (embeddings and reranking)
type Client struct {
	apiKey string

	// Embedding settings
	embeddingModel    string
	embeddingEndpoint string
	embeddingClient   *http.Client

	// Concurrency settings
	maxConcurrentBatches int

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

	// Apply defaults
	if cfg.EmbeddingModel == "" {
		cfg.EmbeddingModel = DefaultEmbeddingModel
	}
	if cfg.EmbeddingEndpoint == "" {
		cfg.EmbeddingEndpoint = DefaultEmbeddingEndpoint
	}
	if cfg.EmbeddingTimeout == 0 {
		cfg.EmbeddingTimeout = DefaultEmbeddingTimeout
	}
	if cfg.RerankModel == "" {
		cfg.RerankModel = DefaultRerankModel
	}
	if cfg.RerankEndpoint == "" {
		cfg.RerankEndpoint = DefaultRerankEndpoint
	}
	if cfg.RerankTimeout == 0 {
		cfg.RerankTimeout = DefaultRerankTimeout
	}
	if cfg.MaxConcurrentBatches <= 0 {
		cfg.MaxConcurrentBatches = DefaultMaxConcurrentBatches
	}

	client := &Client{
		apiKey:            cfg.APIKey,
		embeddingModel:    cfg.EmbeddingModel,
		embeddingEndpoint: cfg.EmbeddingEndpoint,
		embeddingClient: &http.Client{
			Timeout: cfg.EmbeddingTimeout,
		},
		maxConcurrentBatches: cfg.MaxConcurrentBatches,
		rerankModel:          cfg.RerankModel,
		rerankEndpoint:       cfg.RerankEndpoint,
		rerankClient: &http.Client{
			Timeout: cfg.RerankTimeout,
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

// doRequest performs an HTTP request with retry logic for timeouts, rate limits, and server errors
func (c *Client) doRequest(ctx context.Context, httpClient *http.Client, endpoint string, reqBody interface{}) ([]byte, error) {
	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	var lastErr error

	for attempt := 0; attempt <= MaxRetries; attempt++ {
		if attempt > 0 {
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			default:
			}
		}

		req, err := http.NewRequestWithContext(ctx, "POST", endpoint, bytes.NewBuffer(jsonData))
		if err != nil {
			return nil, fmt.Errorf("failed to create request: %w", err)
		}

		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+c.apiKey)

		resp, err := httpClient.Do(req)
		if err != nil {
			lastErr = err
			if isTimeoutError(err) && attempt < MaxRetries {
				waitTime := time.Duration(attempt+1) * time.Duration(attempt+1) * time.Second
				log.Printf("  Request timed out, retrying in %v (%d/%d)...", waitTime, attempt+1, MaxRetries)
				select {
				case <-ctx.Done():
					return nil, ctx.Err()
				case <-time.After(waitTime):
				}
				continue
			}
			return nil, fmt.Errorf("request failed: %w", err)
		}

		body, err := io.ReadAll(resp.Body)
		resp.Body.Close()
		if err != nil {
			return nil, fmt.Errorf("failed to read response: %w", err)
		}

		if resp.StatusCode == http.StatusOK {
			return body, nil
		}

		// Retry on 429 (rate limit) or 5xx (server error)
		if (resp.StatusCode == 429 || resp.StatusCode >= 500) && attempt < MaxRetries {
			waitTime := 30 * time.Second * time.Duration(attempt+1) // 30s, 60s, 90s
			if resp.StatusCode == 429 {
				log.Printf("  Rate limited (429), waiting %v before retry (%d/%d)...", waitTime, attempt+1, MaxRetries)
			} else {
				log.Printf("  Server error (%d), waiting %v before retry (%d/%d)...", resp.StatusCode, waitTime, attempt+1, MaxRetries)
			}
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(waitTime):
			}
			lastErr = fmt.Errorf("API error (status %d): %s", resp.StatusCode, string(body))
			continue
		}

		return nil, fmt.Errorf("API error (status %d): %s", resp.StatusCode, string(body))
	}

	return nil, fmt.Errorf("all retries failed: %w", lastErr)
}

func isTimeoutError(err error) bool {
	if err == nil {
		return false
	}
	if err == context.DeadlineExceeded {
		return true
	}
	if netErr, ok := err.(interface{ Timeout() bool }); ok && netErr.Timeout() {
		return true
	}
	s := err.Error()
	return strings.Contains(s, "timeout") || strings.Contains(s, "timed out") ||
		strings.Contains(s, "context deadline exceeded")
}
