package db

import (
	"context"
	"fmt"
	"strings"

	"github.com/lib/pq"
)

// SearchFTS performs full-text search using keyword matching with relevance scoring
// Uses PostgreSQL's ILIKE for case-insensitive pattern matching with TF-IDF-like scoring
func (db *DB) SearchFTS(ctx context.Context, query string, k int, contentType string) ([]SearchResult, error) {
	return db.SearchFTSWithPathFilter(ctx, query, k, contentType, "")
}

// SearchFTSWithPathFilter performs full-text search with optional path prefix filtering
func (db *DB) SearchFTSWithPathFilter(ctx context.Context, query string, k int, contentType string, pathPrefix string) ([]SearchResult, error) {
	// Tokenize query into keywords (split by spaces, remove empty)
	keywords := tokenizeQueryCommon(query)
	if len(keywords) == 0 {
		return []SearchResult{}, nil
	}

	// Build ILIKE conditions for each keyword
	likeConditions := make([]string, len(keywords))
	for i, keyword := range keywords {
		escapedKeyword := escapeLikePatternPostgres(keyword)
		likeConditions[i] = fmt.Sprintf("content ILIKE '%%%s%%'", escapedKeyword)
	}

	// Build WHERE clause combining all keywords with AND (all must match)
	whereClause := strings.Join(likeConditions, " AND ")

	// Build args list for parameterized query
	args := []interface{}{k}
	argNum := 2

	// Add content type filter
	if contentType != "" {
		whereClause += fmt.Sprintf(" AND content_type = $%d", argNum)
		args = append(args, contentType)
		argNum++
	}

	// Add path prefix filter - use pathPrefix + "/%" to match files inside the folder only
	if pathPrefix != "" {
		whereClause += fmt.Sprintf(" AND file_path LIKE $%d", argNum)
		args = append(args, pathPrefix+"/%")
		argNum++
	}

	// Calculate relevance score using a TF-IDF-like approach
	scoreExpr := buildRelevanceScorePostgresSQL(keywords)

	// Query with relevance scoring
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

	rows, err := db.QueryContext(ctx, queryStr, args...)
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

		r.PageNumbers = convertInt64ArrayToIntSlice(pageNumbers)

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
