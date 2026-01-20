package db

import (
	"fmt"
	"strings"
)

// tokenizeQueryCommon splits a query into keywords, normalizing and filtering
// This is shared between DuckDB and PostgreSQL implementations
func tokenizeQueryCommon(query string) []string {
	// Convert to lowercase and split by whitespace
	words := strings.Fields(strings.ToLower(query))

	// Filter out very short words and common stop words
	keywords := make([]string, 0, len(words))
	stopWords := map[string]bool{
		"the": true, "a": true, "an": true, "and": true, "or": true,
		"but": true, "in": true, "on": true, "at": true, "to": true,
		"for": true, "of": true, "with": true, "by": true, "is": true,
		"are": true, "was": true, "were": true, "be": true, "been": true,
	}

	for _, word := range words {
		// Remove punctuation and trim
		word = strings.Trim(word, ".,!?;:()[]{}\"'")
		// Skip very short words (less than 2 chars) and stop words
		if len(word) >= 2 && !stopWords[word] {
			keywords = append(keywords, word)
		}
	}

	return keywords
}

// escapeLikePatternPostgres escapes special characters in LIKE patterns for PostgreSQL
func escapeLikePatternPostgres(s string) string {
	// Escape special LIKE characters: %, _, \
	s = strings.ReplaceAll(s, "\\", "\\\\")
	s = strings.ReplaceAll(s, "%", "\\%")
	s = strings.ReplaceAll(s, "_", "\\_")
	// Escape single quotes for SQL string literal
	s = strings.ReplaceAll(s, "'", "''")
	return s
}

// buildRelevanceScorePostgresSQL creates a SQL expression for calculating relevance
// Uses a TF-IDF-like scoring:
// - Term Frequency (TF): Count of keyword matches in content
// - Inverse Document Frequency (IDF): Weight based on keyword length and rarity
// - Score = sum of (TF * IDF) for all keywords
func buildRelevanceScorePostgresSQL(keywords []string) string {
	scoreParts := make([]string, len(keywords))

	for i, keyword := range keywords {
		// Escape keyword for SQL
		escapedKeyword := strings.ReplaceAll(keyword, "'", "''")

		// Calculate TF: count occurrences of keyword in content (case-insensitive)
		// PostgreSQL: (LENGTH(content) - LENGTH(REPLACE(LOWER(content), LOWER('keyword'), ''))) / LENGTH('keyword')
		tfExpr := fmt.Sprintf(
			"(LENGTH(content) - LENGTH(REPLACE(LOWER(content), LOWER('%s'), '')))::FLOAT / LENGTH('%s')",
			escapedKeyword, escapedKeyword,
		)

		// Calculate IDF: weight based on keyword length
		// Longer keywords are more specific and should have higher weight
		idfWeight := 1.0 + float64(len(keyword)-2)*0.5
		if idfWeight < 1.0 {
			idfWeight = 1.0
		}
		if idfWeight > 5.0 {
			idfWeight = 5.0
		}

		// Score = TF * IDF
		scoreParts[i] = fmt.Sprintf("(%s * %.2f)", tfExpr, idfWeight)
	}

	// Sum all keyword scores
	return strings.Join(scoreParts, " + ")
}
