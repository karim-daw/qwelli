package processor

import (
	"strings"
)

// Chunk represents a piece of text with metadata
type Chunk struct {
	Content    string
	StartToken int
	EndToken   int
	Metadata   map[string]interface{}
}

// ChunkerConfig defines chunking parameters
type ChunkerConfig struct {
	ChunkSize   int // Target tokens per chunk (e.g., 1000)
	OverlapSize int // Overlap tokens between chunks (e.g., 150)
}

// Chunker splits text into overlapping chunks
type Chunker struct {
	config ChunkerConfig
}

// NewChunker creates a new Chunker with the given configuration
func NewChunker(config ChunkerConfig) *Chunker {
	return &Chunker{config: config}
}

// ChunkByTokens splits text into overlapping chunks by token count
// It attempts to respect sentence boundaries when possible
func (c *Chunker) ChunkByTokens(text string, baseMetadata map[string]interface{}) ([]Chunk, error) {
	totalTokens := EstimateTokens(text)

	// If text is smaller than chunk size, return as single chunk
	if totalTokens <= c.config.ChunkSize {
		metadata := make(map[string]interface{})
		for k, v := range baseMetadata {
			metadata[k] = v
		}
		metadata["chunk_index"] = 0
		metadata["total_chunks"] = 1

		return []Chunk{
			{
				Content:    text,
				StartToken: 0,
				EndToken:   totalTokens,
				Metadata:   metadata,
			},
		}, nil
	}

	// Split text into sentences
	sentences := splitIntoSentences(text)

	var chunks []Chunk
	var currentChunk strings.Builder
	currentTokens := 0
	startToken := 0

	for i := 0; i < len(sentences); i++ {
		sentence := sentences[i]
		sentenceTokens := EstimateTokens(sentence)

		// If adding this sentence would exceed chunk size, finalize current chunk
		if currentTokens > 0 && currentTokens+sentenceTokens > c.config.ChunkSize {
			// Save current chunk
			chunks = append(chunks, Chunk{
				Content:    strings.TrimSpace(currentChunk.String()),
				StartToken: startToken,
				EndToken:   startToken + currentTokens,
				Metadata:   nil, // Will be set later
			})

			// Start new chunk with overlap
			overlap := c.getOverlapText(sentences, i, c.config.OverlapSize)
			currentChunk.Reset()
			currentChunk.WriteString(overlap)
			currentTokens = EstimateTokens(overlap)
			startToken = startToken + currentTokens
		}

		// Add sentence to current chunk
		if currentChunk.Len() > 0 {
			currentChunk.WriteString(" ")
		}
		currentChunk.WriteString(sentence)
		currentTokens += sentenceTokens
	}

	// Add final chunk
	if currentChunk.Len() > 0 {
		chunks = append(chunks, Chunk{
			Content:    strings.TrimSpace(currentChunk.String()),
			StartToken: startToken,
			EndToken:   startToken + currentTokens,
			Metadata:   nil,
		})
	}

	// Add metadata to all chunks
	totalChunks := len(chunks)
	for i := range chunks {
		metadata := make(map[string]interface{})
		for k, v := range baseMetadata {
			metadata[k] = v
		}
		metadata["chunk_index"] = i
		metadata["total_chunks"] = totalChunks
		chunks[i].Metadata = metadata
	}

	return chunks, nil
}

// getOverlapText returns the last N tokens worth of sentences
func (c *Chunker) getOverlapText(sentences []string, currentIndex int, targetTokens int) string {
	var overlap strings.Builder
	tokens := 0

	// Go backwards from current position to gather overlap
	for i := currentIndex - 1; i >= 0 && tokens < targetTokens; i-- {
		sentence := sentences[i]
		sentenceTokens := EstimateTokens(sentence)

		if tokens+sentenceTokens > targetTokens {
			break
		}

		// Prepend sentence
		if overlap.Len() > 0 {
			overlap.WriteString(" ")
		}
		overlap.WriteString(sentence)
		tokens += sentenceTokens
	}

	return overlap.String()
}

// splitIntoSentences splits text into sentences
// Simple implementation that splits on common sentence terminators
func splitIntoSentences(text string) []string {
	// Replace newlines with spaces to treat them as sentence boundaries
	text = strings.ReplaceAll(text, "\n", ". ")

	var sentences []string
	var current strings.Builder

	runes := []rune(text)
	for i := 0; i < len(runes); i++ {
		r := runes[i]
		current.WriteRune(r)

		// Check for sentence terminators
		if r == '.' || r == '!' || r == '?' {
			// Look ahead to see if followed by space or end of text
			if i+1 >= len(runes) || runes[i+1] == ' ' {
				sentence := strings.TrimSpace(current.String())
				if sentence != "" {
					sentences = append(sentences, sentence)
				}
				current.Reset()
			}
		}
	}

	// Add any remaining text
	remaining := strings.TrimSpace(current.String())
	if remaining != "" {
		sentences = append(sentences, remaining)
	}

	// If no sentences were found, return the whole text
	if len(sentences) == 0 {
		sentences = append(sentences, strings.TrimSpace(text))
	}

	return sentences
}
