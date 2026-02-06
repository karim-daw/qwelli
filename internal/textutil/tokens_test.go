package textutil

import (
	"strings"
	"testing"
)

func TestEstimateTokens(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected int
	}{
		{
			name:     "empty string",
			input:    "",
			expected: 0,
		},
		{
			name:     "single character",
			input:    "a",
			expected: 1,
		},
		{
			name:     "three characters",
			input:    "abc",
			expected: 1,
		},
		{
			name:     "six characters",
			input:    "abcdef",
			expected: 2,
		},
		{
			name:     "short sentence",
			input:    "Hello world",
			expected: 4, // 11 chars -> (11+2)/3 = 4
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := EstimateTokens(tt.input)
			if result != tt.expected {
				t.Errorf("EstimateTokens(%q) = %d, want %d", tt.input, result, tt.expected)
			}
		})
	}
}

func TestEstimateTokens_ProportionalToLength(t *testing.T) {
	short := "Hello"
	long := strings.Repeat("Hello ", 100)

	shortTokens := EstimateTokens(short)
	longTokens := EstimateTokens(long)

	if longTokens <= shortTokens {
		t.Errorf("longer text should have more tokens: short=%d, long=%d", shortTokens, longTokens)
	}
}

func TestEstimateTokens_NonNegative(t *testing.T) {
	result := EstimateTokens("")
	if result < 0 {
		t.Errorf("EstimateTokens should never return negative, got %d", result)
	}
}
