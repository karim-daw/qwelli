package db

import (
	"fmt"
)

func (p *ProjectDB) InsertDocument(document Document) error {
	_, err := p.conn.Exec(`
		INSERT INTO documents (doc_id, path, file_type, modified_at, size, metadata, content)
		VALUES (?, ?, ?, ?, ?, ?, ?)
	`,
		document.ID,
		document.Path,
		document.FileType,
		document.ModifiedAt,
		document.Size,
		document.Metadata,
		document.Content,
	)
	if err != nil {
		return fmt.Errorf("failed to insert document: %w", err)
	}
	fmt.Println("Document inserted successfully")
	return err
}

func (p *ProjectDB) InsertEmbedding(embed Embedding) error {
	if len(embed.Vector) != p.VectorDim {
		return fmt.Errorf("expected vector dim %d, got %d", p.VectorDim, len(embed.Vector))
	}

	_, err := p.conn.Exec(`
        INSERT OR REPLACE INTO embeddings (doc_id, vector)
        VALUES (?, ?)
    `,
		embed.DocID,
		embed.Vector, // Pass float32 directly - DuckDB expects float32 for FLOAT arrays
	)
	return err
}

func (p *ProjectDB) LoadAllEmbeddings() ([]Embedding, error) {
	rows, err := p.conn.Query(`SELECT doc_id, vector FROM embeddings`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	result := []Embedding{}
	for rows.Next() {
		var id string
		var vecInterface interface{}

		if err := rows.Scan(&id, &vecInterface); err != nil {
			return nil, err
		}

		// DuckDB returns arrays as []interface{}, convert to []float32
		vecInterfaceSlice, ok := vecInterface.([]interface{})
		if !ok {
			return nil, fmt.Errorf("unexpected vector type: %T", vecInterface)
		}

		vec := make([]float32, len(vecInterfaceSlice))
		for i, v := range vecInterfaceSlice {
			switch val := v.(type) {
			case float32:
				vec[i] = val
			case float64:
				vec[i] = float32(val)
			case int:
				vec[i] = float32(val)
			case int64:
				vec[i] = float32(val)
			default:
				return nil, fmt.Errorf("unexpected vector element type: %T", v)
			}
		}

		result = append(result, Embedding{
			DocID:  id,
			Vector: vec,
		})
	}

	return result, nil
}
