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

	client := &Client{
		apiKey:            cfg.APIKey,
		embeddingModel:    cfg.EmbeddingModel,
		embeddingEndpoint: cfg.EmbeddingEndpoint,
		embeddingClient: &http.Client{
			Timeout: cfg.EmbeddingTimeout,
		},
		rerankModel:    cfg.RerankModel,
		rerankEndpoint: cfg.RerankEndpoint,
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
