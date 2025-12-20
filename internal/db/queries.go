package db

import (
	"database/sql"
	"fmt"
	"strings"
	"time"
)

// File CRUD operations

func (p *ProjectDB) InsertFile(file File) error {
	_, err := p.conn.Exec(`
		INSERT INTO files (file_id, path, file_type, file_hash, modified_at, size, indexed_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		ON CONFLICT (file_id) DO UPDATE SET
			path = EXCLUDED.path,
			file_type = EXCLUDED.file_type,
			file_hash = EXCLUDED.file_hash,
			modified_at = EXCLUDED.modified_at,
			size = EXCLUDED.size,
			indexed_at = EXCLUDED.indexed_at`,
		file.FileID, file.Path, file.FileType, file.FileHash,
		file.ModifiedAt.Format("2006-01-02 15:04:05"),
		file.Size, file.IndexedAt.Format("2006-01-02 15:04:05"),
	)
	return err
}

func (p *ProjectDB) GetFile(fileID string) (*File, error) {
	row := p.conn.QueryRow(`
		SELECT file_id, path, file_type, file_hash, modified_at, size, indexed_at
		FROM files WHERE file_id = $1
	`, fileID)

	var f File
	var modifiedAtStr, indexedAtStr string
	if err := row.Scan(&f.FileID, &f.Path, &f.FileType, &f.FileHash,
		&modifiedAtStr, &f.Size, &indexedAtStr); err != nil {
		return nil, err
	}

	var err error
	f.ModifiedAt, err = parseTime(modifiedAtStr)
	if err != nil {
		return nil, fmt.Errorf("failed to parse modified_at: %w", err)
	}
	f.IndexedAt, err = parseTime(indexedAtStr)
	if err != nil {
		return nil, fmt.Errorf("failed to parse indexed_at: %w", err)
	}

	return &f, nil
}

func (p *ProjectDB) GetFileByPath(path string) (*File, error) {
	row := p.conn.QueryRow(`
		SELECT file_id, path, file_type, file_hash, modified_at, size, indexed_at
		FROM files WHERE path = $1
	`, path)

	var f File
	var modifiedAtStr, indexedAtStr string
	if err := row.Scan(&f.FileID, &f.Path, &f.FileType, &f.FileHash,
		&modifiedAtStr, &f.Size, &indexedAtStr); err != nil {
		return nil, err
	}

	var err error
	f.ModifiedAt, err = parseTime(modifiedAtStr)
	if err != nil {
		return nil, fmt.Errorf("failed to parse modified_at: %w", err)
	}
	f.IndexedAt, err = parseTime(indexedAtStr)
	if err != nil {
		return nil, fmt.Errorf("failed to parse indexed_at: %w", err)
	}

	return &f, nil
}

func (p *ProjectDB) GetAllFiles() ([]File, error) {
	rows, err := p.conn.Query(`
		SELECT file_id, path, file_type, file_hash, modified_at, size, indexed_at
		FROM files ORDER BY path
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var files []File
	for rows.Next() {
		var f File
		var modifiedAtStr, indexedAtStr string
		if err := rows.Scan(&f.FileID, &f.Path, &f.FileType, &f.FileHash,
			&modifiedAtStr, &f.Size, &indexedAtStr); err != nil {
			return nil, err
		}

		var err error
		f.ModifiedAt, err = parseTime(modifiedAtStr)
		if err != nil {
			return nil, fmt.Errorf("failed to parse modified_at: %w", err)
		}
		f.IndexedAt, err = parseTime(indexedAtStr)
		if err != nil {
			return nil, fmt.Errorf("failed to parse indexed_at: %w", err)
		}

		files = append(files, f)
	}
	return files, nil
}

// DeleteFile deletes a file and all associated chunks and embeddings (manual cascade)
func (p *ProjectDB) DeleteFile(fileID string) error {
	tx, err := p.conn.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// Delete in order: embeddings → chunks → file
	_, err = tx.Exec(`
		DELETE FROM embeddings
		WHERE chunk_id IN (SELECT chunk_id FROM chunks WHERE file_id = $1)
	`, fileID)
	if err != nil {
		return fmt.Errorf("failed to delete embeddings: %w", err)
	}

	_, err = tx.Exec("DELETE FROM chunks WHERE file_id = $1", fileID)
	if err != nil {
		return fmt.Errorf("failed to delete chunks: %w", err)
	}

	_, err = tx.Exec("DELETE FROM files WHERE file_id = $1", fileID)
	if err != nil {
		return fmt.Errorf("failed to delete file: %w", err)
	}

	return tx.Commit()
}

// Chunk operations

func (p *ProjectDB) InsertChunk(chunk Chunk) error {
	// Convert page_numbers []int to DuckDB array format
	pageNumbersStr := formatIntArray(chunk.PageNumbers)

	_, err := p.conn.Exec(`
		INSERT INTO chunks (
			chunk_id, file_id, file_path, file_type,
			chunk_index, total_chunks, content,
			start_token, end_token, page_numbers
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
		ON CONFLICT (chunk_id) DO UPDATE SET
			file_id = EXCLUDED.file_id,
			file_path = EXCLUDED.file_path,
			file_type = EXCLUDED.file_type,
			chunk_index = EXCLUDED.chunk_index,
			total_chunks = EXCLUDED.total_chunks,
			content = EXCLUDED.content,
			start_token = EXCLUDED.start_token,
			end_token = EXCLUDED.end_token,
			page_numbers = EXCLUDED.page_numbers`,
		chunk.ChunkID, chunk.FileID, chunk.FilePath, chunk.FileType,
		chunk.ChunkIndex, chunk.TotalChunks, chunk.Content,
		chunk.StartToken, chunk.EndToken, pageNumbersStr,
	)
	return err
}

func (p *ProjectDB) GetChunk(chunkID string) (*Chunk, error) {
	row := p.conn.QueryRow(`
		SELECT chunk_id, file_id, file_path, file_type,
			chunk_index, total_chunks, content,
			start_token, end_token, page_numbers
		FROM chunks WHERE chunk_id = $1
	`, chunkID)

	var c Chunk
	var startToken, endToken sql.NullInt64
	var pageNumbersIface interface{}

	if err := row.Scan(&c.ChunkID, &c.FileID, &c.FilePath, &c.FileType,
		&c.ChunkIndex, &c.TotalChunks, &c.Content,
		&startToken, &endToken, &pageNumbersIface); err != nil {
		return nil, err
	}

	if startToken.Valid {
		val := int(startToken.Int64)
		c.StartToken = &val
	}
	if endToken.Valid {
		val := int(endToken.Int64)
		c.EndToken = &val
	}

	c.PageNumbers = parsePageNumbersFromIface(pageNumbersIface)
	return &c, nil
}

func (p *ProjectDB) GetChunksForFile(fileID string) ([]Chunk, error) {
	rows, err := p.conn.Query(`
		SELECT chunk_id, file_id, file_path, file_type,
			chunk_index, total_chunks, content,
			start_token, end_token, page_numbers
		FROM chunks WHERE file_id = $1 ORDER BY chunk_index
	`, fileID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var chunks []Chunk
	for rows.Next() {
		var c Chunk
		var startToken, endToken sql.NullInt64
		var pageNumbersIface interface{}

		if err := rows.Scan(&c.ChunkID, &c.FileID, &c.FilePath, &c.FileType,
			&c.ChunkIndex, &c.TotalChunks, &c.Content,
			&startToken, &endToken, &pageNumbersIface); err != nil {
			return nil, err
		}

		if startToken.Valid {
			val := int(startToken.Int64)
			c.StartToken = &val
		}
		if endToken.Valid {
			val := int(endToken.Int64)
			c.EndToken = &val
		}

		c.PageNumbers = parsePageNumbersFromIface(pageNumbersIface)
		chunks = append(chunks, c)
	}
	return chunks, nil
}

// Embedding operations

func (p *ProjectDB) InsertEmbedding(emb Embedding) error {
	vecStr := vectorToString(emb.Vector)
	_, err := p.conn.Exec(
		fmt.Sprintf(`INSERT INTO embeddings (chunk_id, vector) VALUES ($1, %s::FLOAT[%d])
			ON CONFLICT (chunk_id) DO UPDATE SET vector = EXCLUDED.vector`, vecStr, p.Dimension),
		emb.ChunkID,
	)
	return err
}

// Helper functions

// parseTime tries multiple time formats to parse a timestamp string
func parseTime(timeStr string) (time.Time, error) {
	timeFormats := []string{
		"2006-01-02 15:04:05",
		time.RFC3339,
		time.RFC3339Nano,
	}

	for _, format := range timeFormats {
		if t, err := time.Parse(format, timeStr); err == nil {
			return t, nil
		}
	}
	return time.Time{}, fmt.Errorf("failed to parse time: %s", timeStr)
}

// formatIntArray converts []int to DuckDB array string format [1,2,3]
func formatIntArray(arr []int) string {
	if len(arr) == 0 {
		return "[]"
	}
	str := "["
	for i, v := range arr {
		if i > 0 {
			str += ","
		}
		str += fmt.Sprintf("%d", v)
	}
	str += "]"
	return str
}

// parsePageNumbersFromIface parses DuckDB array (can be []interface{} or string) into []int
func parsePageNumbersFromIface(iface interface{}) []int {
	if iface == nil {
		return []int{}
	}

	// Handle []interface{} from DuckDB
	if arr, ok := iface.([]interface{}); ok {
		result := make([]int, 0, len(arr))
		for _, v := range arr {
			switch val := v.(type) {
			case int:
				result = append(result, val)
			case int64:
				result = append(result, int(val))
			case float64:
				result = append(result, int(val))
			}
		}
		return result
	}

	// Handle string format (fallback)
	if s, ok := iface.(string); ok {
		return parseIntArrayFromString(s)
	}

	return []int{}
}

// parseIntArrayFromString parses DuckDB array string format [1,2,3] into []int
func parseIntArrayFromString(s string) []int {
	if s == "" || s == "[]" {
		return []int{}
	}
	// Remove brackets
	s = strings.Trim(s, "[]")
	if s == "" {
		return []int{}
	}
	// Split by comma and parse
	parts := strings.Split(s, ",")
	result := make([]int, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		var val int
		if _, err := fmt.Sscanf(part, "%d", &val); err == nil {
			result = append(result, val)
		}
	}
	return result
}
