package server

import (
	"sync"
	"testing"
	"time"
)

func newTestServer() *Server {
	return &Server{
		searchCache: make(map[string]*SearchCacheEntry),
		cacheTTL:    5 * time.Minute,
	}
}

func TestGenerateCacheKey(t *testing.T) {
	s := newTestServer()

	tests := []struct {
		name        string
		query       string
		indexPath   string
		strategy    string
		contentType string
		topK        int
	}{
		{"basic", "hello", "/path", "semantic", "", 5},
		{"with content type", "hello", "/path", "semantic", "text", 5},
		{"different topK", "hello", "/path", "semantic", "", 10},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			key := s.generateCacheKey(tt.query, tt.indexPath, tt.strategy, tt.contentType, tt.topK)
			if key == "" {
				t.Error("generateCacheKey returned empty string")
			}
		})
	}

	// Different parameters should produce different keys
	key1 := s.generateCacheKey("hello", "/path", "semantic", "", 5)
	key2 := s.generateCacheKey("world", "/path", "semantic", "", 5)
	if key1 == key2 {
		t.Error("different queries should produce different cache keys")
	}

	// Same parameters should produce the same key
	key3 := s.generateCacheKey("hello", "/path", "semantic", "", 5)
	if key1 != key3 {
		t.Error("same parameters should produce identical cache keys")
	}
}

func TestSetAndGetCachedResults(t *testing.T) {
	s := newTestServer()

	results := []SearchResultItem{
		{Content: "result 1", FilePath: "/a.txt"},
		{Content: "result 2", FilePath: "/b.txt"},
	}

	cacheKey := "test-key"

	// Initially empty
	_, found := s.getCachedResults(cacheKey)
	if found {
		t.Error("cache should be empty initially")
	}

	// Set and retrieve
	s.setCachedResults(cacheKey, results)

	cached, found := s.getCachedResults(cacheKey)
	if !found {
		t.Fatal("cached results not found after setting")
	}

	if len(cached) != 2 {
		t.Errorf("got %d cached results, want 2", len(cached))
	}

	if cached[0].Content != "result 1" {
		t.Errorf("cached[0].Content = %q, want %q", cached[0].Content, "result 1")
	}
}

func TestCachedResults_Expiration(t *testing.T) {
	s := &Server{
		searchCache: make(map[string]*SearchCacheEntry),
		cacheTTL:    1 * time.Millisecond, // Very short TTL for testing
	}

	results := []SearchResultItem{{Content: "test"}}
	s.setCachedResults("key", results)

	// Should be found immediately
	_, found := s.getCachedResults("key")
	if !found {
		t.Fatal("result should be found immediately after setting")
	}

	// Wait for expiration
	time.Sleep(10 * time.Millisecond)

	// Should be expired
	_, found = s.getCachedResults("key")
	if found {
		t.Error("cached results should have expired")
	}
}

func TestCachedResults_ConcurrentAccess(t *testing.T) {
	s := newTestServer()

	var wg sync.WaitGroup
	for i := 0; i < 50; i++ {
		wg.Add(2)
		go func(idx int) {
			defer wg.Done()
			results := []SearchResultItem{{Content: "test"}}
			s.setCachedResults("concurrent-key", results)
		}(i)
		go func(idx int) {
			defer wg.Done()
			s.getCachedResults("concurrent-key")
		}(i)
	}
	wg.Wait()
}
