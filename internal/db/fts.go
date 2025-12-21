package db

import (
	"fmt"
	"strings"
)

// SearchFTS performs full-text search using keyword matching with relevance scoring
// Uses LIKE queries with TF-IDF-like scoring for ranking
func (p *ProjectDB) SearchFTS(query string, k int, contentType string) ([]SearchResult, error) {
	// Tokenize query into keywords (split by spaces, remove empty)
	keywords := tokenizeQuery(query)
	if len(keywords) == 0 {
		return []SearchResult{}, nil
	}

	// Build content type filter
	contentTypeFilter := ""
	if contentType != "" {
		contentTypeFilter = fmt.Sprintf(" AND content_type = '%s'", contentType)
	}

	// Build LIKE conditions for each keyword
	// We'll search for keywords in the content field (case-insensitive)
	// DuckDB uses ? for placeholders, but we'll use string formatting for LIKE patterns
	// to avoid SQL injection while keeping it simple
	likeConditions := make([]string, len(keywords))

	for i, keyword := range keywords {
		// Escape single quotes in keyword for SQL safety
		escapedKeyword := strings.ReplaceAll(keyword, "'", "''")
		// Use LIKE with escaped pattern
		likeConditions[i] = fmt.Sprintf("LOWER(content) LIKE LOWER('%%%s%%')", escapedKeyword)
	}

	// Build WHERE clause combining all keywords with AND (all must match)
	whereClause := strings.Join(likeConditions, " AND ")
	if contentTypeFilter != "" {
		whereClause += contentTypeFilter
	}

	// Calculate relevance score using a simple TF-IDF-like approach
	// Score = sum of (keyword matches * weight) for each keyword
	// Weight decreases for common words (simple IDF approximation)
	scoreExpr := buildRelevanceScore(keywords)

	// Query with relevance scoring
	// We normalize the relevance score to a 0-1 range, then convert to distance
	// Higher relevance = lower distance (better match)
	// Use a sigmoid-like function: distance = 1.0 / (1.0 + relevance_score)
	// This ensures:
	// - relevance_score = 0 → distance = 1.0 (no match)
	// - relevance_score = 1 → distance = 0.5 (moderate match)
	// - relevance_score = 10 → distance ≈ 0.09 (excellent match)
	queryStr := fmt.Sprintf(`
		WITH scored_chunks AS (
			SELECT
				chunk_id,
				file_path,
				file_type,
				content,
				chunk_index,
				total_chunks,
				page_numbers,
				content_type,
				image_data,
				%s AS relevance_score
			FROM chunks
			WHERE %s
		)
		SELECT
			chunk_id,
			file_path,
			file_type,
			content,
			chunk_index,
			total_chunks,
			page_numbers,
			content_type,
			image_data,
			(1.0 / (1.0 + relevance_score)) AS distance
		FROM scored_chunks
		ORDER BY relevance_score DESC
		LIMIT ?
	`, scoreExpr, whereClause)

	rows, err := p.conn.Query(queryStr, k)
	if err != nil {
		return nil, fmt.Errorf("FTS query failed: %w", err)
	}
	defer rows.Close()

	var results []SearchResult
	for rows.Next() {
		var r SearchResult
		var pageNumbersIface interface{}
		var imageData []byte
		var contentTypeStr string

		if err := rows.Scan(&r.ChunkID, &r.FilePath, &r.FileType, &r.Content,
			&r.ChunkIndex, &r.TotalChunks, &pageNumbersIface, &contentTypeStr, &imageData, &r.Distance); err != nil {
			return nil, err
		}

		// Parse page_numbers array
		r.PageNumbers = parsePageNumbers(pageNumbersIface)
		r.ContentType = contentTypeStr
		if r.ContentType == "" {
			r.ContentType = "text"
		}
		r.ImageData = imageData
		results = append(results, r)
	}

	return results, nil
}

// tokenizeQuery splits a query into keywords, normalizing and filtering
func tokenizeQuery(query string) []string {
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

// buildRelevanceScore creates a SQL expression for calculating relevance
// Uses a simple TF-IDF-like scoring:
// - Term Frequency (TF): Count of keyword matches in content
// - Inverse Document Frequency (IDF): Weight based on keyword length and rarity
// - Score = sum of (TF * IDF) for all keywords
func buildRelevanceScore(keywords []string) string {
	scoreParts := make([]string, len(keywords))

	for i, keyword := range keywords {
		// Escape keyword for SQL
		escapedKeyword := strings.ReplaceAll(keyword, "'", "''")

		// Calculate TF: count occurrences of keyword in content (case-insensitive)
		// Use LENGTH and REPLACE to count occurrences
		tfExpr := fmt.Sprintf(
			"(LENGTH(LOWER(content)) - LENGTH(REPLACE(LOWER(content), LOWER('%s'), ''))) / LENGTH('%s')",
			escapedKeyword, escapedKeyword,
		)

		// Calculate IDF: weight based on keyword length
		// Longer keywords are more specific and should have higher weight
		// Base weight: 1.0 + (length - 2) * 0.5, capped at reasonable max
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
