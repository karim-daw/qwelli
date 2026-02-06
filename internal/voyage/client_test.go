package voyage

import (
	"context"
	"errors"
	"testing"
)

func TestNewClient_Validation(t *testing.T) {
	tests := []struct {
		name      string
		config    ClientConfig
		wantError bool
	}{
		{
			name: "valid config",
			config: ClientConfig{
				APIKey:            "test-key",
				EmbeddingModel:    "voyage-multimodal-3.5",
				EmbeddingEndpoint: "https://api.voyageai.com/v1/multimodalembeddings",
				RerankModel:       "rerank-2",
				RerankEndpoint:    "https://api.voyageai.com/v1/rerank",
			},
			wantError: false,
		},
		{
			name: "missing API key",
			config: ClientConfig{
				EmbeddingModel:    "voyage-multimodal-3.5",
				EmbeddingEndpoint: "https://api.voyageai.com/v1/multimodalembeddings",
				RerankModel:       "rerank-2",
				RerankEndpoint:    "https://api.voyageai.com/v1/rerank",
			},
			wantError: true,
		},
		{
			name: "missing embedding model",
			config: ClientConfig{
				APIKey:            "test-key",
				EmbeddingEndpoint: "https://api.voyageai.com/v1/multimodalembeddings",
				RerankModel:       "rerank-2",
				RerankEndpoint:    "https://api.voyageai.com/v1/rerank",
			},
			wantError: true,
		},
		{
			name: "missing embedding endpoint",
			config: ClientConfig{
				APIKey:         "test-key",
				EmbeddingModel: "voyage-multimodal-3.5",
				RerankModel:    "rerank-2",
				RerankEndpoint: "https://api.voyageai.com/v1/rerank",
			},
			wantError: true,
		},
		{
			name: "missing rerank model",
			config: ClientConfig{
				APIKey:            "test-key",
				EmbeddingModel:    "voyage-multimodal-3.5",
				EmbeddingEndpoint: "https://api.voyageai.com/v1/multimodalembeddings",
				RerankEndpoint:    "https://api.voyageai.com/v1/rerank",
			},
			wantError: true,
		},
		{
			name: "missing rerank endpoint",
			config: ClientConfig{
				APIKey:            "test-key",
				EmbeddingModel:    "voyage-multimodal-3.5",
				EmbeddingEndpoint: "https://api.voyageai.com/v1/multimodalembeddings",
				RerankModel:       "rerank-2",
			},
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client, err := NewClient(tt.config)
			if tt.wantError {
				if err == nil {
					t.Error("NewClient() should return error")
				}
				if client != nil {
					t.Error("NewClient() should return nil client on error")
				}
			} else {
				if err != nil {
					t.Errorf("NewClient() unexpected error: %v", err)
				}
				if client == nil {
					t.Error("NewClient() should return non-nil client")
				}
			}
		})
	}
}

func TestClient_ModelAccessors(t *testing.T) {
	client, err := NewClient(ClientConfig{
		APIKey:            "test-key",
		EmbeddingModel:    "voyage-multimodal-3.5",
		EmbeddingEndpoint: "https://api.voyageai.com/v1/multimodalembeddings",
		RerankModel:       "rerank-2",
		RerankEndpoint:    "https://api.voyageai.com/v1/rerank",
	})
	if err != nil {
		t.Fatalf("NewClient() error: %v", err)
	}

	if got := client.EmbeddingModel(); got != "voyage-multimodal-3.5" {
		t.Errorf("EmbeddingModel() = %q, want %q", got, "voyage-multimodal-3.5")
	}

	if got := client.RerankModel(); got != "rerank-2" {
		t.Errorf("RerankModel() = %q, want %q", got, "rerank-2")
	}
}

func TestIsTimeoutError(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected bool
	}{
		{
			name:     "nil error",
			err:      nil,
			expected: false,
		},
		{
			name:     "context deadline exceeded",
			err:      context.DeadlineExceeded,
			expected: true,
		},
		{
			name:     "regular error",
			err:      errors.New("connection refused"),
			expected: false,
		},
		{
			name:     "error containing timeout",
			err:      errors.New("request timeout occurred"),
			expected: true,
		},
		{
			name:     "error containing timed out",
			err:      errors.New("connection timed out"),
			expected: true,
		},
		{
			name:     "Client.Timeout exceeded",
			err:      errors.New("Client.Timeout exceeded while awaiting headers"),
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isTimeoutError(tt.err)
			if result != tt.expected {
				t.Errorf("isTimeoutError(%v) = %v, want %v", tt.err, result, tt.expected)
			}
		})
	}
}

func TestContainsAny(t *testing.T) {
	tests := []struct {
		name     string
		s        string
		substrs  []string
		expected bool
	}{
		{
			name:     "contains first substr",
			s:        "hello world",
			substrs:  []string{"hello", "foo"},
			expected: true,
		},
		{
			name:     "contains second substr",
			s:        "hello world",
			substrs:  []string{"foo", "world"},
			expected: true,
		},
		{
			name:     "contains none",
			s:        "hello world",
			substrs:  []string{"foo", "bar"},
			expected: false,
		},
		{
			name:     "empty string",
			s:        "",
			substrs:  []string{"foo"},
			expected: false,
		},
		{
			name:     "empty substrs",
			s:        "hello",
			substrs:  []string{},
			expected: false,
		},
		{
			name:     "exact match",
			s:        "hello",
			substrs:  []string{"hello"},
			expected: true,
		},
		{
			name:     "substr longer than string",
			s:        "hi",
			substrs:  []string{"hello world"},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := containsAny(tt.s, tt.substrs)
			if result != tt.expected {
				t.Errorf("containsAny(%q, %v) = %v, want %v", tt.s, tt.substrs, result, tt.expected)
			}
		})
	}
}
