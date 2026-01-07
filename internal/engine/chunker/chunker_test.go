package chunker

import (
	"strings"
	"testing"

	"github.com/karim-daw/qwelli/internal/textutil"
)

func TestChunkService_ChunkText_SmallDocument(t *testing.T) {
	service := NewChunkService(ChunkerConfig{
		ChunkSize:   1000,
		OverlapSize: 150,
	})

	// Small text that should fit in one chunk
	text := "This is a small document. It has only a few sentences. It should not be split."

	chunks, err := service.ChunkText(text)
	if err != nil {
		t.Fatalf("ChunkText failed: %v", err)
	}

	if len(chunks) != 1 {
		t.Errorf("Expected 1 chunk, got %d", len(chunks))
	}

	if chunks[0].Content != text {
		t.Errorf("Chunk content doesn't match original text")
	}

	if chunks[0].ChunkIndex != 0 {
		t.Errorf("Expected ChunkIndex 0, got %d", chunks[0].ChunkIndex)
	}

	if chunks[0].TotalChunks != 1 {
		t.Errorf("Expected TotalChunks 1, got %d", chunks[0].TotalChunks)
	}
}

func TestChunkService_ChunkText_LargeDocument(t *testing.T) {
	service := NewChunkService(ChunkerConfig{
		ChunkSize:   100, // Small size to force chunking
		OverlapSize: 20,
	})

	// Generate a longer text with multiple sentences
	sentences := []string{
		"This is the first sentence of our test document.",
		"It contains multiple sentences to test chunking.",
		"Each sentence is designed to be a reasonable length.",
		"We want to ensure that chunking works correctly.",
		"The chunker should split this into multiple chunks.",
		"Each chunk should have overlap with the previous one.",
		"This helps maintain context across chunk boundaries.",
		"The metadata should be correctly set for each chunk.",
	}
	text := strings.Join(sentences, " ")

	chunks, err := service.ChunkText(text)
	if err != nil {
		t.Fatalf("ChunkText failed: %v", err)
	}

	if len(chunks) < 2 {
		t.Errorf("Expected multiple chunks, got %d", len(chunks))
	}

	// Verify chunk indices
	for i, chunk := range chunks {
		if chunk.ChunkIndex != i {
			t.Errorf("Chunk %d has incorrect ChunkIndex: %d", i, chunk.ChunkIndex)
		}

		if chunk.TotalChunks != len(chunks) {
			t.Errorf("Chunk %d has incorrect TotalChunks: %d", i, chunk.TotalChunks)
		}
	}

	// Verify no chunk exceeds the size limit (with some tolerance)
	for i, chunk := range chunks {
		tokens := textutil.EstimateTokens(chunk.Content)
		// Allow some overage since we don't split mid-sentence
		if tokens > 200 {
			t.Errorf("Chunk %d has %d tokens, exceeds limit too much", i, tokens)
		}
	}
}

func TestChunkService_ChunkText_EmptyText(t *testing.T) {
	service := NewChunkService(ChunkerConfig{
		ChunkSize:   1000,
		OverlapSize: 150,
	})

	chunks, err := service.ChunkText("")
	if err != nil {
		t.Fatalf("ChunkText failed on empty text: %v", err)
	}

	if len(chunks) != 1 {
		t.Errorf("Expected 1 chunk for empty text, got %d", len(chunks))
	}

	if chunks[0].Content != "" {
		t.Errorf("Expected empty content, got: %s", chunks[0].Content)
	}
}

func TestSplitIntoSentences(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected int // expected number of sentences
	}{
		{
			name:     "Simple sentences",
			input:    "First sentence. Second sentence. Third sentence.",
			expected: 3,
		},
		{
			name:     "Mixed terminators",
			input:    "Is this a question? Yes it is! This is a statement.",
			expected: 3,
		},
		{
			name:     "Newlines",
			input:    "First line.\nSecond line.\nThird line.",
			expected: 3,
		},
		{
			name:     "No terminators",
			input:    "This is just one long sentence without any punctuation",
			expected: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sentences := textutil.SplitIntoSentences(tt.input)
			if len(sentences) != tt.expected {
				t.Errorf("Expected %d sentences, got %d: %v", tt.expected, len(sentences), sentences)
			}
		})
	}
}
