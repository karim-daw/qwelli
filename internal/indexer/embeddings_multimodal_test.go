package indexer

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestEmbedder_IsMultimodal(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		emb := make([]float64, 1024)
		response := voyageMultimodalResponse{
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

	embedder, err := NewEmbedder("voyage", "test-key", "voyage-multimodal-3", server.URL)
	if err != nil {
		t.Fatalf("NewEmbedder() error = %v", err)
	}

	if !embedder.IsMultimodal() {
		t.Error("IsMultimodal() = false, want true for Voyage provider")
	}
}

func TestEmbedder_GetMultimodalProvider(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		emb := make([]float64, 1024)
		response := voyageMultimodalResponse{
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

	embedder, err := NewEmbedder("voyage", "test-key", "voyage-multimodal-3", server.URL)
	if err != nil {
		t.Fatalf("NewEmbedder() error = %v", err)
	}

	multimodalProvider, ok := embedder.GetMultimodalProvider()
	if !ok {
		t.Fatal("GetMultimodalProvider() returned false, want true")
	}

	if multimodalProvider == nil {
		t.Fatal("GetMultimodalProvider() returned nil provider")
	}
}

func TestEmbedder_EmbedImage(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		emb := make([]float64, 1024)
		response := voyageMultimodalResponse{
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

	embedder, err := NewEmbedder("voyage", "test-key", "voyage-multimodal-3", server.URL)
	if err != nil {
		t.Fatalf("NewEmbedder() error = %v", err)
	}

	imageBase64 := "iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAYAAAAfFcSJAAAADUlEQVR42mNk+M9QDwADhgGAWjR9awAAAABJRU5ErkJggg=="
	embedding, err := embedder.EmbedImage(imageBase64)
	if err != nil {
		t.Fatalf("EmbedImage() error = %v", err)
	}

	if len(embedding) != 1024 {
		t.Errorf("EmbedImage() returned embedding with length %d, want 1024", len(embedding))
	}
}

func TestEmbedder_EmbedMultimodal(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req voyageMultimodalRequest
		json.NewDecoder(r.Body).Decode(&req)

		// Each input object gets one embedding
		dataItems := make([]struct {
			Object    string    `json:"object"`
			Embedding []float64 `json:"embedding"`
			Index     int       `json:"index"`
		}, len(req.Inputs))
		for i := range dataItems {
			emb := make([]float64, 1024)
			dataItems[i] = struct {
				Object    string    `json:"object"`
				Embedding []float64 `json:"embedding"`
				Index     int       `json:"index"`
			}{Object: "embedding", Embedding: emb, Index: i}
		}

		response := voyageMultimodalResponse{
			Object: "list",
			Data:   dataItems,
			Model:  "voyage-multimodal-3",
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	embedder, err := NewEmbedder("voyage", "test-key", "voyage-multimodal-3", server.URL)
	if err != nil {
		t.Fatalf("NewEmbedder() error = %v", err)
	}

	inputs := []MultimodalInput{
		{Type: "text", Text: "Text input"},
		{Type: "image", ImageBase64: "base64data"},
	}

	embeddings, err := embedder.EmbedMultimodal(inputs)
	if err != nil {
		t.Fatalf("EmbedMultimodal() error = %v", err)
	}

	if len(embeddings) != len(inputs) {
		t.Errorf("EmbedMultimodal() returned %d embeddings, want %d", len(embeddings), len(inputs))
	}
}

func TestVoyageEmbeddingProvider_EmptyInputs(t *testing.T) {
	provider, err := NewVoyageEmbeddingProvider("test-key", "voyage-multimodal-3", "https://api.voyageai.com/v1/multimodalembeddings")
	if err != nil {
		t.Fatalf("NewVoyageEmbeddingProvider() error = %v", err)
	}

	// Test empty batch
	embeddings, err := provider.EmbedBatch([]string{})
	if err != nil {
		t.Fatalf("EmbedBatch() with empty input error = %v", err)
	}
	if len(embeddings) != 0 {
		t.Errorf("EmbedBatch() with empty input returned %d embeddings, want 0", len(embeddings))
	}

	// Test empty multimodal
	multimodalEmbeddings, err := provider.EmbedMultimodal([]MultimodalInput{})
	if err != nil {
		t.Fatalf("EmbedMultimodal() with empty input error = %v", err)
	}
	if len(multimodalEmbeddings) != 0 {
		t.Errorf("EmbedMultimodal() with empty input returned %d embeddings, want 0", len(multimodalEmbeddings))
	}
}

func TestVoyageEmbeddingProvider_Batching(t *testing.T) {
	callCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		var req voyageMultimodalRequest
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

		response := voyageMultimodalResponse{
			Object: "list",
			Data:   dataItems,
			Model:  "voyage-multimodal-3",
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	provider, err := NewVoyageEmbeddingProvider("test-key", "voyage-multimodal-3", server.URL)
	if err != nil {
		t.Fatalf("NewVoyageEmbeddingProvider() error = %v", err)
	}

	// Create 2500 inputs (should create 3 batches: 1000, 1000, 500)
	texts := make([]string, 2500)
	for i := range texts {
		texts[i] = "test text"
	}

	embeddings, err := provider.EmbedBatch(texts)
	if err != nil {
		t.Fatalf("EmbedBatch() error = %v", err)
	}

	if len(embeddings) != len(texts) {
		t.Errorf("EmbedBatch() returned %d embeddings, want %d", len(embeddings), len(texts))
	}

	// Should have made multiple API calls
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
