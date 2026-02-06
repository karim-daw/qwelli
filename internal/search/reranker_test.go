package search

import (
	"context"
	"testing"

	"github.com/karim-daw/qwelli/internal/engine"
	"github.com/karim-daw/qwelli/internal/voyage"
)

func TestApplyReranking_Disabled(t *testing.T) {
	results := []engine.SearchResult{
		{FilePath: "/a.txt", Content: "first", Distance: 0.1},
		{FilePath: "/b.txt", Content: "second", Distance: 0.2},
	}

	reordered, wasReranked := ApplyReranking(
		context.Background(),
		nil, // client not needed when disabled
		"query",
		results,
		RerankOptions{Enabled: false},
	)

	if wasReranked {
		t.Error("wasReranked should be false when disabled")
	}

	if len(reordered) != len(results) {
		t.Errorf("got %d results, want %d", len(reordered), len(results))
	}

	// Results should be unchanged
	if reordered[0].FilePath != "/a.txt" {
		t.Errorf("first result FilePath = %q, want %q", reordered[0].FilePath, "/a.txt")
	}
}

func TestApplyReranking_EmptyResults(t *testing.T) {
	reordered, wasReranked := ApplyReranking(
		context.Background(),
		nil,
		"query",
		[]engine.SearchResult{},
		RerankOptions{Enabled: true},
	)

	if wasReranked {
		t.Error("wasReranked should be false for empty results")
	}

	if len(reordered) != 0 {
		t.Errorf("got %d results, want 0", len(reordered))
	}
}

func TestApplyReranking_NilResults(t *testing.T) {
	reordered, wasReranked := ApplyReranking(
		context.Background(),
		nil,
		"query",
		nil,
		RerankOptions{Enabled: true},
	)

	if wasReranked {
		t.Error("wasReranked should be false for nil results")
	}

	if reordered != nil {
		t.Errorf("got %v results, want nil", reordered)
	}
}

func TestLogRerankImprovement(t *testing.T) {
	// Verify it doesn't panic with edge cases
	t.Run("nil input", func(t *testing.T) {
		logRerankImprovement(nil)
	})

	t.Run("empty input", func(t *testing.T) {
		logRerankImprovement([]voyage.RerankResult{})
	})

	t.Run("top result unchanged", func(t *testing.T) {
		logRerankImprovement([]voyage.RerankResult{
			{Index: 0, RelevanceScore: 0.95},
		})
	})

	t.Run("result promoted", func(t *testing.T) {
		logRerankImprovement([]voyage.RerankResult{
			{Index: 2, RelevanceScore: 0.95},
			{Index: 0, RelevanceScore: 0.80},
		})
	})
}
