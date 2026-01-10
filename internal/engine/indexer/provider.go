package indexer

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"time"

	"github.com/karim-daw/qwelli/internal/textutil"
)

// ============================================================================
// INTERFACES
// ============================================================================

// EmbeddingProvider is the interface that all embedding providers must implement
type EmbeddingProvider interface {
	Embed(text string) ([]float32, error)
	EmbedBatch(ctx context.Context, texts []string, progressCallback func(current, total int)) ([][]float32, error)
}

// MultimodalEmbeddingProvider extends EmbeddingProvider with multimodal support
type MultimodalEmbeddingProvider interface {
	EmbeddingProvider
	EmbedMultimodal(ctx context.Context, inputs []MultimodalInput, progressCallback func(current, total int)) ([][]float32, error)
}

// ============================================================================
// SHARED TYPES
// ============================================================================

// MultimodalInput represents either text or image input for embedding
type MultimodalInput struct {
	Type        string // "text" or "image"
	Text        string // For text inputs
	ImageBase64 string // For image inputs (base64-encoded, without data URI prefix)
	ImageFormat string // For image inputs: "jpeg", "png", "gif", etc. (used to add data URI prefix)
	ImagePixels int    // For image inputs: pixel count (width * height) for token calculation
}

// Voyage API request/response structures
// Based on Voyage AI API: https://docs.voyageai.com/reference/multimodal-embeddings
// Format: inputs is array of objects, each with a content array containing type-specific items
type voyageContentItem struct {
	Type        string `json:"type"` // "text", "image_base64", or "video_base64"
	Text        string `json:"text,omitempty"`
	ImageBase64 string `json:"image_base64,omitempty"`
	VideoBase64 string `json:"video_base64,omitempty"`
}

type voyageInput struct {
	Content []voyageContentItem `json:"content"`
}

type voyageMultimodalRequest struct {
	Model  string        `json:"model"`
	Inputs []voyageInput `json:"inputs"`
}

type voyageMultimodalResponse struct {
	Object string `json:"object"`
	Data   []struct {
		Object    string    `json:"object"`
		Embedding []float64 `json:"embedding"`
		Index     int       `json:"index"`
	} `json:"data"`
	Model string `json:"model"`
	Usage struct {
		TextTokens  int `json:"text_tokens"`
		ImagePixels int `json:"image_pixels"`
		VideoPixels int `json:"video_pixels"`
		TotalTokens int `json:"total_tokens"`
	} `json:"usage"`
}

// ============================================================================
// EMBEDDER (Factory/Dispatcher)
// ============================================================================

// Embedder wraps an EmbeddingProvider and provides a unified interface
type Embedder struct {
	provider EmbeddingProvider
}

// NewEmbedder creates an embedder with the appropriate provider
// Currently only supports Voyage AI, but designed to support multiple providers
func NewEmbedder(apiKey, model, endpoint string) (*Embedder, error) {
	// Currently hardcoded to Voyage - future: add provider selection based on model/endpoint
	provider, err := NewVoyageEmbeddingProvider(apiKey, model, endpoint)
	if err != nil {
		return nil, err
	}
	return &Embedder{provider: provider}, nil
}

// Embed generates an embedding for a single text
func (e *Embedder) Embed(text string) ([]float32, error) {
	return e.provider.Embed(text)
}

// EmbedBatch generates embeddings for multiple texts
// progressCallback is called with (current, total) as batches are processed
func (e *Embedder) EmbedBatch(ctx context.Context, texts []string, progressCallback func(current, total int)) ([][]float32, error) {
	return e.provider.EmbedBatch(ctx, texts, progressCallback)
}

// IsMultimodal returns true if the provider supports multimodal embeddings
func (e *Embedder) IsMultimodal() bool {
	_, ok := e.provider.(MultimodalEmbeddingProvider)
	return ok
}

// EmbedMultimodal generates embeddings for mixed text and image inputs
// Returns an error if the provider doesn't support multimodal embeddings
// progressCallback is called with (current, total) as batches are processed
func (e *Embedder) EmbedMultimodal(ctx context.Context, inputs []MultimodalInput, progressCallback func(current, total int)) ([][]float32, error) {
	multimodalProvider, ok := e.provider.(MultimodalEmbeddingProvider)
	if !ok {
		return nil, fmt.Errorf("provider does not support multimodal embeddings")
	}
	return multimodalProvider.EmbedMultimodal(ctx, inputs, progressCallback)
}

// ============================================================================
// VOYAGE AI PROVIDER
// ============================================================================

// VoyageEmbeddingProvider implements MultimodalEmbeddingProvider for Voyage AI
type VoyageEmbeddingProvider struct {
	apiKey   string
	model    string
	endpoint string
	client   *http.Client
}

// NewVoyageEmbeddingProvider creates a new Voyage AI embedding provider
func NewVoyageEmbeddingProvider(apiKey, model, endpoint string) (*VoyageEmbeddingProvider, error) {
	if apiKey == "" {
		return nil, fmt.Errorf("API key required")
	}
	if model == "" {
		model = "voyage-multimodal-3" // Default model
	}
	if endpoint == "" {
		endpoint = "https://api.voyageai.com/v1/multimodalembeddings"
	}

	p := &VoyageEmbeddingProvider{
		apiKey:   apiKey,
		model:    model,
		endpoint: endpoint,
		client: &http.Client{
			Timeout: 180 * time.Second, // Increased to 3 minutes for large batches
		},
	}

	log.Printf("✅ Voyage embedding provider initialized (model: %s)", model)
	return p, nil
}

// Embed generates an embedding for a single text
func (p *VoyageEmbeddingProvider) Embed(text string) ([]float32, error) {
	embeddings, err := p.EmbedBatch(context.Background(), []string{text}, nil)
	if err != nil {
		return nil, err
	}
	if len(embeddings) != 1 {
		return nil, fmt.Errorf("expected 1 embedding, got %d", len(embeddings))
	}
	return embeddings[0], nil
}

// EmbedBatch generates embeddings for multiple texts
func (p *VoyageEmbeddingProvider) EmbedBatch(ctx context.Context, texts []string, progressCallback func(current, total int)) ([][]float32, error) {
	if len(texts) == 0 {
		return [][]float32{}, nil
	}

	// Convert texts to multimodal inputs
	inputs := make([]MultimodalInput, len(texts))
	for i, text := range texts {
		inputs[i] = MultimodalInput{
			Type: "text",
			Text: text,
		}
	}

	return p.EmbedMultimodal(ctx, inputs, progressCallback)
}

// EmbedMultimodal generates embeddings for mixed text and image inputs
func (p *VoyageEmbeddingProvider) EmbedMultimodal(ctx context.Context, inputs []MultimodalInput, progressCallback func(current, total int)) ([][]float32, error) {
	start := time.Now()

	if len(inputs) == 0 {
		return [][]float32{}, nil
	}

	// Create batches that respect Voyage API limits
	batches := p.createMultimodalBatches(inputs)
	log.Printf("  Split %d inputs into %d batch(es)", len(inputs), len(batches))

	allEmbeddings := make([][]float32, 0, len(inputs))
	totalAPICalls := 0
	batchStartTime := time.Now()
	totalInputs := len(inputs)
	processedInputs := 0

	for batchIdx, batch := range batches {
		// Check for cancellation before processing each batch
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}

		// Calculate estimated time remaining
		var eta string
		if batchIdx > 0 {
			avgTimePerBatch := time.Since(batchStartTime) / time.Duration(batchIdx)
			remainingBatches := len(batches) - batchIdx
			estimatedRemaining := avgTimePerBatch * time.Duration(remainingBatches)
			eta = fmt.Sprintf(" (ETA: %v)", estimatedRemaining.Round(time.Second))
		}

		log.Printf("  📦 Processing batch %d/%d (%d inputs)%s",
			batchIdx+1, len(batches), len(batch), eta)

		batchStart := time.Now()
		embeddings, err := p.callMultimodalAPI(batch)
		if err != nil {
			// Check if cancelled during API call
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			default:
			}
			return nil, fmt.Errorf("failed to process batch %d: %w", batchIdx+1, err)
		}

		log.Printf("  ✓ Batch %d completed in %v", batchIdx+1, time.Since(batchStart).Round(time.Millisecond))

		if len(embeddings) != len(batch) {
			return nil, fmt.Errorf("batch %d: expected %d embeddings, got %d",
				batchIdx+1, len(batch), len(embeddings))
		}

		allEmbeddings = append(allEmbeddings, embeddings...)
		totalAPICalls++
		processedInputs += len(batch)

		// Report progress
		if progressCallback != nil {
			progressCallback(processedInputs, totalInputs)
		}
	}

	log.Printf("⏱️  Generated %d embeddings in %d API call(s): %v (avg: %v per embedding)",
		len(inputs), totalAPICalls, time.Since(start), time.Since(start)/time.Duration(len(inputs)))
	return allEmbeddings, nil
}

// createMultimodalBatches groups inputs into batches respecting Voyage API limits
// Constraints:
// - Max 1,000 inputs per request
// - Max 320,000 total tokens per request
// - Max 32,000 tokens per input
// - Image tokens: 560 pixels = 1 token
func (p *VoyageEmbeddingProvider) createMultimodalBatches(inputs []MultimodalInput) [][]MultimodalInput {
	const maxInputsPerBatch = 200    // Increased from 50 to improve throughput (API max is 1000)
	const maxTokensPerBatch = 200000 // Increased from 50000 (API max is 320000)
	const maxTokensPerInput = 32000
	const pixelsPerImageToken = 560 // 560 pixels = 1 token for images

	batches := [][]MultimodalInput{}
	currentBatch := []MultimodalInput{}
	currentTokens := 0

	for _, input := range inputs {
		var inputTokens int
		switch input.Type {
		case "text":
			inputTokens = textutil.EstimateTokens(input.Text)
		case "image":
			if input.ImagePixels > 0 {
				inputTokens = (input.ImagePixels + pixelsPerImageToken - 1) / pixelsPerImageToken
			} else {
				inputTokens = 1000
			}
		default:
			inputTokens = 1000
		}

		// Check per-input token limit (32,000 tokens per input)
		if inputTokens > maxTokensPerInput {
			log.Printf("  ⚠️  Skipping input: exceeds per-input token limit (%d tokens, max %d tokens)", inputTokens, maxTokensPerInput)
			continue
		}

		// If adding this input would exceed batch limits, start a new batch
		if len(currentBatch) >= maxInputsPerBatch ||
			(currentTokens+inputTokens > maxTokensPerBatch && len(currentBatch) > 0) {
			batches = append(batches, currentBatch)
			currentBatch = []MultimodalInput{}
			currentTokens = 0
		}

		currentBatch = append(currentBatch, input)
		currentTokens += inputTokens
	}

	// Don't forget the last batch
	if len(currentBatch) > 0 {
		batches = append(batches, currentBatch)
	}

	// Log batch composition for debugging
	if len(batches) > 0 {
		log.Printf("  Created %d batch(es) with token counts:", len(batches))
		for i, batch := range batches {
			batchTokens := 0
			textCount := 0
			imageCount := 0
			for _, inp := range batch {
				switch inp.Type {
				case "text":
					batchTokens += textutil.EstimateTokens(inp.Text)
					textCount++
				case "image":
					if inp.ImagePixels > 0 {
						batchTokens += (inp.ImagePixels + pixelsPerImageToken - 1) / pixelsPerImageToken
					} else {
						batchTokens += 1000
					}
					imageCount++
				}
			}
			log.Printf("    Batch %d: %d inputs (%d text, %d images), ~%d tokens",
				i+1, len(batch), textCount, imageCount, batchTokens)
		}
	}

	return batches
}

// callMultimodalAPI makes the API call to Voyage AI
func (p *VoyageEmbeddingProvider) callMultimodalAPI(inputs []MultimodalInput) ([][]float32, error) {
	// Convert inputs to Voyage API format
	// Each input becomes a separate input object with one content item
	// The API returns one embedding per input object
	apiInputs := make([]voyageInput, len(inputs))
	for i, input := range inputs {
		var contentItem voyageContentItem
		switch input.Type {
		case "text":
			contentItem = voyageContentItem{
				Type: "text",
				Text: input.Text,
			}
		case "image":
			// Validate base64 string before sending
			if input.ImageBase64 == "" {
				// Create empty content item - this will likely fail but at least we log it
				contentItem = voyageContentItem{
					Type:        "image_base64",
					ImageBase64: "",
				}
			} else {
				// Verify base64 is valid by checking characters
				validChars := true
				for _, c := range input.ImageBase64 {
					if !((c >= 'A' && c <= 'Z') || (c >= 'a' && c <= 'z') || (c >= '0' && c <= '9') || c == '+' || c == '/' || c == '=' || c == '\n' || c == '\r' || c == ' ') {
						validChars = false
						break
					}
				}
				if !validChars {
					contentItem = voyageContentItem{
						Type:        "image_base64",
						ImageBase64: "",
					}
				} else {
					// Add data URI prefix based on image format (Voyage API requires this)
					// Format must be: data:[<mediatype>];base64,<data>
					// Map format to MIME type
					mimeType := "image/jpeg" // default fallback
					if input.ImageFormat != "" {
						switch input.ImageFormat {
						case "png":
							mimeType = "image/png"
						case "jpeg", "jpg":
							mimeType = "image/jpeg"
						case "gif":
							mimeType = "image/gif"
						case "webp":
							mimeType = "image/webp"
						}
					}
					// Always add data URI prefix (required by Voyage API)
					imageBase64WithPrefix := fmt.Sprintf("data:%s;base64,%s", mimeType, input.ImageBase64)
					contentItem = voyageContentItem{
						Type:        "image_base64",
						ImageBase64: imageBase64WithPrefix,
					}
				}
			}
		default:
			contentItem = voyageContentItem{} // Or handle unknown type appropriately
		}
		apiInputs[i] = voyageInput{
			Content: []voyageContentItem{contentItem},
		}
	}

	reqBody := voyageMultimodalRequest{
		Model:  p.model,
		Inputs: apiInputs,
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

	// Retry logic with exponential backoff for timeout errors
	var resp *http.Response
	maxRetries := 3
	for attempt := 0; attempt <= maxRetries; attempt++ {
		if attempt > 0 {
			waitTime := time.Duration(attempt*attempt) * time.Second // 1s, 4s, 9s
			log.Printf("  ⏳ Retry attempt %d/%d after %v (previous attempt timed out)", attempt, maxRetries, waitTime)
			time.Sleep(waitTime)

			// Recreate request body for retry
			req.Body = io.NopCloser(bytes.NewBuffer(jsonData))
		}

		resp, err = p.client.Do(req)
		if err != nil {
			// Check if it's a timeout error
			if isTimeoutError(err) && attempt < maxRetries {
				log.Printf("  ⚠️  Request timed out, will retry...")
				continue
			}
			return nil, fmt.Errorf("API request failed: %w", err)
		}
		// Success, break out of retry loop
		break
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		// Log which inputs were sent for debugging
		imageInputs := 0
		textInputs := 0
		for _, inp := range inputs {
			if inp.Type == "image" {
				imageInputs++
			} else {
				textInputs++
			}
		}
		log.Printf("  API error: sent %d text inputs and %d image inputs", textInputs, imageInputs)
		return nil, fmt.Errorf("API error (status %d): %s", resp.StatusCode, string(body))
	}

	var apiResp voyageMultimodalResponse
	if err := json.NewDecoder(resp.Body).Decode(&apiResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	// Convert response data to [][]float32
	// The API returns embeddings in the data array, one per content item
	embeddings := make([][]float32, len(apiResp.Data))
	for i, item := range apiResp.Data {
		embedding := make([]float32, len(item.Embedding))
		for j, v := range item.Embedding {
			embedding[j] = float32(v)
		}
		embeddings[i] = embedding
	}

	return embeddings, nil
}

// ============================================================================
// HELPER FUNCTIONS
// ============================================================================

// isTimeoutError checks if an error is a timeout error
func isTimeoutError(err error) bool {
	if err == nil {
		return false
	}
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
