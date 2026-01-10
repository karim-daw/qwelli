package voyage

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/karim-daw/qwelli/internal/textutil"
)

// Batch limits for Voyage API
const (
	maxInputsPerBatch   = 200    // API max is 1000, but we use 200 for reliability
	maxTokensPerBatch   = 200000 // API max is 320000
	maxTokensPerInput   = 32000
	pixelsPerImageToken = 560 // 560 pixels = 1 token for images
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

		log.Printf("  Processing batch %d/%d (%d inputs)%s",
			batchIdx+1, len(batches), len(batch), eta)

		batchStart := time.Now()
		embeddings, err := c.callEmbeddingAPI(ctx, batch)
		if err != nil {
			// Check if cancelled during API call
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			default:
			}
			return nil, fmt.Errorf("failed to process batch %d: %w", batchIdx+1, err)
		}

		log.Printf("  Batch %d completed in %v", batchIdx+1, time.Since(batchStart).Round(time.Millisecond))

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

	log.Printf("Generated %d embeddings in %d API call(s): %v (avg: %v per embedding)",
		len(inputs), totalAPICalls, time.Since(start), time.Since(start)/time.Duration(len(inputs)))
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
