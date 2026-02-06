package textutil

import (
	"testing"
)

func TestSplitIntoSentences(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected []string
	}{
		{
			name:     "simple sentences",
			input:    "Hello world. How are you? I am fine!",
			expected: []string{"Hello world.", "How are you?", "I am fine!"},
		},
		{
			name:     "single sentence no terminator",
			input:    "Hello world",
			expected: []string{"Hello world"},
		},
		{
			name:     "empty string",
			input:    "",
			expected: []string{""},
		},
		{
			name:     "newlines treated as sentence boundary",
			input:    "First line\nSecond line",
			expected: []string{"First line.", "Second line"},
		},
		{
			name:     "multiple terminators",
			input:    "Wait... Really?! Yes.",
			expected: []string{"Wait...", "Really?!", "Yes."},
		},
		{
			name:     "sentence with abbreviation-like period",
			input:    "Dr.Smith went home. He was tired.",
			expected: []string{"Dr.Smith went home.", "He was tired."},
		},
		{
			name:     "only whitespace",
			input:    "   ",
			expected: []string{""},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := SplitIntoSentences(tt.input)
			if len(result) != len(tt.expected) {
				t.Errorf("SplitIntoSentences(%q) returned %d sentences, want %d\ngot:  %v\nwant: %v",
					tt.input, len(result), len(tt.expected), result, tt.expected)
				return
			}
			for i, s := range result {
				if s != tt.expected[i] {
					t.Errorf("SplitIntoSentences(%q)[%d] = %q, want %q",
						tt.input, i, s, tt.expected[i])
				}
			}
		})
	}
}

func TestSplitIntoSentences_AlwaysReturnsAtLeastOne(t *testing.T) {
	inputs := []string{"", "hello", "a.b.c", "no terminator here"}
	for _, input := range inputs {
		result := SplitIntoSentences(input)
		if len(result) == 0 {
			t.Errorf("SplitIntoSentences(%q) returned empty slice, want at least 1 element", input)
		}
	}
}
