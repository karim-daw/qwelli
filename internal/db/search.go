package db

func (p *ProjectDB) BuildHNSWIndex() error {
	// HNSW indexes are created per model automatically since each model has different dimensions
	// We don't create a global index since FLOAT[] doesn't support HNSW
	// Searches will use WHERE model_id = ? which is indexed by the primary key
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
