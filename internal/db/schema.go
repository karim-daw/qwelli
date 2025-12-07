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
        CREATE SEQUENCE IF NOT EXISTS models_seq START 1;
        CREATE TABLE IF NOT EXISTS models (
            model_id INTEGER PRIMARY KEY DEFAULT nextval('models_seq'),
            model_name TEXT UNIQUE NOT NULL,
            dimension INTEGER NOT NULL,
            created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
        );
        `,
		`
        CREATE TABLE IF NOT EXISTS embeddings (
            doc_id TEXT NOT NULL REFERENCES documents(doc_id),
            model_id INTEGER NOT NULL REFERENCES models(model_id),
            vector FLOAT[],
            PRIMARY KEY (doc_id, model_id)
        );
        `,
		`
        CREATE TABLE IF NOT EXISTS metadata (
            key TEXT PRIMARY KEY,
            value TEXT
        );
        `,
	}
}
