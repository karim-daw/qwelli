package db

import (
	"fmt"
	"strings"
)

// SearchFTS performs full-text search using keyword matching with TF-IDF-like scoring
func (p *ProjectDB) SearchFTS(query string, k int, contentType string) ([]SearchResult, error) {
	keywords := tokenizeQuery(query)
	if len(keywords) == 0 {
		return []SearchResult{}, nil
	}

	// Build LIKE conditions using parameterized queries
	likeConditions := make([]string, len(keywords))
	queryArgs := make([]interface{}, 0, len(keywords)+2)

	for i, kw := range keywords {
		likeConditions[i] = "LOWER(content) LIKE LOWER(?)"
		queryArgs = append(queryArgs, "%"+kw+"%")
	}

	where := strings.Join(likeConditions, " AND ")
	if contentType != "" {
		where += " AND content_type = ?"
		queryArgs = append(queryArgs, contentType)
	}

	scoreExpr := buildRelevanceScore(keywords)

	q := fmt.Sprintf(`
		WITH scored AS (
			SELECT chunk_id, file_path, file_type, content,
				chunk_index, total_chunks, page_numbers, content_type, image_data,
				%s AS relevance
			FROM chunks WHERE %s
		)
		SELECT chunk_id, file_path, file_type, content,
			chunk_index, total_chunks, page_numbers, content_type, image_data,
			(1.0 / (1.0 + relevance)) AS distance
		FROM scored ORDER BY relevance DESC LIMIT ?
	`, scoreExpr, where)

	queryArgs = append(queryArgs, k)
	return p.querySearchResults(q, queryArgs...)
}

// tokenizeQuery splits query into keywords, removing stop words
func tokenizeQuery(query string) []string {
	stopWords := map[string]bool{
		"the": true, "a": true, "an": true, "and": true, "or": true,
		"but": true, "in": true, "on": true, "at": true, "to": true,
		"for": true, "of": true, "with": true, "by": true, "is": true,
		"are": true, "was": true, "were": true, "be": true, "been": true,
	}

	var keywords []string
	for _, word := range strings.Fields(strings.ToLower(query)) {
		word = strings.Trim(word, ".,!?;:()[]{}\"'")
		if len(word) >= 2 && !stopWords[word] {
			keywords = append(keywords, word)
		}
	}
	return keywords
}

// buildRelevanceScore creates a SQL expression for TF-IDF-like scoring
func buildRelevanceScore(keywords []string) string {
	parts := make([]string, len(keywords))
	for i, kw := range keywords {
		escaped := strings.ReplaceAll(kw, "'", "''")
		tf := fmt.Sprintf("(LENGTH(LOWER(content)) - LENGTH(REPLACE(LOWER(content), LOWER('%s'), ''))) / LENGTH('%s')", escaped, escaped)
		weight := ClampF(1.0+float64(len(kw)-2)*0.5, 1.0, 5.0)
		parts[i] = fmt.Sprintf("(%s * %.2f)", tf, weight)
	}
	return strings.Join(parts, " + ")
}
