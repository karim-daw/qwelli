package indexer

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestNewVoyageEmbeddingProvider(t *testing.T) {
	tests := []struct {
		name         string
		apiKey       string
		model        string
		endpoint     string
		wantError    bool
		wantModel    string
		wantEndpoint string
	}{
		{
			name:         "valid config",
			apiKey:       "test-key",
			model:        "voyage-multimodal-3",
			endpoint:     "https://api.voyageai.com/v1/multimodalembeddings",
			wantError:    false,
			wantModel:    "voyage-multimodal-3",
			wantEndpoint: "https://api.voyageai.com/v1/multimodalembeddings",
		},
		{
			name:      "missing api key",
			apiKey:    "",
			model:     "voyage-multimodal-3",
			endpoint:  "https://api.voyageai.com/v1/multimodalembeddings",
			wantError: true,
		},
		{
			name:         "default model",
			apiKey:       "test-key",
			model:        "",
			endpoint:     "https://api.voyageai.com/v1/multimodalembeddings",
			wantError:    false,
			wantModel:    "voyage-multimodal-3",
			wantEndpoint: "https://api.voyageai.com/v1/multimodalembeddings",
		},
		{
			name:         "default endpoint",
			apiKey:       "test-key",
			model:        "voyage-multimodal-3",
			endpoint:     "",
			wantError:    false,
			wantModel:    "voyage-multimodal-3",
			wantEndpoint: "https://api.voyageai.com/v1/multimodalembeddings",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			provider, err := NewVoyageEmbeddingProvider(tt.apiKey, tt.model, tt.endpoint)
			if tt.wantError {
				if err == nil {
					t.Errorf("NewVoyageEmbeddingProvider() expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Fatalf("NewVoyageEmbeddingProvider() error = %v", err)
			}

			if provider == nil {
				t.Fatal("NewVoyageEmbeddingProvider() returned nil")
			}

			if tt.wantModel != "" && provider.model != tt.wantModel {
				t.Errorf("NewVoyageEmbeddingProvider() model = %q, want %q", provider.model, tt.wantModel)
			}

			if tt.wantEndpoint != "" && provider.endpoint != tt.wantEndpoint {
				t.Errorf("NewVoyageEmbeddingProvider() endpoint = %q, want %q", provider.endpoint, tt.wantEndpoint)
			}
		})
	}
}

func TestVoyageEmbeddingProvider_Embed(t *testing.T) {
	// Create mock server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			t.Errorf("Expected POST, got %s", r.Method)
		}

		var req voyageMultimodalRequest
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

	provider, err := NewVoyageEmbeddingProvider("test-key", "voyage-multimodal-3", server.URL)
	if err != nil {
		t.Fatalf("NewVoyageEmbeddingProvider() error = %v", err)
	}

	embedding, err := provider.Embed("test text")
	if err != nil {
		t.Fatalf("Embed() error = %v", err)
	}

	if len(embedding) != 1024 {
		t.Errorf("Embed() returned embedding with length %d, want 1024", len(embedding))
	}
}

func TestVoyageEmbeddingProvider_EmbedBatch(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req voyageMultimodalRequest
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

	texts := []string{"text 1", "text 2", "text 3"}
	embeddings, err := provider.EmbedBatch(context.Background(), texts, nil)
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

func TestVoyageEmbeddingProvider_EmbedImage(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req voyageMultimodalRequest
		json.NewDecoder(r.Body).Decode(&req)

		// Verify image input format
		if len(req.Inputs) != 1 {
			t.Errorf("Expected 1 input, got %d", len(req.Inputs))
		}

		if len(req.Inputs[0].Content) != 1 {
			t.Errorf("Expected 1 content item, got %d", len(req.Inputs[0].Content))
		}

		contentItem := req.Inputs[0].Content[0]
		if contentItem.Type != "image_base64" {
			t.Errorf("Expected content type 'image_base64', got %s", contentItem.Type)
		}

		if contentItem.ImageBase64 == "" {
			t.Error("Expected image input to have 'image_base64' field")
		}

		emb := make([]float64, 1024)
		for i := range emb {
			emb[i] = 0.2
		}
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

	provider, err := NewVoyageEmbeddingProvider("test-key", "voyage-multimodal-3", server.URL)
	if err != nil {
		t.Fatalf("NewVoyageEmbeddingProvider() error = %v", err)
	}

	imageBase64 := "iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAYAAAAfFcSJAAAADUlEQVR42mNk+M9QDwADhgGAWjR9awAAAABJRU5ErkJggg=="
	embedding, err := provider.EmbedImage(imageBase64)
	if err != nil {
		t.Fatalf("EmbedImage() error = %v", err)
	}

	if len(embedding) != 1024 {
		t.Errorf("EmbedImage() returned embedding with length %d, want 1024", len(embedding))
	}
}

func TestVoyageEmbeddingProvider_EmbedMultimodal(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req voyageMultimodalRequest
		json.NewDecoder(r.Body).Decode(&req)

		// Verify mixed inputs - each input should be a separate input object
		if len(req.Inputs) != 3 {
			t.Errorf("Expected 3 input objects, got %d", len(req.Inputs))
		}

		// First should be text
		if len(req.Inputs[0].Content) != 1 || req.Inputs[0].Content[0].Type != "text" {
			t.Errorf("Expected first input to have one text content item")
		}

		// Second should be image
		if len(req.Inputs[1].Content) != 1 || req.Inputs[1].Content[0].Type != "image_base64" {
			t.Errorf("Expected second input to have one image_base64 content item")
		}

		// Third should be text
		if len(req.Inputs[2].Content) != 1 || req.Inputs[2].Content[0].Type != "text" {
			t.Errorf("Expected third input to have one text content item")
		}

		// Return embeddings - one per input object
		dataItems := make([]struct {
			Object    string    `json:"object"`
			Embedding []float64 `json:"embedding"`
			Index     int       `json:"index"`
		}, len(req.Inputs))
		for i := range dataItems {
			emb := make([]float64, 1024)
			for j := range emb {
				emb[j] = 0.15
			}
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

	inputs := []MultimodalInput{
		{Type: "text", Text: "First text"},
		{Type: "image", ImageBase64: "base64imagedata"},
		{Type: "text", Text: "Second text"},
	}

	embeddings, err := provider.EmbedMultimodal(context.Background(), inputs, nil)
	if err != nil {
		t.Fatalf("EmbedMultimodal() error = %v", err)
	}

	if len(embeddings) != len(inputs) {
		t.Errorf("EmbedMultimodal() returned %d embeddings, want %d", len(embeddings), len(inputs))
	}

	for i, emb := range embeddings {
		if len(emb) != 1024 {
			t.Errorf("EmbedMultimodal() embedding %d has length %d, want 1024", i, len(emb))
		}
	}
}

func TestVoyageEmbeddingProvider_CreateMultimodalBatches(t *testing.T) {
	provider, err := NewVoyageEmbeddingProvider("test-key", "voyage-multimodal-3", "https://api.voyageai.com/v1/multimodalembeddings")
	if err != nil {
		t.Fatalf("NewVoyageEmbeddingProvider() error = %v", err)
	}

	// Create inputs that should be split into batches
	inputs := make([]MultimodalInput, 1500) // Exceeds max 1000 per batch
	for i := 0; i < 1500; i++ {
		if i%2 == 0 {
			inputs[i] = MultimodalInput{Type: "text", Text: "text content"}
		} else {
			inputs[i] = MultimodalInput{Type: "image", ImageBase64: "base64data"}
		}
	}

	batches := provider.createMultimodalBatches(inputs)

	// Should create at least 2 batches (1500 > 1000)
	if len(batches) < 2 {
		t.Errorf("createMultimodalBatches() created %d batches, want at least 2", len(batches))
	}

	// Verify no batch exceeds max inputs
	for i, batch := range batches {
		if len(batch) > 1000 {
			t.Errorf("createMultimodalBatches() batch %d has %d inputs, max is 1000", i, len(batch))
		}
	}

	// Verify all inputs are accounted for
	totalInputs := 0
	for _, batch := range batches {
		totalInputs += len(batch)
	}

	if totalInputs != len(inputs) {
		t.Errorf("createMultimodalBatches() total inputs = %d, want %d", totalInputs, len(inputs))
	}
}

func TestVoyageEmbeddingProvider_ErrorHandling(t *testing.T) {
	// Test API error response
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(`{"error": "Invalid API key"}`))
	}))
	defer server.Close()

	provider, err := NewVoyageEmbeddingProvider("invalid-key", "voyage-multimodal-3", server.URL)
	if err != nil {
		t.Fatalf("NewVoyageEmbeddingProvider() error = %v", err)
	}

	_, err = provider.Embed("test")
	if err == nil {
		t.Error("Embed() expected error for invalid API key, got nil")
	}
}
