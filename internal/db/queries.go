package db

import "fmt"
// TODO: add content_type 
func (p *ProjectDB) InsertDocument(doc Document) error {
	var metadataStr string
	switch v := doc.TextMetadata.(type) {
	case string:
		metadataStr = v
	case []byte:
		metadataStr = string(v)
	default:
		metadataStr = "{}"
	}

	_, err := p.conn.Exec(`
		INSERT OR REPLACE INTO documents (doc_id, path, file_type, modified_at, size, text_metadata, content)
		VALUES ($1, $2, $3, $4, $5, $6, $7)`,
		doc.ID, doc.Path, doc.FileType, doc.ModifiedAt.Format("2006-01-02 15:04:05"), doc.Size, metadataStr, doc.Content,
	)
	return err
}

func (p *ProjectDB) InsertEmbedding(emb Embedding) error {
	_, err := p.conn.Exec(`INSERT OR REPLACE INTO embeddings (doc_id, vector) VALUES ($1, $2)`, emb.DocID, emb.Vector)
	return err
}

func (p *ProjectDB) LoadAllEmbeddings() ([]Embedding, error) {
	rows, err := p.conn.Query(`SELECT doc_id, vector FROM embeddings`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []Embedding
	for rows.Next() {
		var id string
		var vecIface interface{}
		if err := rows.Scan(&id, &vecIface); err != nil {
			return nil, err
		}

		vecSlice, ok := vecIface.([]interface{})
		if !ok {
			return nil, fmt.Errorf("unexpected vector type: %T", vecIface)
		}

		vec := make([]float32, len(vecSlice))
		for i, v := range vecSlice {
			switch val := v.(type) {
			case float32:
				vec[i] = val
			case float64:
				vec[i] = float32(val)
			}
		}
		result = append(result, Embedding{DocID: id, Vector: vec})
	}
	return result, nil
}
