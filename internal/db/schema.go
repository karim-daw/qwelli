package db

import "fmt"

// buildSchema returns SQL statements to create PostgreSQL tables with pgvector
func buildSchema(dimension int) []string {
	return []string{
		// Indexes table - stores indexed folders
		`CREATE TABLE IF NOT EXISTS indexes (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			name TEXT NOT NULL,
			path TEXT NOT NULL UNIQUE,
			created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
		)`,

		// Files table - metadata about indexed files
		`CREATE TABLE IF NOT EXISTS files (
			file_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			path TEXT NOT NULL UNIQUE,
			file_type TEXT NOT NULL,
			file_hash TEXT NOT NULL,
			size BIGINT NOT NULL,
			modified_at TIMESTAMPTZ NOT NULL,
			indexed_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
		)`,

		// Indexes for files table
		`CREATE INDEX IF NOT EXISTS idx_files_path ON files(path)`,
		`CREATE INDEX IF NOT EXISTS idx_files_hash ON files(file_hash)`,
		`CREATE INDEX IF NOT EXISTS idx_files_indexed_at ON files(indexed_at)`,

		// Chunks table - content segments from files (multimodal: text and images)
		`CREATE TABLE IF NOT EXISTS chunks (
			chunk_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			file_id UUID NOT NULL REFERENCES files(file_id) ON DELETE CASCADE,
			file_path TEXT NOT NULL,
			file_type TEXT NOT NULL,
			chunk_index INTEGER NOT NULL,
			total_chunks INTEGER NOT NULL,
			content TEXT NOT NULL,
			page_numbers INTEGER[],
			content_type TEXT DEFAULT 'text',
			image_data BYTEA,
			created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
		)`,

		// Indexes for chunks table
		`CREATE INDEX IF NOT EXISTS idx_chunks_file_id ON chunks(file_id)`,
		`CREATE INDEX IF NOT EXISTS idx_chunks_file_path ON chunks(file_path)`,

		// Embeddings table - vector embeddings for chunks
		fmt.Sprintf(`CREATE TABLE IF NOT EXISTS embeddings (
			chunk_id UUID PRIMARY KEY REFERENCES chunks(chunk_id) ON DELETE CASCADE,
			embedding vector(%d) NOT NULL
		)`, dimension),

		// HNSW index for fast approximate nearest neighbor search
		// This uses cosine distance as the metric
		`CREATE INDEX IF NOT EXISTS idx_embeddings_hnsw ON embeddings
		 USING hnsw (embedding vector_cosine_ops)`,

		// Metadata table - key-value store for index metadata
		`CREATE TABLE IF NOT EXISTS metadata (
			key TEXT PRIMARY KEY,
			value TEXT NOT NULL
		)`,

		// Initialize metadata if not exists
		`INSERT INTO metadata (key, value)
		 VALUES ('schema_version', '1')
		 ON CONFLICT (key) DO NOTHING`,

		`INSERT INTO metadata (key, value)
		 VALUES ('created_at', NOW()::TEXT)
		 ON CONFLICT (key) DO NOTHING`,
	}
}
