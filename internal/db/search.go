package db

func (p *ProjectDB) BuildHNSWIndex() error {
	_, err := p.conn.Exec(`
        CREATE INDEX IF NOT EXISTS hnsw_idx
        ON embeddings
		USING HNSW (vector)
        WITH (metric='cosine');
    `)
	return err
}

type SearchResult struct {
	DocID    string
	Distance float64
}

func (p *ProjectDB) SearchANN(query []float32, k int) ([]SearchResult, error) {
	rows, err := p.conn.Query(`
        SELECT doc_id, vector <-> ? AS distance
        FROM embeddings
        ORDER BY vector <-> ?
        LIMIT ?
    `,
		query, // Pass float32 directly - DuckDB expects float32 for FLOAT arrays
		query, // Used again in ORDER BY
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
