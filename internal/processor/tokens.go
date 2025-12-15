package processor

// EstimateTokens provides a rough estimate of tokens for a text
// Rule of thumb: ~3 characters per token for English text (conservative estimate)
// This is used to determine chunk sizes and validate content before sending to embedding API
func EstimateTokens(text string) int {
	// Conservative estimate: 3 chars per token (safer than 4)
	return (len(text) + 2) / 3
}
