package embeddings

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/karim-daw/qwelli/internal/voyage"
)

// Test request/response types for mock server
type testEmbeddingContentItem struct {
	Type        string `json:"type"`
	Text        string `json:"text,omitempty"`
	ImageBase64 string `json:"image_base64,omitempty"`
}

type testEmbeddingInput struct {
	Content []testEmbeddingContentItem `json:"content"`
}

type testEmbeddingRequest struct {
	Model  string               `json:"model"`
	Inputs []testEmbeddingInput `json:"inputs"`
}

type testEmbeddingResponse struct {
	Object string `json:"object"`
	Data   []struct {
		Object    string    `json:"object"`
		Embedding []float64 `json:"embedding"`
		Index     int       `json:"index"`
	} `json:"data"`
	Model string `json:"model"`
}

func TestNewEmbedder(t *testing.T) {
	tests := []struct {
		name      string
		apiKey    string
		wantError bool
	}{
		{
			name:      "valid config",
			apiKey:    "test-key",
			wantError: false,
		},
		{
			name:      "missing api key",
			apiKey:    "",
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client, err := voyage.NewClient(voyage.ClientConfig{
				APIKey: tt.apiKey,
			})
			if tt.wantError {
				if err == nil {
					t.Errorf("NewClient() expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Fatalf("NewClient() error = %v", err)
			}

			embedder, err := NewEmbedder(client)
			if err != nil {
				t.Fatalf("NewEmbedder() error = %v", err)
			}

			if embedder == nil {
				t.Fatal("NewEmbedder() returned nil")
			}
		})
	}
}

func TestEmbedder_Embed(t *testing.T) {
	// Create mock server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			t.Errorf("Expected POST, got %s", r.Method)
		}

		var req testEmbeddingRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Errorf("Failed to decode request: %v", err)
		}

		// Verify request structure
		if req.Model == "" {
			t.Error("Request missing model")
		}
		if len(req.Inputs) != 1 {
			t.Errorf("Expected 1 input, got %d", len(req.Inputs))
		}

		// Return mock response
		emb := make([]float64, 1024)
		for i := range emb {
			emb[i] = 0.1
		}
		response := testEmbeddingResponse{
			Object: "list",
			Data: []struct {
				Object    string    `json:"object"`
				Embedding []float64 `json:"embedding"`
				Index     int       `json:"index"`
			}{
				{Object: "embedding", Embedding: emb, Index: 0},
			},
			Model: "voyage-multimodal-3",
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	client, err := voyage.NewClient(voyage.ClientConfig{
		APIKey:            "test-key",
		EmbeddingModel:    "voyage-multimodal-3",
		EmbeddingEndpoint: server.URL,
	})
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}

	embedder, err := NewEmbedder(client)
	if err != nil {
		t.Fatalf("NewEmbedder() error = %v", err)
	}

	embedding, err := embedder.Embed("test text")
	if err != nil {
		t.Fatalf("Embed() error = %v", err)
	}

	if len(embedding) != 1024 {
		t.Errorf("Embed() returned embedding with length %d, want 1024", len(embedding))
	}
}

func TestEmbedder_EmbedBatch(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req testEmbeddingRequest
		json.NewDecoder(r.Body).Decode(&req)

		// Create response with same number of embeddings as input objects
		dataItems := make([]struct {
			Object    string    `json:"object"`
			Embedding []float64 `json:"embedding"`
			Index     int       `json:"index"`
		}, len(req.Inputs))
		for i := range dataItems {
			emb := make([]float64, 1024)
			for j := range emb {
				emb[j] = 0.1
			}
			dataItems[i] = struct {
				Object    string    `json:"object"`
				Embedding []float64 `json:"embedding"`
				Index     int       `json:"index"`
			}{Object: "embedding", Embedding: emb, Index: i}
		}

		response := testEmbeddingResponse{
			Object: "list",
			Data:   dataItems,
			Model:  "voyage-multimodal-3",
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	client, err := voyage.NewClient(voyage.ClientConfig{
		APIKey:            "test-key",
		EmbeddingModel:    "voyage-multimodal-3",
		EmbeddingEndpoint: server.URL,
	})
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}

	embedder, err := NewEmbedder(client)
	if err != nil {
		t.Fatalf("NewEmbedder() error = %v", err)
	}

	texts := []string{"text 1", "text 2", "text 3"}
	embeddings, err := embedder.EmbedBatch(context.Background(), texts, nil)
	if err != nil {
		t.Fatalf("EmbedBatch() error = %v", err)
	}

	if len(embeddings) != len(texts) {
		t.Errorf("EmbedBatch() returned %d embeddings, want %d", len(embeddings), len(texts))
	}

	for i, emb := range embeddings {
		if len(emb) != 1024 {
			t.Errorf("EmbedBatch() embedding %d has length %d, want 1024", i, len(emb))
		}
	}
}

func TestEmbedder_ErrorHandling(t *testing.T) {
	// Test API error response
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(`{"error": "Invalid API key"}`))
	}))
	defer server.Close()

	client, err := voyage.NewClient(voyage.ClientConfig{
		APIKey:            "invalid-key",
		EmbeddingModel:    "voyage-multimodal-3",
		EmbeddingEndpoint: server.URL,
	})
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}

	embedder, err := NewEmbedder(client)
	if err != nil {
		t.Fatalf("NewEmbedder() error = %v", err)
	}

	_, err = embedder.Embed("test")
	if err == nil {
		t.Error("Embed() expected error for invalid API key, got nil")
	}
}
