package chunker

import (
	"strings"
	"testing"

	"github.com/karim-daw/qwelli/internal/textutil"
)

func TestChunkByTokens_SmallDocument(t *testing.T) {
	chunker := NewChunker(ChunkerConfig{
		ChunkSize:   1000,
		OverlapSize: 150,
	})

	// Small text that should fit in one chunk
	text := "This is a small document. It has only a few sentences. It should not be split."
	baseMetadata := map[string]interface{}{
		"file_name": "test.txt",
	}

	chunks, err := chunker.strategy.Chunkify(text, chunker.config, baseMetadata)
	if err != nil {
		t.Fatalf("ChunkByTokens failed: %v", err)
	}

	if len(chunks) != 1 {
		t.Errorf("Expected 1 chunk, got %d", len(chunks))
	}

	if chunks[0].Content != text {
		t.Errorf("Chunk content doesn't match original text")
	}

	if chunks[0].Metadata["chunk_index"] != 0 {
		t.Errorf("Expected chunk_index 0, got %v", chunks[0].Metadata["chunk_index"])
	}

	if chunks[0].Metadata["total_chunks"] != 1 {
		t.Errorf("Expected total_chunks 1, got %v", chunks[0].Metadata["total_chunks"])
	}

	if chunks[0].Metadata["file_name"] != "test.txt" {
		t.Errorf("Expected file_name to be preserved in metadata")
	}
}

func TestChunkByTokens_LargeDocument(t *testing.T) {
	chunker := NewChunker(ChunkerConfig{
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

	baseMetadata := map[string]interface{}{
		"file_name": "large.txt",
	}

	chunks, err := chunker.ChunkByTokens(text, baseMetadata)
	if err != nil {
		t.Fatalf("ChunkByTokens failed: %v", err)
	}

	if len(chunks) < 2 {
		t.Errorf("Expected multiple chunks, got %d", len(chunks))
	}

	// Verify chunk indices
	for i, chunk := range chunks {
		if chunk.Metadata["chunk_index"] != i {
			t.Errorf("Chunk %d has incorrect chunk_index: %v", i, chunk.Metadata["chunk_index"])
		}

		if chunk.Metadata["total_chunks"] != len(chunks) {
			t.Errorf("Chunk %d has incorrect total_chunks: %v", i, chunk.Metadata["total_chunks"])
		}

		if chunk.Metadata["file_name"] != "large.txt" {
			t.Errorf("Chunk %d has incorrect file_name", i)
		}
	}

	// Verify no chunk exceeds the size limit (with some tolerance)
	for i, chunk := range chunks {
		tokens := textutil.EstimateTokens(chunk.Content)
		// Allow some overage since we don't split mid-sentence
		if tokens > chunker.config.ChunkSize*2 {
			t.Errorf("Chunk %d has %d tokens, exceeds limit too much", i, tokens)
		}
	}
}

func TestChunkByTokens_MetadataPreservation(t *testing.T) {
	chunker := NewChunker(ChunkerConfig{
		ChunkSize:   50,
		OverlapSize: 10,
	})

	text := "First sentence. Second sentence. Third sentence. Fourth sentence. Fifth sentence. Sixth sentence."
	baseMetadata := map[string]interface{}{
		"file_name":  "test.txt",
		"file_path":  "/path/to/test.txt",
		"custom_key": "custom_value",
	}

	chunks, err := chunker.ChunkByTokens(text, baseMetadata)
	if err != nil {
		t.Fatalf("ChunkByTokens failed: %v", err)
	}

	// Verify all base metadata is preserved in all chunks
	for i, chunk := range chunks {
		if chunk.Metadata["file_name"] != "test.txt" {
			t.Errorf("Chunk %d missing file_name", i)
		}
		if chunk.Metadata["file_path"] != "/path/to/test.txt" {
			t.Errorf("Chunk %d missing file_path", i)
		}
		if chunk.Metadata["custom_key"] != "custom_value" {
			t.Errorf("Chunk %d missing custom_key", i)
		}
	}
}

func TestChunkByTokens_EmptyText(t *testing.T) {
	chunker := NewChunker(ChunkerConfig{
		ChunkSize:   1000,
		OverlapSize: 150,
	})

	chunks, err := chunker.ChunkByTokens("", map[string]interface{}{})
	if err != nil {
		t.Fatalf("ChunkByTokens failed on empty text: %v", err)
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

func TestNewChunker(t *testing.T) {
	config := ChunkerConfig{
		ChunkSize:   1000,
		OverlapSize: 150,
	}

	chunker := NewChunker(config)

	if chunker == nil {
		t.Fatal("NewChunker returned nil")
	}

	if chunker.config.ChunkSize != 1000 {
		t.Errorf("Expected ChunkSize 1000, got %d", chunker.config.ChunkSize)
	}

	if chunker.config.OverlapSize != 150 {
		t.Errorf("Expected OverlapSize 150, got %d", chunker.config.OverlapSize)
	}
}
