package db

import (
	"fmt"
)

// ensureHNSWPersistence re-applies the hnsw_enable_experimental_persistence setting
// on the current connection. This is necessary because database/sql may use a pooled
// connection that never received the session-level SET from OpenProjectDB.
func (p *ProjectDB) ensureHNSWPersistence() error {
	_, err := p.conn.Exec("SET hnsw_enable_experimental_persistence = true")
	return err
}

// BuildHNSWIndex creates the HNSW index if embeddings exist
func (p *ProjectDB) BuildHNSWIndex() error {
	var count int
	if err := p.conn.QueryRow("SELECT COUNT(*) FROM embeddings").Scan(&count); err != nil {
		return fmt.Errorf("check embeddings: %w", err)
	}
	if count == 0 {
		return nil
	}
	if err := p.ensureHNSWPersistence(); err != nil {
		return fmt.Errorf("ensure HNSW persistence: %w", err)
	}
	_, err := p.conn.Exec(`CREATE INDEX IF NOT EXISTS hnsw_idx ON embeddings USING HNSW (vector) WITH (metric = 'cosine')`)
	return err
}

// RebuildHNSWIndex drops and recreates the HNSW index, clearing any stale flag.
func (p *ProjectDB) RebuildHNSWIndex() error {
	if _, err := p.conn.Exec("DROP INDEX IF EXISTS hnsw_idx"); err != nil {
		return fmt.Errorf("drop HNSW index: %w", err)
	}
	var count int
	if err := p.conn.QueryRow("SELECT COUNT(*) FROM embeddings").Scan(&count); err != nil {
		return fmt.Errorf("check embeddings: %w", err)
	}
	if count == 0 {
		_ = p.SetMetadata("hnsw_stale", "false")
		return nil
	}
	if err := p.ensureHNSWPersistence(); err != nil {
		return fmt.Errorf("ensure HNSW persistence: %w", err)
	}
	if _, err := p.conn.Exec(`CREATE INDEX hnsw_idx ON embeddings USING HNSW(vector) WITH (metric='cosine')`); err != nil {
		return fmt.Errorf("create HNSW index: %w", err)
	}
	_ = p.SetMetadata("hnsw_stale", "false")
	return nil
}

// BuildHNSWIndexIfNeeded rebuilds the HNSW index only if new embeddings were added since prevCount.
// For small incremental updates (<5% of total when total >1000), defers the rebuild by marking
// the index as stale. This saves 10-30 seconds on small incremental updates.
// Returns true if a rebuild was performed.
func (p *ProjectDB) BuildHNSWIndexIfNeeded(prevCount int) (rebuilt bool, err error) {
	current, err := p.CountEmbeddings()
	if err != nil {
		return false, fmt.Errorf("count embeddings: %w", err)
	}
	if current == prevCount {
		return false, nil // No new embeddings, skip rebuild
	}

	newEmbeddings := current - prevCount
	changeRatio := float64(newEmbeddings) / float64(current)

	// Defer rebuild for small incremental updates
	if current > 1000 && changeRatio < 0.05 {
		if err := p.SetMetadata("hnsw_stale", "true"); err != nil {
			return false, fmt.Errorf("set hnsw_stale: %w", err)
		}
		return false, nil
	}

	// Clear stale flag and rebuild
	if err := p.RebuildHNSWIndex(); err != nil {
		return false, err
	}
	_ = p.SetMetadata("hnsw_stale", "false")
	return true, nil
}

// IsHNSWStale returns true if the HNSW index has deferred updates pending.
func (p *ProjectDB) IsHNSWStale() bool {
	val, err := p.GetMetadata("hnsw_stale")
	if err != nil {
		return false
	}
	return val == "true"
}

// ForceRebuildHNSWIfStale rebuilds the HNSW index if it's marked as stale.
// Used during full re-indexes to ensure the index is up to date.
func (p *ProjectDB) ForceRebuildHNSWIfStale() (bool, error) {
	if !p.IsHNSWStale() {
		return false, nil
	}
	if err := p.RebuildHNSWIndex(); err != nil {
		return false, err
	}
	_ = p.SetMetadata("hnsw_stale", "false")
	return true, nil
}

func (p *ProjectDB) SearchANN(query []float32, k int) ([]SearchResult, error) {
	return p.SearchANNWithFilter(query, k, "")
}

// SearchANNWithFilter performs approximate nearest neighbor search with optional content type filter
func (p *ProjectDB) SearchANNWithFilter(query []float32, k int, contentType string) ([]SearchResult, error) {
	vecStr := vectorToString(query)

	// Over-fetch when filtering to compensate for filtered-out results
	candidateLimit := k
	if contentType != "" {
		candidateLimit = Clamp(k*3, 50, 1000)
	}

	var q string
	var args []interface{}

	if contentType != "" {
		q = fmt.Sprintf(`
			WITH ranked AS (
				SELECT chunk_id, array_cosine_distance(vector, %s::FLOAT[%d]) AS distance
				FROM embeddings ORDER BY distance LIMIT ?
			)
			SELECT c.chunk_id, c.file_path, c.file_type, c.content,
				c.chunk_index, c.total_chunks, c.page_numbers, c.content_type, c.image_data,
				r.distance
			FROM ranked r JOIN chunks c ON r.chunk_id = c.chunk_id
			WHERE c.content_type = ? ORDER BY r.distance LIMIT ?
		`, vecStr, p.Dimension)
		args = []interface{}{candidateLimit, contentType, k}
	} else {
		q = fmt.Sprintf(`
			WITH ranked AS (
				SELECT chunk_id, array_cosine_distance(vector, %s::FLOAT[%d]) AS distance
				FROM embeddings ORDER BY distance LIMIT ?
			)
			SELECT c.chunk_id, c.file_path, c.file_type, c.content,
				c.chunk_index, c.total_chunks, c.page_numbers, c.content_type, c.image_data,
				r.distance
			FROM ranked r JOIN chunks c ON r.chunk_id = c.chunk_id
			ORDER BY r.distance LIMIT ?
		`, vecStr, p.Dimension)
		args = []interface{}{candidateLimit, k}
	}

	return p.querySearchResults(q, args...)
}

// querySearchResults executes a search query and scans results
func (p *ProjectDB) querySearchResults(query string, args ...interface{}) ([]SearchResult, error) {
	rows, err := p.conn.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []SearchResult
	for rows.Next() {
		var r SearchResult
		var pageIface interface{}
		var imageData []byte
		if err := rows.Scan(&r.ChunkID, &r.FilePath, &r.FileType, &r.Content,
			&r.ChunkIndex, &r.TotalChunks, &pageIface, &r.ContentType, &imageData, &r.Distance); err != nil {
			return nil, err
		}
		r.PageNumbers = parsePageNumbers(pageIface)
		r.ImageData = imageData
		if r.ContentType == "" {
			r.ContentType = "text"
		}
		results = append(results, r)
	}
	return results, nil
}
