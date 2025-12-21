package chunker

import (
	"strings"

	"github.com/karim-daw/qwelli/internal/textutil"
)

// TextChunkStrategy implements chunking for plain text
type TextChunkStrategy struct{}

func (s *TextChunkStrategy) Chunkify(content interface{}, config ChunkerConfig, baseMetadata map[string]interface{}) ([]Chunk, error) {
	text, ok := content.(string)
	if !ok {
		return nil, &StrategyError{Message: "TextChunkStrategy requires string content"}
	}

	totalTokens := textutil.EstimateTokens(text)

	// If text is smaller than chunk size, return as single chunk
	if totalTokens <= config.ChunkSize {
		metadata := make(map[string]interface{})
		for k, v := range baseMetadata {
			metadata[k] = v
		}
		metadata["chunk_index"] = 0
		metadata["total_chunks"] = 1

		return []Chunk{
			{
				Content:     text,
				ContentType: "text",
				PageNumbers: []int{},
				ChunkIndex:  0,
				TotalChunks: 1,
				StartToken:  0,
				EndToken:    totalTokens,
				Metadata:    metadata,
			},
		}, nil
	}

	// Split text into sentences
	sentences := textutil.SplitIntoSentences(text)

	var chunks []Chunk
	var currentChunk strings.Builder
	currentTokens := 0
	startToken := 0

	for i := 0; i < len(sentences); i++ {
		sentence := sentences[i]
		sentenceTokens := textutil.EstimateTokens(sentence)

		// If adding this sentence would exceed chunk size, finalize current chunk
		if currentTokens > 0 && currentTokens+sentenceTokens > config.ChunkSize {
			// Save current chunk
			chunks = append(chunks, Chunk{
				Content:     strings.TrimSpace(currentChunk.String()),
				ContentType: "text",
				PageNumbers: []int{},
				StartToken:  startToken,
				EndToken:    startToken + currentTokens,
				Metadata:    nil, // Will be set later
			})

			// Start new chunk with overlap
			overlap := s.getOverlapText(sentences, i, config.OverlapSize)
			currentChunk.Reset()
			currentChunk.WriteString(overlap)
			currentTokens = textutil.EstimateTokens(overlap)
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
			Content:     strings.TrimSpace(currentChunk.String()),
			ContentType: "text",
			PageNumbers: []int{},
			StartToken:  startToken,
			EndToken:    startToken + currentTokens,
			Metadata:    nil,
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
		chunks[i].ChunkIndex = i
		chunks[i].TotalChunks = totalChunks
	}

	return chunks, nil
}

// getOverlapText returns the last N tokens worth of sentences
// This is a helper function specific to text chunking
func (s *TextChunkStrategy) getOverlapText(sentences []string, currentIndex int, targetTokens int) string {
	var overlap strings.Builder
	tokens := 0

	// Go backwards from current position to gather overlap
	for i := currentIndex - 1; i >= 0 && tokens < targetTokens; i-- {
		sentence := sentences[i]
		sentenceTokens := textutil.EstimateTokens(sentence)

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
