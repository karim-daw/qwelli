package db

import (
	"fmt"
)

func (p *ProjectDB) InsertDocument(document Document) error {
	// Convert metadata to string for JSON column
	var metadataStr string
	if document.Metadata != nil {
		switch v := document.Metadata.(type) {
		case string:
			metadataStr = v
		case []byte:
			metadataStr = string(v)
		default:
			metadataStr = "{}"
		}
	} else {
		metadataStr = "{}"
	}

	_, err := p.conn.Exec(`
		INSERT OR REPLACE INTO documents (doc_id, path, file_type, modified_at, size, metadata, content)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
	`,
		document.ID,
		document.Path,
		document.FileType,
		document.ModifiedAt.Format("2006-01-02 15:04:05"),
		document.Size,
		metadataStr,
		document.Content,
	)
	if err != nil {
		return fmt.Errorf("failed to insert document (id=%s, path=%s): %w", document.ID, document.Path, err)
	}
	return nil
}

func (p *ProjectDB) InsertEmbedding(embed Embedding) error {
	_, err := p.conn.Exec(`
        INSERT OR REPLACE INTO embeddings (doc_id, model_id, vector)
        VALUES ($1, $2, $3)
    `,
		embed.DocID,
		embed.ModelID,
		embed.Vector, // Pass float32 directly - DuckDB expects float32 for FLOAT arrays
	)
	return err
}

func (p *ProjectDB) LoadAllEmbeddings() ([]Embedding, error) {
	rows, err := p.conn.Query(`SELECT doc_id, model_id, vector FROM embeddings`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	result := []Embedding{}
	for rows.Next() {
		var id string
		var modelID int
		var vecInterface interface{}

		if err := rows.Scan(&id, &modelID, &vecInterface); err != nil {
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
			DocID:   id,
			ModelID: modelID,
			Vector:  vec,
		})
	}

	return result, nil
}
