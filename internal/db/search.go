package db

import (
	"fmt"
	"log"
	"strings"
)

// BuildHNSWIndex creates the HNSW index conditionally if embeddings exist
func (p *ProjectDB) BuildHNSWIndex() error {
	// Check if embeddings table has any rows
	var count int
	err := p.conn.QueryRow("SELECT COUNT(*) FROM embeddings").Scan(&count)
	if err != nil {
		return fmt.Errorf("failed to check embeddings count: %w", err)
	}

	// Only create index if embeddings exist
	if count == 0 {
		return nil
	}

	_, err = p.conn.Exec(`
		CREATE INDEX IF NOT EXISTS hnsw_idx
		ON embeddings USING HNSW (vector)
		WITH (metric = 'cosine')
	`)
	return err
}

// RebuildHNSWIndex drops and recreates the HNSW index (required after embedding changes)
func (p *ProjectDB) RebuildHNSWIndex() error {
	// Drop existing index (if exists)
	_, err := p.conn.Exec("DROP INDEX IF EXISTS hnsw_idx")
	if err != nil {
		return fmt.Errorf("failed to drop HNSW index: %w", err)
	}

	// Check if embeddings exist before rebuilding
	var count int
	err = p.conn.QueryRow("SELECT COUNT(*) FROM embeddings").Scan(&count)
	if err != nil {
		return fmt.Errorf("failed to check embeddings count: %w", err)
	}

	// Only rebuild if embeddings exist
	if count == 0 {
		return nil
	}

	// Rebuild from current embeddings
	_, err = p.conn.Exec(`
		CREATE INDEX hnsw_idx ON embeddings
		USING HNSW(vector)
		WITH (metric='cosine')
	`)
	if err != nil {
		return fmt.Errorf("failed to create HNSW index: %w", err)
	}

	return nil
}

func (p *ProjectDB) SearchANN(query []float32, k int) ([]SearchResult, error) {
	return p.SearchANNWithFilter(query, k, "")
}

// SearchANNWithFilter performs ANN search with optional content type filtering
// contentType can be "text", "image", or "" (empty string) for all types
func (p *ProjectDB) SearchANNWithFilter(query []float32, k int, contentType string) ([]SearchResult, error) {
	vecStr := vectorToString(query)

	// Build WHERE clause for content type filtering
	contentTypeFilter := ""
	if contentType != "" {
		contentTypeFilter = fmt.Sprintf(" AND c.content_type = '%s'", contentType)
	}

	// When filtering by content type, fetch more candidates to ensure we get enough results
	// after filtering. Use a multiplier to account for the distribution of content types.
	candidateLimit := k
	if contentType != "" {
		// Fetch 3x more candidates when filtering to ensure we get enough results
		// This is a heuristic - in practice, the distribution might vary
		candidateLimit = k * 3
		if candidateLimit < 50 {
			candidateLimit = 50 // Minimum candidates to fetch
		}
		if candidateLimit > 1000 {
			candidateLimit = 1000 // Maximum candidates to avoid performance issues
		}
	}

	// Optimized CTE query: avoid double distance calculation
	// Join with chunks table to get denormalized fields (no second JOIN needed)
	queryStr := fmt.Sprintf(`
		WITH ranked_embeddings AS (
			SELECT
				chunk_id,
				array_cosine_distance(vector, %s::FLOAT[%d]) AS distance
			FROM embeddings
			ORDER BY distance
			LIMIT ?
		)
		SELECT
			c.chunk_id,
			c.file_path,
			c.file_type,
			c.content,
			c.chunk_index,
			c.total_chunks,
			c.page_numbers,
			c.content_type,
			c.image_data,
			r.distance
		FROM ranked_embeddings r
		JOIN chunks c ON r.chunk_id = c.chunk_id
		WHERE 1=1%s
		ORDER BY r.distance
		LIMIT ?
	`, vecStr, p.Dimension, contentTypeFilter)

	rows, err := p.conn.Query(queryStr, candidateLimit, k)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []SearchResult
	for rows.Next() {
		var r SearchResult
		var pageNumbersIface interface{} // DuckDB returns arrays as []interface{}
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

// parsePageNumbers parses DuckDB array (can be []interface{} or string) into []int
func parsePageNumbers(iface interface{}) []int {
	if iface == nil {
		return []int{}
	}

	// Handle []interface{} from DuckDB
	if arr, ok := iface.([]interface{}); ok {
		result := make([]int, 0, len(arr))
		for _, v := range arr {
			switch val := v.(type) {
			case int:
				result = append(result, val)
			case int64:
				result = append(result, int(val))
			case int32:
				result = append(result, int(val))
			case int16:
				result = append(result, int(val))
			case int8:
				result = append(result, int(val))
			case float64:
				result = append(result, int(val))
			default:
				// Log unexpected type for debugging
				log.Printf("⚠️  parsePageNumbers: unexpected type %T for value %v", val, val)
			}
		}
		return result
	}

	// Handle string format (fallback)
	if s, ok := iface.(string); ok {
		return parseIntArray(s)
	}

	// Debug: log if we can't parse
	log.Printf("⚠️  parsePageNumbers: cannot parse type %T, value: %v", iface, iface)
	return []int{}
}

// parseIntArray parses DuckDB array string format [1,2,3] into []int
func parseIntArray(s string) []int {
	if s == "" || s == "[]" {
		return []int{}
	}
	// Remove brackets
	s = strings.Trim(s, "[]")
	if s == "" {
		return []int{}
	}
	// Split by comma and parse
	parts := strings.Split(s, ",")
	result := make([]int, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		var val int
		if _, err := fmt.Sscanf(part, "%d", &val); err == nil {
			result = append(result, val)
		}
	}
	return result
}

func vectorToString(v []float32) string {
	parts := make([]string, len(v))
	for i, f := range v {
		parts[i] = fmt.Sprintf("%g", f)
	}
	return "[" + strings.Join(parts, ", ") + "]"
}
