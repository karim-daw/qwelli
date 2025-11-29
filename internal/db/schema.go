package db

import (
	"fmt"
	"strings"
)

func buildSchema(vectorDim int) []string {
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
		replace(`
        CREATE TABLE IF NOT EXISTS embeddings (
            doc_id TEXT PRIMARY KEY REFERENCES documents(doc_id),
            vector FLOAT[?]
        );
        `, "?", fmt.Sprintf("%d", vectorDim)),
	}
}

// will replace first "?" with new in s
func replace(s, old, new string) string {
	return strings.Replace(s, old, new, 1)
}
