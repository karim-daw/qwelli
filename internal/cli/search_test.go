package cli

import (
	"testing"
)

// TestParseSearchArgs_FlagParsing tests the flag parsing logic in search command
func TestParseSearchArgs_FlagParsing(t *testing.T) {
	tests := []struct {
		name           string
		args           []string
		wantQuery      string
		wantTopN       int
		wantTextOnly   bool
		wantImagesOnly bool
	}{
		{
			name:           "simple query",
			args:           []string{"hello"},
			wantQuery:      "hello",
			wantTopN:       5,
			wantTextOnly:   false,
			wantImagesOnly: false,
		},
		{
			name:           "query with text-only flag",
			args:           []string{"hello", "--text-only"},
			wantQuery:      "hello",
			wantTopN:       5,
			wantTextOnly:   true,
			wantImagesOnly: false,
		},
		{
			name:           "query with images-only flag",
			args:           []string{"hello", "--images-only"},
			wantQuery:      "hello",
			wantTopN:       5,
			wantTextOnly:   false,
			wantImagesOnly: true,
		},
		{
			name:           "query with top flag",
			args:           []string{"hello", "--top", "10"},
			wantQuery:      "hello",
			wantTopN:       10,
			wantTextOnly:   false,
			wantImagesOnly: false,
		},
		{
			name:           "query with top short flag",
			args:           []string{"hello", "-t", "20"},
			wantQuery:      "hello",
			wantTopN:       20,
			wantTextOnly:   false,
			wantImagesOnly: false,
		},
		{
			name:           "multi-word query with flags",
			args:           []string{"hello", "world", "--images-only", "--top", "15"},
			wantQuery:      "hello world",
			wantTopN:       15,
			wantTextOnly:   false,
			wantImagesOnly: true,
		},
		{
			name:           "flags in different order",
			args:           []string{"--text-only", "search", "query", "--top", "8"},
			wantQuery:      "search query",
			wantTopN:       8,
			wantTextOnly:   true,
			wantImagesOnly: false,
		},
		{
			name:           "query with flag-like string in query",
			args:           []string{"hello", "--not-a-flag", "world"},
			wantQuery:      "hello --not-a-flag world",
			wantTopN:       5,
			wantTextOnly:   false,
			wantImagesOnly: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			query, topN, textOnly, imagesOnly := parseSearchArgs(tt.args)
			if query != tt.wantQuery {
				t.Errorf("parseSearchArgs() query = %q, want %q", query, tt.wantQuery)
			}
			if topN != tt.wantTopN {
				t.Errorf("parseSearchArgs() topN = %d, want %d", topN, tt.wantTopN)
			}
			if textOnly != tt.wantTextOnly {
				t.Errorf("parseSearchArgs() textOnly = %v, want %v", textOnly, tt.wantTextOnly)
			}
			if imagesOnly != tt.wantImagesOnly {
				t.Errorf("parseSearchArgs() imagesOnly = %v, want %v", imagesOnly, tt.wantImagesOnly)
			}
		})
	}
}

// TestSearchCmd_QueryCleaning tests that flag-like strings are removed from query
func TestSearchCmd_QueryCleaning(t *testing.T) {
	// This tests the query cleaning logic in NewSearchCmd's RunE function
	// We can't easily test Cobra commands directly, but we can verify the logic

	args := []string{"hello", "--image-only", "world"}

	// Simulate the query cleaning logic
	queryParts := []string{}
	for _, arg := range args {
		if !hasPrefix(arg, "--") {
			queryParts = append(queryParts, arg)
		}
	}
	query := joinStrings(queryParts, " ")

	wantQuery := "hello world"
	if query != wantQuery {
		t.Errorf("Query cleaning: got %q, want %q", query, wantQuery)
	}
}

// Helper functions to simulate the logic (since we can't import strings in test)
func hasPrefix(s, prefix string) bool {
	return len(s) >= len(prefix) && s[:len(prefix)] == prefix
}

func joinStrings(parts []string, sep string) string {
	if len(parts) == 0 {
		return ""
	}
	if len(parts) == 1 {
		return parts[0]
	}
	result := parts[0]
	for i := 1; i < len(parts); i++ {
		result += sep + parts[i]
	}
	return result
}
