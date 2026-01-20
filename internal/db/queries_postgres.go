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
func (p *PostgresDB) InsertFile(ctx context.Context, file File) error {
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

	err := p.db.QueryRowContext(ctx, query,
		file.FileID, file.Path, file.FileType, file.FileHash,
		file.Size, file.ModifiedAt, file.IndexedAt).Scan(&file.FileID)

	if err != nil {
		return fmt.Errorf("failed to insert file: %w", err)
	}

	return nil
}

// GetFile retrieves a file by ID
func (p *PostgresDB) GetFile(ctx context.Context, fileID string) (*File, error) {
	query := `
		SELECT file_id, path, file_type, file_hash, modified_at, size, indexed_at
		FROM files WHERE file_id = $1`

	var f File
	err := p.db.QueryRowContext(ctx, query, fileID).Scan(
		&f.FileID, &f.Path, &f.FileType, &f.FileHash,
		&f.ModifiedAt, &f.Size, &f.IndexedAt,
	)

	if err != nil {
		return nil, fmt.Errorf("failed to get file: %w", err)
	}

	return &f, nil
}

// GetFileByPath retrieves a file by path
func (p *PostgresDB) GetFileByPath(ctx context.Context, path string) (*File, error) {
	query := `
		SELECT file_id, path, file_type, file_hash, modified_at, size, indexed_at
		FROM files WHERE path = $1`

	var f File
	err := p.db.QueryRowContext(ctx, query, path).Scan(
		&f.FileID, &f.Path, &f.FileType, &f.FileHash,
		&f.ModifiedAt, &f.Size, &f.IndexedAt,
	)

	if err != nil {
		return nil, fmt.Errorf("failed to get file by path: %w", err)
	}

	return &f, nil
}

// GetAllFiles retrieves all files ordered by path
func (p *PostgresDB) GetAllFiles(ctx context.Context) ([]File, error) {
	query := `
		SELECT file_id, path, file_type, file_hash, modified_at, size, indexed_at
		FROM files ORDER BY path`

	rows, err := p.db.QueryContext(ctx, query)
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
func (p *PostgresDB) DeleteFile(ctx context.Context, fileID string) error {
	query := `DELETE FROM files WHERE file_id = $1`

	result, err := p.db.ExecContext(ctx, query, fileID)
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
func (p *PostgresDB) InsertChunk(ctx context.Context, chunk Chunk) (string, error) {
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

	_, err := p.db.ExecContext(ctx, query,
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
func (p *PostgresDB) GetChunk(ctx context.Context, chunkID string) (*Chunk, error) {
	query := `
		SELECT chunk_id, file_id, file_path, file_type,
			chunk_index, total_chunks, content,
			page_numbers, content_type, image_data
		FROM chunks WHERE chunk_id = $1`

	var c Chunk
	var pageNumbers pq.Int64Array

	err := p.db.QueryRowContext(ctx, query, chunkID).Scan(
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
func (p *PostgresDB) GetChunksForFile(ctx context.Context, fileID string) ([]Chunk, error) {
	query := `
		SELECT chunk_id, file_id, file_path, file_type,
			chunk_index, total_chunks, content,
			page_numbers, content_type, image_data
		FROM chunks WHERE file_id = $1 ORDER BY chunk_index`

	rows, err := p.db.QueryContext(ctx, query, fileID)
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
func (p *PostgresDB) GetChunksByType(ctx context.Context, contentType string, fileID string) ([]Chunk, error) {
	query := `
		SELECT chunk_id, file_id, file_path, file_type,
			chunk_index, total_chunks, content,
			page_numbers, content_type, image_data
		FROM chunks
		WHERE file_id = $1 AND content_type = $2
		ORDER BY chunk_index`

	rows, err := p.db.QueryContext(ctx, query, fileID, contentType)
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
func (p *PostgresDB) InsertEmbedding(ctx context.Context, chunkID string, vector []float32) error {
	query := `
		INSERT INTO embeddings (chunk_id, embedding)
		VALUES ($1, $2)
		ON CONFLICT (chunk_id)
		DO UPDATE SET embedding = EXCLUDED.embedding`

	_, err := p.db.ExecContext(ctx, query, chunkID, pgvector.NewVector(vector))
	if err != nil {
		return fmt.Errorf("failed to insert embedding: %w", err)
	}

	return nil
}

// GetEmbedding retrieves a vector embedding for a chunk
func (p *PostgresDB) GetEmbedding(ctx context.Context, chunkID string) (*Embedding, error) {
	query := `SELECT chunk_id, embedding FROM embeddings WHERE chunk_id = $1`

	var e Embedding
	var vec pgvector.Vector

	err := p.db.QueryRowContext(ctx, query, chunkID).Scan(&e.ChunkID, &vec)
	if err != nil {
		return nil, fmt.Errorf("failed to get embedding: %w", err)
	}

	e.Vector = vec.Slice()

	return &e, nil
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
