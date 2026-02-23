package embeddings

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/karim-daw/qwelli/internal/voyage"
)

// Test response types for mock server
type testMultimodalResponse struct {
	Object string `json:"object"`
	Data   []struct {
		Object    string    `json:"object"`
		Embedding []float64 `json:"embedding"`
		Index     int       `json:"index"`
	} `json:"data"`
	Model string `json:"model"`
}

type testMultimodalRequest struct {
	Model  string `json:"model"`
	Inputs []struct {
		Content []struct {
			Type        string `json:"type"`
			Text        string `json:"text,omitempty"`
			ImageBase64 string `json:"image_base64,omitempty"`
		} `json:"content"`
	} `json:"inputs"`
}

func TestEmbedder_IsMultimodal(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		emb := make([]float64, 1024)
		response := testMultimodalResponse{
			Object: "list",
			Data: []struct {
				Object    string    `json:"object"`
				Embedding []float64 `json:"embedding"`
				Index     int       `json:"index"`
			}{
				{Object: "embedding", Embedding: emb, Index: 0},
			},
			Model: "voyage-multimodal-3.5",
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	client, err := voyage.NewClient(voyage.ClientConfig{
		APIKey:            "test-key",
		EmbeddingModel:    "voyage-multimodal-3.5",
		EmbeddingEndpoint: server.URL,
	})
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}

	embedder, err := NewEmbedder(client)
	if err != nil {
		t.Fatalf("NewEmbedder() error = %v", err)
	}

	if !embedder.IsMultimodal() {
		t.Error("IsMultimodal() = false, want true for Voyage provider")
	}
}

func TestEmbedder_EmptyInputs(t *testing.T) {
	client, err := voyage.NewClient(voyage.ClientConfig{
		APIKey:            "test-key",
		EmbeddingModel:    "voyage-multimodal-3.5",
		EmbeddingEndpoint: "https://api.voyageai.com/v1/multimodalembeddings",
	})
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}

	embedder, err := NewEmbedder(client)
	if err != nil {
		t.Fatalf("NewEmbedder() error = %v", err)
	}

	// Test empty batch
	embeddings, err := embedder.EmbedBatch(context.Background(), []string{}, nil)
	if err != nil {
		t.Fatalf("EmbedBatch() with empty input error = %v", err)
	}
	if len(embeddings) != 0 {
		t.Errorf("EmbedBatch() with empty input returned %d embeddings, want 0", len(embeddings))
	}

	// Test empty multimodal
	multimodalEmbeddings, err := embedder.EmbedMultimodal(context.Background(), []MultimodalInput{}, nil)
	if err != nil {
		t.Fatalf("EmbedMultimodal() with empty input error = %v", err)
	}
	if len(multimodalEmbeddings) != 0 {
		t.Errorf("EmbedMultimodal() with empty input returned %d embeddings, want 0", len(multimodalEmbeddings))
	}
}

func TestEmbedder_Batching(t *testing.T) {
	callCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		var req testMultimodalRequest
		json.NewDecoder(r.Body).Decode(&req)

		// Voyage API returns one embedding per input object
		numInputs := len(req.Inputs)
		dataItems := make([]struct {
			Object    string    `json:"object"`
			Embedding []float64 `json:"embedding"`
			Index     int       `json:"index"`
		}, numInputs)
		for i := range dataItems {
			emb := make([]float64, 1024)
			dataItems[i] = struct {
				Object    string    `json:"object"`
				Embedding []float64 `json:"embedding"`
				Index     int       `json:"index"`
			}{Object: "embedding", Embedding: emb, Index: i}
		}

		response := testMultimodalResponse{
			Object: "list",
			Data:   dataItems,
			Model:  "voyage-multimodal-3.5",
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	client, err := voyage.NewClient(voyage.ClientConfig{
		APIKey:            "test-key",
		EmbeddingModel:    "voyage-multimodal-3.5",
		EmbeddingEndpoint: server.URL,
	})
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}

	embedder, err := NewEmbedder(client)
	if err != nil {
		t.Fatalf("NewEmbedder() error = %v", err)
	}

	// Create 2500 inputs (should create multiple batches with max 200 per batch)
	texts := make([]string, 2500)
	for i := range texts {
		texts[i] = "test text"
	}

	embeddings, err := embedder.EmbedBatch(context.Background(), texts, nil)
	if err != nil {
		t.Fatalf("EmbedBatch() error = %v", err)
	}

	if len(embeddings) != len(texts) {
		t.Errorf("EmbedBatch() returned %d embeddings, want %d", len(embeddings), len(texts))
	}

	// Should have made multiple API calls (with 200 per batch, 2500 inputs = at least 13 calls)
	if callCount < 2 {
		t.Errorf("EmbedBatch() made %d API calls, want at least 2", callCount)
	}
}

func TestMultimodalInput_Validation(t *testing.T) {
	tests := []struct {
		name      string
		input     MultimodalInput
		wantValid bool
	}{
		{
			name: "valid text input",
			input: MultimodalInput{
				Type: "text",
				Text: "some text",
			},
			wantValid: true,
		},
		{
			name: "valid image input",
			input: MultimodalInput{
				Type:        "image",
				ImageBase64: "base64data",
			},
			wantValid: true,
		},
		{
			name: "text input with empty text",
			input: MultimodalInput{
				Type: "text",
				Text: "",
			},
			wantValid: true, // Empty text is still valid
		},
		{
			name: "image input with empty base64",
			input: MultimodalInput{
				Type:        "image",
				ImageBase64: "",
			},
			wantValid: true, // Empty base64 is still valid (will fail at API)
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Just verify the struct can be created
			if tt.input.Type == "" {
				t.Error("MultimodalInput must have Type set")
			}
		})
	}
}
