package indexer

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
)

func TestNewOpenAIProvider_Validation(t *testing.T) {
	tests := []struct {
		name     string
		apiKey   string
		model    string
		endpoint string
		wantErr  bool
		errMsg   string
	}{
		{
			name:     "missing_api_key",
			apiKey:   "",
			model:    "text-embedding-3-small",
			endpoint: "https://api.openai.com/v1/embeddings",
			wantErr:  true,
			errMsg:   "OpenAI API key required",
		},
		{
			name:     "missing_model",
			apiKey:   "test-key",
			model:    "",
			endpoint: "https://api.openai.com/v1/embeddings",
			wantErr:  true,
			errMsg:   "OpenAI model required",
		},
		{
			name:     "missing_endpoint",
			apiKey:   "test-key",
			model:    "text-embedding-3-small",
			endpoint: "",
			wantErr:  true,
			errMsg:   "OpenAI endpoint required",
		},
		{
			name:     "valid_config",
			apiKey:   "test-key",
			model:    "text-embedding-3-small",
			endpoint: "https://api.openai.com/v1/embeddings",
			wantErr:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			provider, err := NewOpenAIProvider(tt.apiKey, tt.model, tt.endpoint)

			if tt.wantErr {
				if err == nil {
					t.Fatal("NewOpenAIProvider() expected error, got nil")
				}
				if tt.errMsg != "" && err.Error() != tt.errMsg {
					t.Errorf("error = %q, want %q", err.Error(), tt.errMsg)
				}
			} else {
				if err != nil {
					t.Fatalf("NewOpenAIProvider() error = %v, want nil", err)
				}
				if provider == nil {
					t.Fatal("NewOpenAIProvider() returned nil provider")
				}
			}
		})
	}
}

func TestNewEmbedder_Validation(t *testing.T) {
	t.Run("missing_api_key", func(t *testing.T) {
		_, err := NewEmbedder("", "model", "endpoint")
		if err == nil {
			t.Fatal("NewEmbedder() expected error for missing API key")
		}
	})

	t.Run("valid_config", func(t *testing.T) {
		embedder, err := NewEmbedder("test-key", "test-model", "http://localhost/v1/embeddings")
		if err != nil {
			t.Fatalf("NewEmbedder() error = %v", err)
		}
		if embedder == nil {
			t.Fatal("NewEmbedder() returned nil")
		}
	})
}

// mockOpenAIResponse creates a mock OpenAI embedding response
func mockOpenAIResponse(embeddings [][]float64) openAIResponse {
	data := make([]struct {
		Embedding []float64 `json:"embedding"`
	}, len(embeddings))

	for i, emb := range embeddings {
		data[i].Embedding = emb
	}

	return openAIResponse{Data: data}
}

func TestOpenAIProvider_Embed_MockServer(t *testing.T) {
	// Create mock server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify request
		if r.Method != "POST" {
			t.Errorf("expected POST, got %s", r.Method)
		}

		if r.Header.Get("Content-Type") != "application/json" {
			t.Errorf("expected Content-Type: application/json, got %s", r.Header.Get("Content-Type"))
		}

		if r.Header.Get("Authorization") != "Bearer test-api-key" {
			t.Errorf("expected Authorization: Bearer test-api-key, got %s", r.Header.Get("Authorization"))
		}

		// Parse request
		var req openAIRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Errorf("failed to decode request: %v", err)
		}

		if req.Model != "test-model" {
			t.Errorf("expected model test-model, got %s", req.Model)
		}

		// Return mock response (4-dimension embedding)
		resp := mockOpenAIResponse([][]float64{{0.1, 0.2, 0.3, 0.4}})
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	// Create provider with mock server
	provider, err := NewOpenAIProvider("test-api-key", "test-model", server.URL)
	if err != nil {
		t.Fatalf("NewOpenAIProvider() error = %v", err)
	}

	// Test embed
	embedding, err := provider.Embed("test text")
	if err != nil {
		t.Fatalf("Embed() error = %v", err)
	}

	expected := []float32{0.1, 0.2, 0.3, 0.4}
	if len(embedding) != len(expected) {
		t.Fatalf("len(embedding) = %d, want %d", len(embedding), len(expected))
	}

	for i, v := range embedding {
		if v != expected[i] {
			t.Errorf("embedding[%d] = %v, want %v", i, v, expected[i])
		}
	}
}

func TestOpenAIProvider_EmbedBatch_MockServer(t *testing.T) {
	// Create mock server for batch requests
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req openAIRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Errorf("failed to decode request: %v", err)
		}

		// Return 3 embeddings for batch request
		resp := mockOpenAIResponse([][]float64{
			{1.0, 0.0, 0.0, 0.0},
			{0.0, 1.0, 0.0, 0.0},
			{0.0, 0.0, 1.0, 0.0},
		})
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	provider, err := NewOpenAIProvider("test-api-key", "test-model", server.URL)
	if err != nil {
		t.Fatalf("NewOpenAIProvider() error = %v", err)
	}

	texts := []string{"text1", "text2", "text3"}
	embeddings, err := provider.EmbedBatch(texts)
	if err != nil {
		t.Fatalf("EmbedBatch() error = %v", err)
	}

	if len(embeddings) != 3 {
		t.Fatalf("len(embeddings) = %d, want 3", len(embeddings))
	}

	// Verify first embedding
	expected := []float32{1.0, 0.0, 0.0, 0.0}
	for i, v := range embeddings[0] {
		if v != expected[i] {
			t.Errorf("embeddings[0][%d] = %v, want %v", i, v, expected[i])
		}
	}
}

func TestOpenAIProvider_APIError(t *testing.T) {
	// Create mock server that returns an error
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte(`{"error": {"message": "Invalid API key"}}`))
	}))
	defer server.Close()

	provider, err := NewOpenAIProvider("invalid-key", "test-model", server.URL)
	if err != nil {
		t.Fatalf("NewOpenAIProvider() error = %v", err)
	}

	_, err = provider.Embed("test text")
	if err == nil {
		t.Fatal("Embed() expected error for unauthorized response")
	}
}

func TestOpenAIProvider_MalformedResponse(t *testing.T) {
	// Create mock server that returns invalid JSON
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{invalid json`))
	}))
	defer server.Close()

	provider, err := NewOpenAIProvider("test-key", "test-model", server.URL)
	if err != nil {
		t.Fatalf("NewOpenAIProvider() error = %v", err)
	}

	_, err = provider.Embed("test text")
	if err == nil {
		t.Fatal("Embed() expected error for malformed response")
	}
}

func TestOpenAIProvider_WrongEmbeddingCount(t *testing.T) {
	// Create mock server that returns wrong number of embeddings
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Return 2 embeddings when only 1 was requested
		resp := mockOpenAIResponse([][]float64{
			{0.1, 0.2, 0.3, 0.4},
			{0.5, 0.6, 0.7, 0.8},
		})
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	provider, err := NewOpenAIProvider("test-key", "test-model", server.URL)
	if err != nil {
		t.Fatalf("NewOpenAIProvider() error = %v", err)
	}

	_, err = provider.Embed("test text")
	if err == nil {
		t.Fatal("Embed() expected error for wrong embedding count")
	}
}

// Tests using real OpenAI API (skipped if env vars not set)
func TestOpenAIProvider_RealAPI(t *testing.T) {
	apiKey := os.Getenv("QWELLI_EMBEDDING_KEY")
	model := os.Getenv("QWELLI_EMBEDDING_MODEL")
	endpoint := os.Getenv("OPENAI_ENDPOINT")

	if apiKey == "" || model == "" || endpoint == "" {
		t.Skip("Skipping real API test: QWELLI_EMBEDDING_KEY, QWELLI_EMBEDDING_MODEL, or OPENAI_ENDPOINT not set")
	}

	provider, err := NewOpenAIProvider(apiKey, model, endpoint)
	if err != nil {
		t.Fatalf("NewOpenAIProvider() error = %v", err)
	}

	t.Run("embed_single", func(t *testing.T) {
		embedding, err := provider.Embed("Hello, world!")
		if err != nil {
			t.Fatalf("Embed() error = %v", err)
		}

		if len(embedding) == 0 {
			t.Fatal("Embed() returned empty embedding")
		}

		t.Logf("Embedding dimension: %d", len(embedding))
	})

	t.Run("embed_batch", func(t *testing.T) {
		texts := []string{
			"First text about cats",
			"Second text about dogs",
			"Third text about birds",
		}

		embeddings, err := provider.EmbedBatch(texts)
		if err != nil {
			t.Fatalf("EmbedBatch() error = %v", err)
		}

		if len(embeddings) != len(texts) {
			t.Fatalf("len(embeddings) = %d, want %d", len(embeddings), len(texts))
		}

		for i, emb := range embeddings {
			if len(emb) == 0 {
				t.Errorf("embeddings[%d] is empty", i)
			}
		}

		t.Logf("Batch embedding dimensions: %d", len(embeddings[0]))
	})
}

func TestEmbedder_RealAPI(t *testing.T) {
	apiKey := os.Getenv("QWELLI_EMBEDDING_KEY")
	model := os.Getenv("QWELLI_EMBEDDING_MODEL")
	endpoint := os.Getenv("OPENAI_ENDPOINT")

	if apiKey == "" || model == "" || endpoint == "" {
		t.Skip("Skipping real API test: QWELLI_EMBEDDING_KEY, QWELLI_EMBEDDING_MODEL, or OPENAI_ENDPOINT not set")
	}

	embedder, err := NewEmbedder(apiKey, model, endpoint)
	if err != nil {
		t.Fatalf("NewEmbedder() error = %v", err)
	}

	t.Run("embed_via_wrapper", func(t *testing.T) {
		embedding, err := embedder.Embed("Test embedding via wrapper")
		if err != nil {
			t.Fatalf("Embedder.Embed() error = %v", err)
		}

		if len(embedding) == 0 {
			t.Fatal("Embedder.Embed() returned empty embedding")
		}
	})

	t.Run("embed_batch_via_wrapper", func(t *testing.T) {
		texts := []string{"text one", "text two"}
		embeddings, err := embedder.EmbedBatch(texts)
		if err != nil {
			t.Fatalf("Embedder.EmbedBatch() error = %v", err)
		}

		if len(embeddings) != len(texts) {
			t.Fatalf("len(embeddings) = %d, want %d", len(embeddings), len(texts))
		}
	})
}
