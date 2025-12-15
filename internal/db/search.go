package db

import (
	"fmt"
	"strings"
)

type SearchResult struct {
	DocID    string
	Distance float64
}

func (p *ProjectDB) BuildHNSWIndex() error {
	_, err := p.conn.Exec(`
		CREATE INDEX IF NOT EXISTS hnsw_idx 
		ON embeddings USING HNSW (vector) 
		WITH (metric = 'cosine')
	`)
	return err
}

func (p *ProjectDB) SearchANN(query []float32, k int) ([]SearchResult, error) {
	vecStr := vectorToString(query)
	rows, err := p.conn.Query(fmt.Sprintf(`
		SELECT doc_id, array_cosine_distance(vector, %s::FLOAT[%d]) AS dist
		FROM embeddings
		ORDER BY array_cosine_distance(vector, %s::FLOAT[%d])
		LIMIT %d
	`, vecStr, p.Dimension, vecStr, p.Dimension, k))
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []SearchResult
	for rows.Next() {
		var r SearchResult
		if err := rows.Scan(&r.DocID, &r.Distance); err != nil {
			return nil, err
		}
		results = append(results, r)
	}
	return results, nil
}

func (p *ProjectDB) GetDocument(id string) (*Document, error) {
	row := p.conn.QueryRow(`
		SELECT doc_id, path, file_type, modified_at, size, metadata, content
		FROM documents WHERE doc_id = ?
	`, id)

	var d Document
	var metadata any
	if err := row.Scan(&d.ID, &d.Path, &d.FileType, &d.ModifiedAt, &d.Size, &metadata, &d.Content); err != nil {
		return nil, err
	}
	d.Metadata = metadata
	return &d, nil
}

func vectorToString(v []float32) string {
	parts := make([]string, len(v))
	for i, f := range v {
		parts[i] = fmt.Sprintf("%g", f)
	}
	return "[" + strings.Join(parts, ", ") + "]"
}
