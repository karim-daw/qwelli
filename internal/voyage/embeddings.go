package voyage

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/karim-daw/qwelli/internal/textutil"
)

// Batch limits for Voyage API
const (
	maxInputsPerBatch     = 800    // API max is 1000, using 800 for better throughput
	maxTokensPerBatch     = 250000 // API max is 320000, ~78% for safe headroom with imprecise token estimation
	maxTokensPerInput     = 32000
	pixelsPerImageToken   = 560 // 560 pixels = 1 token for images
	maxConcurrentBatches  = 3   // Number of parallel API calls
)

// Embed generates an embedding for a single text
func (c *Client) Embed(text string) ([]float32, error) {
	embeddings, err := c.EmbedBatch(context.Background(), []string{text}, nil)
	if err != nil {
		return nil, err
	}
	if len(embeddings) != 1 {
		return nil, fmt.Errorf("expected 1 embedding, got %d", len(embeddings))
	}
	return embeddings[0], nil
}

// EmbedBatch generates embeddings for multiple texts
func (c *Client) EmbedBatch(ctx context.Context, texts []string, progressCallback func(current, total int)) ([][]float32, error) {
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

	return c.EmbedMultimodal(ctx, inputs, progressCallback)
}

// EmbedMultimodal generates embeddings for mixed text and image inputs
func (c *Client) EmbedMultimodal(ctx context.Context, inputs []MultimodalInput, progressCallback func(current, total int)) ([][]float32, error) {
	start := time.Now()

	if len(inputs) == 0 {
		return [][]float32{}, nil
	}

	// Create batches that respect Voyage API limits
	batches := c.createMultimodalBatches(inputs)
	log.Printf("  Split %d inputs into %d batch(es), processing up to %d concurrently", len(inputs), len(batches), maxConcurrentBatches)

	// For small number of batches, use sequential processing
	if len(batches) <= 2 {
		return c.embedBatchesSequential(ctx, batches, inputs, progressCallback, start)
	}

	// Use concurrent processing for many batches
	return c.embedBatchesConcurrent(ctx, batches, inputs, progressCallback, start)
}

// embedBatchesSequential processes batches one at a time (for small batch counts)
func (c *Client) embedBatchesSequential(ctx context.Context, batches [][]MultimodalInput, inputs []MultimodalInput, progressCallback func(current, total int), start time.Time) ([][]float32, error) {
	allEmbeddings := make([][]float32, 0, len(inputs))
	totalInputs := len(inputs)
	processedInputs := 0
	batchStartTime := time.Now()

	for batchIdx, batch := range batches {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}

		var eta string
		if batchIdx > 0 {
			avgTimePerBatch := time.Since(batchStartTime) / time.Duration(batchIdx)
			remainingBatches := len(batches) - batchIdx
			estimatedRemaining := avgTimePerBatch * time.Duration(remainingBatches)
			eta = fmt.Sprintf(" (ETA: %v)", estimatedRemaining.Round(time.Second))
		}

		log.Printf("  Processing batch %d/%d (%d inputs)%s", batchIdx+1, len(batches), len(batch), eta)

		batchStart := time.Now()
		embeddings, err := c.callEmbeddingAPI(ctx, batch)
		if err != nil {
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			default:
			}
			return nil, fmt.Errorf("failed to process batch %d: %w", batchIdx+1, err)
		}

		log.Printf("  Batch %d completed in %v", batchIdx+1, time.Since(batchStart).Round(time.Millisecond))

		if len(embeddings) != len(batch) {
			return nil, fmt.Errorf("batch %d: expected %d embeddings, got %d", batchIdx+1, len(batch), len(embeddings))
		}

		allEmbeddings = append(allEmbeddings, embeddings...)
		processedInputs += len(batch)

		if progressCallback != nil {
			progressCallback(processedInputs, totalInputs)
		}
	}

	log.Printf("Generated %d embeddings in %d API call(s): %v (avg: %v per embedding)",
		len(inputs), len(batches), time.Since(start), time.Since(start)/time.Duration(len(inputs)))
	return allEmbeddings, nil
}

// batchResult holds the result of processing a single batch
type batchResult struct {
	index      int
	embeddings [][]float32
	inputCount int
	err        error
}

// embedBatchesConcurrent processes batches concurrently with a worker pool
func (c *Client) embedBatchesConcurrent(ctx context.Context, batches [][]MultimodalInput, inputs []MultimodalInput, progressCallback func(current, total int), start time.Time) ([][]float32, error) {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	numBatches := len(batches)
	totalInputs := len(inputs)

	// Results channel and slice for ordered reassembly
	results := make([]batchResult, numBatches)
	resultChan := make(chan batchResult, numBatches)

	// Semaphore for limiting concurrent API calls
	sem := make(chan struct{}, maxConcurrentBatches)

	// WaitGroup for tracking completion
	var wg sync.WaitGroup

	// Progress tracking
	var processedInputs int
	var progressMu sync.Mutex

	log.Printf("  Starting concurrent batch processing with %d workers", maxConcurrentBatches)

	// Launch goroutines for each batch
	for batchIdx, batch := range batches {
		wg.Add(1)
		go func(idx int, b []MultimodalInput) {
			defer wg.Done()

			// Acquire semaphore
			select {
			case <-ctx.Done():
				resultChan <- batchResult{index: idx, err: ctx.Err()}
				return
			case sem <- struct{}{}:
			}
			defer func() { <-sem }()

			batchStart := time.Now()
			embeddings, err := c.callEmbeddingAPI(ctx, b)

			if err != nil {
				select {
				case <-ctx.Done():
					resultChan <- batchResult{index: idx, err: ctx.Err()}
				default:
					resultChan <- batchResult{index: idx, err: fmt.Errorf("batch %d: %w", idx+1, err)}
				}
				return
			}

			if len(embeddings) != len(b) {
				resultChan <- batchResult{
					index: idx,
					err:   fmt.Errorf("batch %d: expected %d embeddings, got %d", idx+1, len(b), len(embeddings)),
				}
				return
			}

			log.Printf("  Batch %d/%d completed in %v (%d inputs)", idx+1, numBatches, time.Since(batchStart).Round(time.Millisecond), len(b))

			// Update progress
			progressMu.Lock()
			processedInputs += len(b)
			currentProgress := processedInputs
			progressMu.Unlock()

			if progressCallback != nil {
				progressCallback(currentProgress, totalInputs)
			}

			resultChan <- batchResult{index: idx, embeddings: embeddings, inputCount: len(b)}
		}(batchIdx, batch)
	}

	// Wait for all goroutines and close channel
	go func() {
		wg.Wait()
		close(resultChan)
	}()

	// Collect results
	for result := range resultChan {
		if result.err != nil {
			cancel() // Cancel in-flight goroutines before returning
			return nil, result.err
		}
		results[result.index] = result
	}

	// Reassemble in order
	allEmbeddings := make([][]float32, 0, len(inputs))
	for _, result := range results {
		allEmbeddings = append(allEmbeddings, result.embeddings...)
	}

	log.Printf("Generated %d embeddings in %d concurrent API call(s): %v (avg: %v per embedding)",
		len(inputs), numBatches, time.Since(start), time.Since(start)/time.Duration(len(inputs)))
	return allEmbeddings, nil
}

// createMultimodalBatches groups inputs into batches respecting Voyage API limits
func (c *Client) createMultimodalBatches(inputs []MultimodalInput) [][]MultimodalInput {
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
			log.Printf("  Skipping input: exceeds per-input token limit (%d tokens, max %d tokens)", inputTokens, maxTokensPerInput)
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

// callEmbeddingAPI makes the API call to Voyage AI embedding endpoint
func (c *Client) callEmbeddingAPI(ctx context.Context, inputs []MultimodalInput) ([][]float32, error) {
	// Convert inputs to Voyage API format
	apiInputs := make([]embeddingInput, len(inputs))
	for i, input := range inputs {
		var contentItem embeddingContentItem
		switch input.Type {
		case "text":
			contentItem = embeddingContentItem{
				Type: "text",
				Text: input.Text,
			}
		case "image":
			// Validate base64 string before sending
			if input.ImageBase64 == "" {
				contentItem = embeddingContentItem{
					Type:        "image_base64",
					ImageBase64: "",
				}
			} else {
				// Verify base64 is valid by checking characters
				validChars := true
				for _, char := range input.ImageBase64 {
					if !((char >= 'A' && char <= 'Z') || (char >= 'a' && char <= 'z') || (char >= '0' && char <= '9') || char == '+' || char == '/' || char == '=' || char == '\n' || char == '\r' || char == ' ') {
						validChars = false
						break
					}
				}
				if !validChars {
					contentItem = embeddingContentItem{
						Type:        "image_base64",
						ImageBase64: "",
					}
				} else {
					// Add data URI prefix based on image format (Voyage API requires this)
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
					imageBase64WithPrefix := fmt.Sprintf("data:%s;base64,%s", mimeType, input.ImageBase64)
					contentItem = embeddingContentItem{
						Type:        "image_base64",
						ImageBase64: imageBase64WithPrefix,
					}
				}
			}
		default:
			contentItem = embeddingContentItem{}
		}
		apiInputs[i] = embeddingInput{
			Content: []embeddingContentItem{contentItem},
		}
	}

	reqBody := embeddingRequest{
		Model:  c.embeddingModel,
		Inputs: apiInputs,
	}

	respBody, err := c.doRequest(ctx, c.embeddingClient, c.embeddingEndpoint, reqBody)
	if err != nil {
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
		return nil, err
	}

	var apiResp embeddingResponse
	if err := json.Unmarshal(respBody, &apiResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	// Convert response data to [][]float32
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
