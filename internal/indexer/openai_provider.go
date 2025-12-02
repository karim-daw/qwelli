package indexer

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"time"
)

const (
	defaultOpenAIEndpoint = "https://api.openai.com/v1/embeddings"
	defaultOpenAIModel    = "text-embedding-3-small"
)

// OpenAIProvider implements EmbeddingProvider for OpenAI API
type OpenAIProvider struct {
	apiKey    string
	model     string
	endpoint  string
	dimension int
	client    *http.Client
}

// OpenAI API request/response structures
type openAIRequest struct {
	Model string `json:"model"`
	Input any    `json:"input"` // string or []string
}

type openAIResponse struct {
	Data []struct {
		Embedding []float64 `json:"embedding"`
	} `json:"data"`
}

// NewOpenAIProvider creates a new OpenAI embedding provider
func NewOpenAIProvider(cfg Config) (*OpenAIProvider, error) {
	apiKey := cfg.APIKey
	if apiKey == "" {
		apiKey = os.Getenv("OPENAI_API_KEY")
	}
	if apiKey == "" {
		return nil, fmt.Errorf("OpenAI API key required (set OPENAI_API_KEY env var or pass in config)")
	}

	model := cfg.Model
	if model == "" {
		model = defaultOpenAIModel
	}

	endpoint := cfg.Endpoint
	if endpoint == "" {
		endpoint = defaultOpenAIEndpoint
	}

	p := &OpenAIProvider{
		apiKey:   apiKey,
		model:    model,
		endpoint: endpoint,
		client: &http.Client{
			Timeout: 60 * time.Second,
		},
	}

	// Auto-detect dimension if not provided
	if cfg.Dimension > 0 {
		p.dimension = cfg.Dimension
	} else {
		dim, err := p.detectDimension()
		if err != nil {
			return nil, fmt.Errorf("failed to detect dimension: %w", err)
		}
		p.dimension = dim
	}

	log.Printf("✅ OpenAI provider initialized (model: %s, dimension: %d)", model, p.dimension)
	return p, nil
}

// Embed generates an embedding for a single text
func (p *OpenAIProvider) Embed(text string) ([]float32, error) {
	start := time.Now()

	embeddings, err := p.callAPI(text)
	if err != nil {
		return nil, err
	}

	if len(embeddings) != 1 {
		return nil, fmt.Errorf("expected 1 embedding, got %d", len(embeddings))
	}

	log.Printf("⏱️  Generated embedding (%d chars) in %v", len(text), time.Since(start))
	return embeddings[0], nil
}

// EmbedBatch generates embeddings for multiple texts (true batching)
func (p *OpenAIProvider) EmbedBatch(texts []string) ([][]float32, error) {
	start := time.Now()

	embeddings, err := p.callAPI(texts)
	if err != nil {
		return nil, err
	}

	if len(embeddings) != len(texts) {
		return nil, fmt.Errorf("expected %d embeddings, got %d", len(texts), len(embeddings))
	}

	log.Printf("⏱️  Generated %d embeddings in batch: %v (avg: %v per embedding)",
		len(texts), time.Since(start), time.Since(start)/time.Duration(len(texts)))
	return embeddings, nil
}

// Dimension returns the embedding dimension
func (p *OpenAIProvider) Dimension() int {
	return p.dimension
}

// Close cleans up resources (no-op for OpenAI)
func (p *OpenAIProvider) Close() error {
	return nil
}

// callAPI makes the API call to OpenAI
func (p *OpenAIProvider) callAPI(input any) ([][]float32, error) {
	reqBody := openAIRequest{
		Model: p.model,
		Input: input,
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequest("POST", p.endpoint, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+p.apiKey)

	resp, err := p.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("API request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("API error (status %d): %s", resp.StatusCode, string(body))
	}

	var apiResp openAIResponse
	if err := json.NewDecoder(resp.Body).Decode(&apiResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	// Convert []float64 to [][]float32
	embeddings := make([][]float32, len(apiResp.Data))
	for i, data := range apiResp.Data {
		embedding := make([]float32, len(data.Embedding))
		for j, v := range data.Embedding {
			embedding[j] = float32(v)
		}
		embeddings[i] = embedding
	}

	return embeddings, nil
}

// detectDimension detects the embedding dimension by making a test call
func (p *OpenAIProvider) detectDimension() (int, error) {
	log.Println("🔢 Detecting embedding dimension...")
	embeddings, err := p.callAPI("test")
	if err != nil {
		return 0, err
	}
	return len(embeddings[0]), nil
}
