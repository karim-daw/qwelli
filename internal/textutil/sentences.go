package textutil

import "strings"

// SplitIntoSentences splits text into sentences
// Simple implementation that splits on common sentence terminators
func SplitIntoSentences(text string) []string {
	// Replace newlines with spaces to treat them as sentence boundaries
	text = strings.ReplaceAll(text, "\n", ". ")

	var sentences []string
	var current strings.Builder

	runes := []rune(text)
	for i := 0; i < len(runes); i++ {
		r := runes[i]
		current.WriteRune(r)

		// Check for sentence terminators
		if r == '.' || r == '!' || r == '?' {
			// Look ahead to see if followed by space or end of text
			if i+1 >= len(runes) || runes[i+1] == ' ' {
				sentence := strings.TrimSpace(current.String())
				if sentence != "" {
					sentences = append(sentences, sentence)
				}
				current.Reset()
			}
		}
	}

	// Add any remaining text
	remaining := strings.TrimSpace(current.String())
	if remaining != "" {
		sentences = append(sentences, remaining)
	}

	// If no sentences were found, return the whole text
	if len(sentences) == 0 {
		sentences = append(sentences, strings.TrimSpace(text))
	}

	return sentences
}
