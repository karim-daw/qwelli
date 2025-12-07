package indexer

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"time"
)

// OpenAIProvider implements EmbeddingProvider for OpenAI API
type OpenAIProvider struct {
	apiKey   string
	model    string
	endpoint string
	client   *http.Client
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
func NewOpenAIProvider(apiKey, model, endpoint string) (*OpenAIProvider, error) {
	if apiKey == "" {
		return nil, fmt.Errorf("OpenAI API key required")
	}
	if model == "" {
		return nil, fmt.Errorf("OpenAI model required")
	}
	if endpoint == "" {
		return nil, fmt.Errorf("OpenAI endpoint required")
	}

	p := &OpenAIProvider{
		apiKey:   apiKey,
		model:    model,
		endpoint: endpoint,
		client: &http.Client{
			Timeout: 60 * time.Second,
		},
	}

	log.Printf("✅ OpenAI provider initialized (model: %s)", model)
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
