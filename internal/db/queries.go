package db

import (
	"fmt"
)

const fileColumns = `file_id, path, file_type, file_hash, modified_at, size, indexed_at`
const chunkColumns = `chunk_id, file_id, file_path, file_type, chunk_index, total_chunks, content, page_numbers, content_type, image_data`

// scanner is satisfied by *sql.Row and *sql.Rows
type scanner interface {
	Scan(dest ...interface{}) error
}

func scanFile(s scanner) (*File, error) {
	var f File
	var modAt, idxAt string
	if err := s.Scan(&f.FileID, &f.Path, &f.FileType, &f.FileHash, &modAt, &f.Size, &idxAt); err != nil {
		return nil, err
	}
	var err error
	if f.ModifiedAt, err = parseTime(modAt); err != nil {
		return nil, fmt.Errorf("parse modified_at: %w", err)
	}
	if f.IndexedAt, err = parseTime(idxAt); err != nil {
		return nil, fmt.Errorf("parse indexed_at: %w", err)
	}
	return &f, nil
}

func scanChunk(s scanner) (*Chunk, error) {
	var c Chunk
	var pageIface interface{}
	var imageData []byte
	if err := s.Scan(&c.ChunkID, &c.FileID, &c.FilePath, &c.FileType,
		&c.ChunkIndex, &c.TotalChunks, &c.Content,
		&pageIface, &c.ContentType, &imageData); err != nil {
		return nil, err
	}
	c.ImageData = imageData
	c.PageNumbers = parsePageNumbers(pageIface)
	if c.ContentType == "" {
		c.ContentType = "text"
	}
	return &c, nil
}

func (p *ProjectDB) InsertFile(file File) error {
	_, err := p.conn.Exec(`
		INSERT INTO files (`+fileColumns+`)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		ON CONFLICT (file_id) DO UPDATE SET
			path = EXCLUDED.path, file_type = EXCLUDED.file_type,
			file_hash = EXCLUDED.file_hash, modified_at = EXCLUDED.modified_at,
			size = EXCLUDED.size, indexed_at = EXCLUDED.indexed_at`,
		file.FileID, file.Path, file.FileType, file.FileHash,
		file.ModifiedAt.Format("2006-01-02 15:04:05"),
		file.Size, file.IndexedAt.Format("2006-01-02 15:04:05"),
	)
	return err
}

func (p *ProjectDB) GetFile(fileID string) (*File, error) {
	return scanFile(p.conn.QueryRow(`SELECT `+fileColumns+` FROM files WHERE file_id = $1`, fileID))
}

func (p *ProjectDB) GetFileByPath(path string) (*File, error) {
	return scanFile(p.conn.QueryRow(`SELECT `+fileColumns+` FROM files WHERE path = $1`, path))
}

func (p *ProjectDB) GetAllFiles() ([]File, error) {
	rows, err := p.conn.Query(`SELECT ` + fileColumns + ` FROM files ORDER BY path`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var files []File
	for rows.Next() {
		f, err := scanFile(rows)
		if err != nil {
			return nil, err
		}
		files = append(files, *f)
	}
	return files, nil
}

// DeleteFile deletes a file and all associated chunks and embeddings
func (p *ProjectDB) DeleteFile(fileID string) error {
	tx, err := p.conn.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	for _, q := range []string{
		`DELETE FROM embeddings WHERE chunk_id IN (SELECT chunk_id FROM chunks WHERE file_id = $1)`,
		`DELETE FROM chunks WHERE file_id = $1`,
		`DELETE FROM files WHERE file_id = $1`,
	} {
		if _, err := tx.Exec(q, fileID); err != nil {
			return err
		}
	}
	return tx.Commit()
}

func (p *ProjectDB) InsertChunk(chunk Chunk) error {
	contentType := chunk.ContentType
	if contentType == "" {
		contentType = "text"
	}
	_, err := p.conn.Exec(`
		INSERT INTO chunks (`+chunkColumns+`)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
		ON CONFLICT (chunk_id) DO UPDATE SET
			file_id = EXCLUDED.file_id, file_path = EXCLUDED.file_path,
			file_type = EXCLUDED.file_type, chunk_index = EXCLUDED.chunk_index,
			total_chunks = EXCLUDED.total_chunks, content = EXCLUDED.content,
			page_numbers = EXCLUDED.page_numbers, content_type = EXCLUDED.content_type,
			image_data = EXCLUDED.image_data`,
		chunk.ChunkID, chunk.FileID, chunk.FilePath, chunk.FileType,
		chunk.ChunkIndex, chunk.TotalChunks, chunk.Content,
		formatIntArray(chunk.PageNumbers), contentType, chunk.ImageData,
	)
	return err
}

func (p *ProjectDB) GetChunk(chunkID string) (*Chunk, error) {
	return scanChunk(p.conn.QueryRow(`SELECT `+chunkColumns+` FROM chunks WHERE chunk_id = $1`, chunkID))
}

func (p *ProjectDB) GetChunksForFile(fileID string) ([]Chunk, error) {
	return p.queryChunks(`SELECT `+chunkColumns+` FROM chunks WHERE file_id = $1 ORDER BY chunk_index`, fileID)
}

func (p *ProjectDB) GetChunksByType(contentType, fileID string) ([]Chunk, error) {
	return p.queryChunks(`SELECT `+chunkColumns+` FROM chunks WHERE file_id = $1 AND content_type = $2 ORDER BY chunk_index`, fileID, contentType)
}

// queryChunks executes a query and scans all rows into Chunks
func (p *ProjectDB) queryChunks(query string, args ...interface{}) ([]Chunk, error) {
	rows, err := p.conn.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var chunks []Chunk
	for rows.Next() {
		c, err := scanChunk(rows)
		if err != nil {
			return nil, err
		}
		chunks = append(chunks, *c)
	}
	return chunks, nil
}

func (p *ProjectDB) InsertEmbedding(emb Embedding) error {
	vecStr := vectorToString(emb.Vector)
	_, err := p.conn.Exec(
		fmt.Sprintf(`INSERT INTO embeddings (chunk_id, vector) VALUES ($1, %s::FLOAT[%d])
			ON CONFLICT (chunk_id) DO UPDATE SET vector = EXCLUDED.vector`, vecStr, p.Dimension),
		emb.ChunkID,
	)
	return err
}
