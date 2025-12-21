package db

import "fmt"

func buildSchema(dimension int) []string {
	return []string{
		`CREATE TABLE IF NOT EXISTS files (
			file_id TEXT PRIMARY KEY,
			path TEXT NOT NULL UNIQUE,
			file_type TEXT NOT NULL,
			file_hash TEXT NOT NULL,
			modified_at TIMESTAMP NOT NULL,
			size BIGINT NOT NULL,
			indexed_at TIMESTAMP NOT NULL
		)`,
		`CREATE TABLE IF NOT EXISTS chunks (
			chunk_id TEXT PRIMARY KEY,
			file_id TEXT NOT NULL,
			file_path TEXT NOT NULL,
			file_type TEXT NOT NULL,
			chunk_index INT NOT NULL,
			total_chunks INT NOT NULL,
			content TEXT NOT NULL,
			page_numbers INT[],
			content_type TEXT DEFAULT 'text',
			image_data BLOB
		)`,
		`CREATE INDEX IF NOT EXISTS idx_chunks_file_id ON chunks(file_id)`,
		`CREATE INDEX IF NOT EXISTS idx_chunks_content_type ON chunks(content_type)`,
		fmt.Sprintf(`CREATE TABLE IF NOT EXISTS embeddings (
			chunk_id TEXT PRIMARY KEY,
			vector FLOAT[%d]
		)`, dimension),
		`CREATE TABLE IF NOT EXISTS metadata (
			key TEXT PRIMARY KEY,
			value TEXT
		)`,
		fmt.Sprintf(`INSERT OR REPLACE INTO metadata (key, value) VALUES ('dimension', '%d')`, dimension),
		`INSERT OR REPLACE INTO metadata (key, value) VALUES ('schema_version', '2')`,
	}
}
