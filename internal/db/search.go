package db

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/lib/pq"
	"github.com/pgvector/pgvector-go"
)

// BuildHNSWIndex creates the HNSW index conditionally if embeddings exist
// For PostgreSQL, the index is already created in the schema, so this is a no-op
// but we keep it for compatibility with the existing interface
func (db *DB) BuildHNSWIndex(ctx context.Context) error {
	// Check if embeddings table has any rows
	var count int
	err := db.QueryRowContext(ctx, "SELECT COUNT(*) FROM embeddings").Scan(&count)
	if err != nil {
		return fmt.Errorf("failed to check embeddings count: %w", err)
	}

	// Index is already created in schema, nothing more to do
	if count == 0 {
		return nil
	}

	return nil
}

// RebuildHNSWIndex drops and recreates the HNSW index (required after bulk embedding changes)
func (db *DB) RebuildHNSWIndex(ctx context.Context) error {
	// Drop existing index (if exists)
	_, err := db.ExecContext(ctx, "DROP INDEX IF EXISTS idx_embeddings_hnsw")
	if err != nil {
		return fmt.Errorf("failed to drop HNSW index: %w", err)
	}

	// Check if embeddings exist before rebuilding
	var count int
	err = db.QueryRowContext(ctx, "SELECT COUNT(*) FROM embeddings").Scan(&count)
	if err != nil {
		return fmt.Errorf("failed to check embeddings count: %w", err)
	}

	// Only rebuild if embeddings exist
	if count == 0 {
		return nil
	}

	// Rebuild HNSW index with cosine distance
	_, err = db.ExecContext(ctx, `
		CREATE INDEX idx_embeddings_hnsw ON embeddings
		USING hnsw (embedding vector_cosine_ops)
	`)
	if err != nil {
		return fmt.Errorf("failed to create HNSW index: %w", err)
	}

	return nil
}

// SearchANN performs approximate nearest neighbor search
func (db *DB) SearchANN(ctx context.Context, query []float32, k int) ([]SearchResult, error) {
	return db.SearchANNWithPathFilter(ctx, query, k, "", "")
}

// SearchANNWithFilter performs ANN search with optional content type filtering
// contentType can be "text", "image", or "" (empty string) for all types
func (db *DB) SearchANNWithFilter(ctx context.Context, query []float32, k int, contentType string) ([]SearchResult, error) {
	return db.SearchANNWithPathFilter(ctx, query, k, contentType, "")
}

// SearchANNWithPathFilter performs ANN search with optional content type and path prefix filtering
// contentType can be "text", "image", or "" (empty string) for all types
// pathPrefix filters results to only include files whose path starts with the given prefix (or "" for all)
func (db *DB) SearchANNWithPathFilter(ctx context.Context, query []float32, k int, contentType string, pathPrefix string) ([]SearchResult, error) {
	// When filtering, fetch more candidates to ensure we get enough results
	candidateLimit := k
	if contentType != "" || pathPrefix != "" {
		// Fetch 3x more candidates when filtering to ensure we get enough results
		candidateLimit = k * 3
		if candidateLimit < 50 {
			candidateLimit = 50 // Minimum candidates to fetch
		}
		if candidateLimit > 1000 {
			candidateLimit = 1000 // Maximum candidates to avoid performance issues
		}
	}

	// Build query with optional filters
	var rows *sql.Rows
	var err error

	queryVec := pgvector.NewVector(query)

	// Build WHERE clauses and arguments dynamically
	whereClause := ""
	args := []interface{}{queryVec}
	argNum := 2

	if contentType != "" && pathPrefix != "" {
		// Both filters - use pathPrefix + "/%" to match files inside the folder only
		whereClause = fmt.Sprintf("WHERE c.content_type = $%d AND c.file_path LIKE $%d", argNum, argNum+1)
		args = append(args, contentType, pathPrefix+"/%")
		argNum += 2
	} else if contentType != "" {
		// Content type filter only
		whereClause = fmt.Sprintf("WHERE c.content_type = $%d", argNum)
		args = append(args, contentType)
		argNum++
	} else if pathPrefix != "" {
		// Path prefix filter only - use pathPrefix + "/%" to match files inside the folder only
		whereClause = fmt.Sprintf("WHERE c.file_path LIKE $%d", argNum)
		args = append(args, pathPrefix+"/%")
		argNum++
	}

	args = append(args, candidateLimit)

	query_sql := fmt.Sprintf(`
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
			(e.embedding <=> $1) AS distance
		FROM embeddings e
		JOIN chunks c ON e.chunk_id = c.chunk_id
		%s
		ORDER BY distance ASC
		LIMIT $%d`, whereClause, argNum)

	rows, err = db.QueryContext(ctx, query_sql, args...)

	if err != nil {
		return nil, fmt.Errorf("ANN search failed: %w", err)
	}
	defer rows.Close()

	results, err := db.scanSearchResults(rows)
	if err != nil {
		return nil, err
	}

	// If we fetched more candidates for filtering, trim to k results
	if len(results) > k {
		return results[:k], nil
	}

	return results, nil
}

// scanSearchResults scans SQL rows into SearchResult slice
func (db *DB) scanSearchResults(rows *sql.Rows) ([]SearchResult, error) {
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
			return nil, fmt.Errorf("failed to scan result: %w", err)
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
		return nil, fmt.Errorf("error iterating results: %w", err)
	}

	return results, nil
}
