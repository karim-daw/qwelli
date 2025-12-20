package db

import "fmt"

func buildSchema(dimension int) []string {
	return []string{
		`CREATE TABLE IF NOT EXISTS documents (
			doc_id TEXT PRIMARY KEY,
			path TEXT,
			file_type TEXT,
			modified_at TIMESTAMP,
			size BIGINT,
			text_metadata JSON,
			content TEXT,
			content_type TEXT,
			image_metadata JSON
		)`,
		fmt.Sprintf(`CREATE TABLE IF NOT EXISTS embeddings (
			doc_id TEXT PRIMARY KEY REFERENCES documents(doc_id),
			vector FLOAT[%d]
		)`, dimension),
		`CREATE TABLE IF NOT EXISTS metadata (
			key TEXT PRIMARY KEY,
			value TEXT
		)`,
	}
}
