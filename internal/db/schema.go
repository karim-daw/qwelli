package db

func buildSchema() []string {
	return []string{
		`
        CREATE TABLE IF NOT EXISTS documents (
            doc_id TEXT PRIMARY KEY,
            path TEXT,
            file_type TEXT,
            modified_at TIMESTAMP,
            size BIGINT,
            metadata JSON,
            content TEXT
        );
        `,
		`
        CREATE TABLE IF NOT EXISTS embeddings (
            doc_id TEXT PRIMARY KEY REFERENCES documents(doc_id),
            vector FLOAT[]
        );
        `,
	}
}
