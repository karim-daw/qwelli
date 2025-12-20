package indexer

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"time"

	"github.com/karim-daw/qwelli/internal/processor"
)

// HTTPEmbeddingProvider implements EmbeddingProvider for HTTP-based embedding APIs
// Works with OpenAI, VoyageAI, and other compatible APIs
type HTTPEmbeddingProvider struct {
	apiKey   string
	model    string
	endpoint string
	client   *http.Client
}

// Embedding API request/response structures (OpenAI-compatible format)
type embeddingRequest struct {
	Model string `json:"model"`
	Input any    `json:"input"` // string or []string
}

type embeddingResponse struct {
	Data []struct {
		Embedding []float64 `json:"embedding"`
	} `json:"data"`
}

// NewHTTPEmbeddingProvider creates a new HTTP-based embedding provider
// Works with OpenAI, VoyageAI, and other compatible APIs
func NewHTTPEmbeddingProvider(apiKey, model, endpoint string) (*HTTPEmbeddingProvider, error) {
	if apiKey == "" {
		return nil, fmt.Errorf("API key required")
	}
	if model == "" {
		return nil, fmt.Errorf("model required")
	}
	if endpoint == "" {
		return nil, fmt.Errorf("endpoint required")
	}

	p := &HTTPEmbeddingProvider{
		apiKey:   apiKey,
		model:    model,
		endpoint: endpoint,
		client: &http.Client{
			Timeout: 60 * time.Second,
		},
	}

	log.Printf("✅ Embedding provider initialized (model: %s)", model)
	return p, nil
}

// Embed generates an embedding for a single text
func (p *HTTPEmbeddingProvider) Embed(text string) ([]float32, error) {
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

// EmbedBatch generates embeddings for multiple texts with smart batching
// to optimize API calls while respecting token limits (8192 tokens default).
func (p *HTTPEmbeddingProvider) EmbedBatch(texts []string) ([][]float32, error) {
	start := time.Now()

	if len(texts) == 0 {
		return [][]float32{}, nil
	}

	// Create batches that respect token limits
	batches := p.createBatches(texts)
	log.Printf("  Split %d documents into %d batch(es)", len(texts), len(batches))

	allEmbeddings := make([][]float32, 0, len(texts))
	totalAPICalls := 0

	for batchIdx, batch := range batches {
		log.Printf("  Processing batch %d/%d (%d documents, ~%d tokens)...",
			batchIdx+1, len(batches), len(batch.texts), batch.estimatedTokens)

		embeddings, err := p.callAPI(batch.texts)
		if err != nil {
			return nil, fmt.Errorf("failed to process batch %d: %w", batchIdx+1, err)
		}

		if len(embeddings) != len(batch.texts) {
			return nil, fmt.Errorf("batch %d: expected %d embeddings, got %d",
				batchIdx+1, len(batch.texts), len(embeddings))
		}

		allEmbeddings = append(allEmbeddings, embeddings...)
		totalAPICalls++
	}

	log.Printf("⏱️  Generated %d embeddings in %d API call(s): %v (avg: %v per embedding)",
		len(texts), totalAPICalls, time.Since(start), time.Since(start)/time.Duration(len(texts)))
	return allEmbeddings, nil
}

// batch represents a group of texts to embed together
type batch struct {
	texts           []string
	estimatedTokens int
}

// createBatches groups texts into batches that respect token limits
func (p *HTTPEmbeddingProvider) createBatches(texts []string) []batch {
	const maxTokensPerBatch = 8000 // Conservative limit (actual is 8192 for most providers)

	batches := []batch{}
	currentBatch := batch{
		texts:           []string{},
		estimatedTokens: 0,
	}

	for _, text := range texts {
		estimatedTokens := processor.EstimateTokens(text)

		// If this single text exceeds the limit, it goes in its own batch
		if estimatedTokens > maxTokensPerBatch {
			// Flush current batch if not empty
			if len(currentBatch.texts) > 0 {
				batches = append(batches, currentBatch)
				currentBatch = batch{texts: []string{}, estimatedTokens: 0}
			}
			// Add large text as its own batch
			batches = append(batches, batch{
				texts:           []string{text},
				estimatedTokens: estimatedTokens,
			})
			continue
		}

		// If adding this text would exceed the limit, start a new batch
		if currentBatch.estimatedTokens+estimatedTokens > maxTokensPerBatch && len(currentBatch.texts) > 0 {
			batches = append(batches, currentBatch)
			currentBatch = batch{texts: []string{}, estimatedTokens: 0}
		}

		// Add text to current batch
		currentBatch.texts = append(currentBatch.texts, text)
		currentBatch.estimatedTokens += estimatedTokens
	}

	// Don't forget the last batch
	if len(currentBatch.texts) > 0 {
		batches = append(batches, currentBatch)
	}

	return batches
}

// callAPI makes the API call to the embedding provider
func (p *HTTPEmbeddingProvider) callAPI(input any) ([][]float32, error) {
	reqBody := embeddingRequest{
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

	var apiResp embeddingResponse
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
