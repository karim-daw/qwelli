package db

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"github.com/lib/pq"
)

// SearchFTS performs full-text search using keyword matching with relevance scoring
// Uses PostgreSQL's ILIKE for case-insensitive pattern matching with TF-IDF-like scoring
func (db *DB) SearchFTS(ctx context.Context, query string, k int, contentType string) ([]SearchResult, error) {
	// Tokenize query into keywords (split by spaces, remove empty)
	keywords := tokenizeQueryCommon(query)
	if len(keywords) == 0 {
		return []SearchResult{}, nil
	}

	// Build content type filter as WHERE condition
	contentTypeFilter := ""
	if contentType != "" {
		contentTypeFilter = " AND content_type = $2"
	}

	// Build ILIKE conditions for each keyword
	// PostgreSQL uses $N for placeholders, we'll use ILIKE for case-insensitive matching
	likeConditions := make([]string, len(keywords))
	for i, keyword := range keywords {
		// Escape special characters for LIKE pattern
		escapedKeyword := escapeLikePatternPostgres(keyword)
		// Use ILIKE (case-insensitive LIKE)
		likeConditions[i] = fmt.Sprintf("content ILIKE '%%%s%%'", escapedKeyword)
	}

	// Build WHERE clause combining all keywords with AND (all must match)
	whereClause := strings.Join(likeConditions, " AND ")
	if contentTypeFilter != "" {
		whereClause += contentTypeFilter
	}

	// Calculate relevance score using a TF-IDF-like approach
	// Score = sum of (keyword matches * weight) for each keyword
	scoreExpr := buildRelevanceScorePostgresSQL(keywords)

	// Query with relevance scoring
	// Convert relevance score to distance: distance = 1.0 / (1.0 + relevance_score)
	// Higher relevance = lower distance (better match)
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
		LIMIT $1
	`, scoreExpr, whereClause)

	var rows *sql.Rows
	var err error

	if contentType != "" {
		rows, err = db.QueryContext(ctx, queryStr, k, contentType)
	} else {
		rows, err = db.QueryContext(ctx, queryStr, k)
	}

	if err != nil {
		return nil, fmt.Errorf("FTS query failed: %w", err)
	}
	defer rows.Close()

	var results []SearchResult
	for rows.Next() {
		var r SearchResult
		var pageNumbers pq.Int64Array

		err := rows.Scan(
			&r.ChunkID, &r.FilePath, &r.FileType, &r.Content,
			&r.ChunkIndex, &r.TotalChunks, &pageNumbers,
			&r.ContentType, &r.ImageData, &r.Distance,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan FTS result: %w", err)
		}

		// Convert pq.Int64Array to []int
		r.PageNumbers = convertInt64ArrayToIntSlice(pageNumbers)

		// Set default content type if empty
		if r.ContentType == "" {
			r.ContentType = "text"
		}

		results = append(results, r)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating FTS results: %w", err)
	}

	return results, nil
}

// tokenizeQueryCommon splits a query into keywords, normalizing and filtering
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
// Uses a TF-IDF-like scoring
func buildRelevanceScorePostgresSQL(keywords []string) string {
	scoreParts := make([]string, len(keywords))

	for i, keyword := range keywords {
		// Escape keyword for SQL
		escapedKeyword := strings.ReplaceAll(keyword, "'", "''")

		// Calculate TF: count occurrences of keyword in content (case-insensitive)
		tfExpr := fmt.Sprintf(
			"(LENGTH(content) - LENGTH(REPLACE(LOWER(content), LOWER('%s'), '')))::FLOAT / LENGTH('%s')",
			escapedKeyword, escapedKeyword,
		)

		// Calculate IDF: weight based on keyword length
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
