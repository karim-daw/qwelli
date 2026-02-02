package db

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/google/uuid"
	"github.com/lib/pq"
	"github.com/pgvector/pgvector-go"
)

// File CRUD operations

// InsertFile creates or updates a file record
func (db *DB) InsertFile(ctx context.Context, file File) error {
	// Generate UUID if not provided
	if file.FileID == "" {
		file.FileID = uuid.New().String()
	}

	query := `
		INSERT INTO files (file_id, path, file_type, file_hash, size, modified_at, indexed_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		ON CONFLICT (path)
		DO UPDATE SET
			file_type = EXCLUDED.file_type,
			file_hash = EXCLUDED.file_hash,
			size = EXCLUDED.size,
			modified_at = EXCLUDED.modified_at,
			indexed_at = EXCLUDED.indexed_at
		RETURNING file_id`

	err := db.QueryRowContext(ctx, query,
		file.FileID, file.Path, file.FileType, file.FileHash,
		file.Size, file.ModifiedAt, file.IndexedAt).Scan(&file.FileID)

	if err != nil {
		return fmt.Errorf("failed to insert file: %w", err)
	}

	return nil
}

// GetFile retrieves a file by ID
func (db *DB) GetFile(ctx context.Context, fileID string) (*File, error) {
	query := `
		SELECT file_id, path, file_type, file_hash, modified_at, size, indexed_at
		FROM files WHERE file_id = $1`

	var f File
	err := db.QueryRowContext(ctx, query, fileID).Scan(
		&f.FileID, &f.Path, &f.FileType, &f.FileHash,
		&f.ModifiedAt, &f.Size, &f.IndexedAt,
	)

	if err != nil {
		return nil, fmt.Errorf("failed to get file: %w", err)
	}

	return &f, nil
}

// GetFileByPath retrieves a file by path
func (db *DB) GetFileByPath(ctx context.Context, path string) (*File, error) {
	query := `
		SELECT file_id, path, file_type, file_hash, modified_at, size, indexed_at
		FROM files WHERE path = $1`

	var f File
	err := db.QueryRowContext(ctx, query, path).Scan(
		&f.FileID, &f.Path, &f.FileType, &f.FileHash,
		&f.ModifiedAt, &f.Size, &f.IndexedAt,
	)

	if err != nil {
		return nil, fmt.Errorf("failed to get file by path: %w", err)
	}

	return &f, nil
}

// GetAllFiles retrieves all files ordered by path
func (db *DB) GetAllFiles(ctx context.Context) ([]File, error) {
	query := `
		SELECT file_id, path, file_type, file_hash, modified_at, size, indexed_at
		FROM files ORDER BY path`

	rows, err := db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to query files: %w", err)
	}
	defer rows.Close()

	var files []File
	for rows.Next() {
		var f File
		err := rows.Scan(
			&f.FileID, &f.Path, &f.FileType, &f.FileHash,
			&f.ModifiedAt, &f.Size, &f.IndexedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan file: %w", err)
		}
		files = append(files, f)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating files: %w", err)
	}

	return files, nil
}

// DeleteFile deletes a file and all associated chunks and embeddings
// PostgreSQL handles cascade deletion automatically via foreign keys
func (db *DB) DeleteFile(ctx context.Context, fileID string) error {
	query := `DELETE FROM files WHERE file_id = $1`

	result, err := db.ExecContext(ctx, query, fileID)
	if err != nil {
		return fmt.Errorf("failed to delete file: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return sql.ErrNoRows
	}

	return nil
}

// Chunk operations

// InsertChunk creates a new chunk
func (db *DB) InsertChunk(ctx context.Context, chunk Chunk) (string, error) {
	// Generate UUID if not provided
	chunkID := chunk.ChunkID
	if chunkID == "" {
		chunkID = uuid.New().String()
	}

	// Set default content_type if not set
	contentType := chunk.ContentType
	if contentType == "" {
		contentType = "text"
	}

	query := `
		INSERT INTO chunks (
			chunk_id, file_id, file_path, file_type,
			chunk_index, total_chunks, content, page_numbers,
			content_type, image_data
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
		ON CONFLICT (chunk_id)
		DO UPDATE SET
			file_id = EXCLUDED.file_id,
			file_path = EXCLUDED.file_path,
			file_type = EXCLUDED.file_type,
			chunk_index = EXCLUDED.chunk_index,
			total_chunks = EXCLUDED.total_chunks,
			content = EXCLUDED.content,
			page_numbers = EXCLUDED.page_numbers,
			content_type = EXCLUDED.content_type,
			image_data = EXCLUDED.image_data`

	_, err := db.ExecContext(ctx, query,
		chunkID, chunk.FileID, chunk.FilePath, chunk.FileType,
		chunk.ChunkIndex, chunk.TotalChunks, chunk.Content,
		pq.Array(chunk.PageNumbers), contentType, chunk.ImageData,
	)

	if err != nil {
		return "", fmt.Errorf("failed to insert chunk: %w", err)
	}

	return chunkID, nil
}

// GetChunk retrieves a chunk by ID
func (db *DB) GetChunk(ctx context.Context, chunkID string) (*Chunk, error) {
	query := `
		SELECT chunk_id, file_id, file_path, file_type,
			chunk_index, total_chunks, content,
			page_numbers, content_type, image_data
		FROM chunks WHERE chunk_id = $1`

	var c Chunk
	var pageNumbers pq.Int64Array

	err := db.QueryRowContext(ctx, query, chunkID).Scan(
		&c.ChunkID, &c.FileID, &c.FilePath, &c.FileType,
		&c.ChunkIndex, &c.TotalChunks, &c.Content,
		&pageNumbers, &c.ContentType, &c.ImageData,
	)

	if err != nil {
		return nil, fmt.Errorf("failed to get chunk: %w", err)
	}

	// Convert pq.Int64Array to []int
	c.PageNumbers = convertInt64ArrayToIntSlice(pageNumbers)

	if c.ContentType == "" {
		c.ContentType = "text"
	}

	return &c, nil
}

// GetChunksForFile retrieves all chunks for a file ordered by chunk index
func (db *DB) GetChunksForFile(ctx context.Context, fileID string) ([]Chunk, error) {
	query := `
		SELECT chunk_id, file_id, file_path, file_type,
			chunk_index, total_chunks, content,
			page_numbers, content_type, image_data
		FROM chunks WHERE file_id = $1 ORDER BY chunk_index`

	rows, err := db.QueryContext(ctx, query, fileID)
	if err != nil {
		return nil, fmt.Errorf("failed to query chunks: %w", err)
	}
	defer rows.Close()

	var chunks []Chunk
	for rows.Next() {
		var c Chunk
		var pageNumbers pq.Int64Array

		err := rows.Scan(
			&c.ChunkID, &c.FileID, &c.FilePath, &c.FileType,
			&c.ChunkIndex, &c.TotalChunks, &c.Content,
			&pageNumbers, &c.ContentType, &c.ImageData,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan chunk: %w", err)
		}

		c.PageNumbers = convertInt64ArrayToIntSlice(pageNumbers)

		if c.ContentType == "" {
			c.ContentType = "text"
		}

		chunks = append(chunks, c)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating chunks: %w", err)
	}

	return chunks, nil
}

// GetChunksByType returns chunks filtered by content type for a specific file
func (db *DB) GetChunksByType(ctx context.Context, contentType string, fileID string) ([]Chunk, error) {
	query := `
		SELECT chunk_id, file_id, file_path, file_type,
			chunk_index, total_chunks, content,
			page_numbers, content_type, image_data
		FROM chunks
		WHERE file_id = $1 AND content_type = $2
		ORDER BY chunk_index`

	rows, err := db.QueryContext(ctx, query, fileID, contentType)
	if err != nil {
		return nil, fmt.Errorf("failed to query chunks by type: %w", err)
	}
	defer rows.Close()

	var chunks []Chunk
	for rows.Next() {
		var c Chunk
		var pageNumbers pq.Int64Array

		err := rows.Scan(
			&c.ChunkID, &c.FileID, &c.FilePath, &c.FileType,
			&c.ChunkIndex, &c.TotalChunks, &c.Content,
			&pageNumbers, &c.ContentType, &c.ImageData,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan chunk: %w", err)
		}

		c.PageNumbers = convertInt64ArrayToIntSlice(pageNumbers)

		if c.ContentType == "" {
			c.ContentType = "text"
		}

		chunks = append(chunks, c)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating chunks: %w", err)
	}

	return chunks, nil
}

// Embedding operations

// InsertEmbedding stores a vector embedding for a chunk
func (db *DB) InsertEmbedding(ctx context.Context, chunkID string, vector []float32) error {
	query := `
		INSERT INTO embeddings (chunk_id, embedding)
		VALUES ($1, $2)
		ON CONFLICT (chunk_id)
		DO UPDATE SET embedding = EXCLUDED.embedding`

	_, err := db.ExecContext(ctx, query, chunkID, pgvector.NewVector(vector))
	if err != nil {
		return fmt.Errorf("failed to insert embedding: %w", err)
	}

	return nil
}

// GetEmbedding retrieves a vector embedding for a chunk
func (db *DB) GetEmbedding(ctx context.Context, chunkID string) (*Embedding, error) {
	query := `SELECT chunk_id, embedding FROM embeddings WHERE chunk_id = $1`

	var e Embedding
	var vec pgvector.Vector

	err := db.QueryRowContext(ctx, query, chunkID).Scan(&e.ChunkID, &vec)
	if err != nil {
		return nil, fmt.Errorf("failed to get embedding: %w", err)
	}

	e.Vector = vec.Slice()

	return &e, nil
}

// DeleteFilesByPathPrefix deletes all files whose path starts with the given prefix
// PostgreSQL handles cascade deletion automatically via foreign keys (chunks, embeddings)
func (db *DB) DeleteFilesByPathPrefix(ctx context.Context, pathPrefix string) (int64, error) {
	query := `DELETE FROM files WHERE path LIKE $1`

	result, err := db.ExecContext(ctx, query, pathPrefix+"%")
	if err != nil {
		return 0, fmt.Errorf("failed to delete files: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("failed to get rows affected: %w", err)
	}

	return rowsAffected, nil
}

// Helper functions

// convertInt64ArrayToIntSlice converts pq.Int64Array to []int
func convertInt64ArrayToIntSlice(arr pq.Int64Array) []int {
	if arr == nil {
		return []int{}
	}

	result := make([]int, len(arr))
	for i, v := range arr {
		result[i] = int(v)
	}

	return result
}
