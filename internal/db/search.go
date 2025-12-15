package db

import "fmt"

func (p *ProjectDB) BuildHNSWIndex() error {
	// Create HNSW index on the vector column for fast similarity search
	// The index will work with the WHERE model_id = ? filter in searches
	// Using cosine distance metric for semantic similarity
	_, err := p.conn.Exec(`
		CREATE INDEX IF NOT EXISTS hnsw_embeddings_index 
		ON embeddings USING HNSW (vector) 
		WITH (metric = 'cosine', ef_construction = 200, M = 16)
	`)
	if err != nil {
		return fmt.Errorf("failed to create HNSW index: %w", err)
	}
	return nil
}

type SearchResult struct {
	DocID    string
	Distance float64
}

func (p *ProjectDB) SearchANN(query []float32, k int, modelID int) ([]SearchResult, error) {
	rows, err := p.conn.Query(`
        SELECT doc_id, vector <-> $1 AS distance
        FROM embeddings
        WHERE model_id = $2
        ORDER BY vector <-> $1
        LIMIT $3
    `,
		query,   // Pass float32 directly - DuckDB expects float32 for FLOAT arrays
		modelID, // Only search embeddings for this model
		k,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []SearchResult
	for rows.Next() {
		var id string
		var distance float64

		if err := rows.Scan(&id, &distance); err != nil {
			return nil, err
		}

		results = append(results, SearchResult{
			DocID:    id,
			Distance: distance,
		})
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

	err := row.Scan(
		&d.ID,
		&d.Path,
		&d.FileType,
		&d.ModifiedAt,
		&d.Size,
		&metadata,
		&d.Content,
	)
	if err != nil {
		return nil, err
	}

	d.Metadata = metadata
	return &d, nil
}
