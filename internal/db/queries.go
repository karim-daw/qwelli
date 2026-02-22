package db

import (
	"context"
	"database/sql/driver"
	"fmt"
	"strings"
	"time"

	"github.com/duckdb/duckdb-go/v2"
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

// AppendFiles bulk-inserts files using DuckDB's Appender API (no UPSERT).
// Caller must ensure no conflicts (fresh index or DeleteFile already called).
func (p *ProjectDB) AppendFiles(files []File) error {
	if len(files) == 0 {
		return nil
	}
	sqlConn, err := p.conn.Conn(context.Background())
	if err != nil {
		return err
	}
	defer sqlConn.Close()

	return sqlConn.Raw(func(driverConn interface{}) error {
		dc := driverConn.(driver.Conn)
		app, err := duckdb.NewAppenderFromConn(dc, "", "files")
		if err != nil {
			return err
		}
		defer app.Close()

		for _, f := range files {
			if err := app.AppendRow(
				f.FileID, f.Path, f.FileType, f.FileHash,
				f.ModifiedAt, f.Size, f.IndexedAt,
			); err != nil {
				return err
			}
		}
		return app.Flush()
	})
}

// AppendChunksAndEmbeddings bulk-inserts chunks and embeddings using DuckDB's Appender API.
// embeddings maps chunk index -> vector; chunks without an embedding are skipped for embeddings table.
func (p *ProjectDB) AppendChunksAndEmbeddings(chunks []Chunk, embeddings map[int][]float32) error {
	if len(chunks) == 0 {
		return nil
	}
	sqlConn, err := p.conn.Conn(context.Background())
	if err != nil {
		return err
	}
	defer sqlConn.Close()

	return sqlConn.Raw(func(driverConn interface{}) error {
		dc := driverConn.(driver.Conn)

		// Bulk-append chunks
		ca, err := duckdb.NewAppenderFromConn(dc, "", "chunks")
		if err != nil {
			return err
		}
		for _, c := range chunks {
			contentType := c.ContentType
			if contentType == "" {
				contentType = "text"
			}
			pageNums := intSliceToInt64(c.PageNumbers)
			if err := ca.AppendRow(
				c.ChunkID, c.FileID, c.FilePath, c.FileType,
				c.ChunkIndex, c.TotalChunks, c.Content,
				pageNums, contentType, c.ImageData,
			); err != nil {
				ca.Close()
				return err
			}
		}
		if err := ca.Flush(); err != nil {
			ca.Close()
			return err
		}
		if err := ca.Close(); err != nil {
			return err
		}

		// Bulk-append embeddings
		ea, err := duckdb.NewAppenderFromConn(dc, "", "embeddings")
		if err != nil {
			return err
		}
		for i, c := range chunks {
			if vec, ok := embeddings[i]; ok {
				if err := ea.AppendRow(c.ChunkID, vec); err != nil {
					ea.Close()
					return err
				}
			}
		}
		if err := ea.Flush(); err != nil {
			ea.Close()
			return err
		}
		return ea.Close()
	})
}

func intSliceToInt64(arr []int) []int64 {
	if len(arr) == 0 {
		return nil
	}
	out := make([]int64, len(arr))
	for i, v := range arr {
		out[i] = int64(v)
	}
	return out
}

// CacheEntry represents a single embedding cache entry.
type CacheEntry struct {
	ContentHash string
	Vector      []float32
}

// LookupEmbeddingCache does a batch lookup of content hashes in the embedding_cache table.
// Returns a map of content_hash → vector for any hits found.
func (p *ProjectDB) LookupEmbeddingCache(hashes []string) (map[string][]float32, error) {
	if len(hashes) == 0 {
		return nil, nil
	}

	result := make(map[string][]float32, len(hashes))
	const batchSize = 500

	for i := 0; i < len(hashes); i += batchSize {
		end := i + batchSize
		if end > len(hashes) {
			end = len(hashes)
		}
		batch := hashes[i:end]

		// Build placeholders: $1, $2, ...
		placeholders := make([]string, len(batch))
		args := make([]interface{}, len(batch))
		for j, h := range batch {
			placeholders[j] = fmt.Sprintf("$%d", j+1)
			args[j] = h
		}

		query := fmt.Sprintf(
			"SELECT content_hash, vector FROM embedding_cache WHERE content_hash IN (%s)",
			strings.Join(placeholders, ", "),
		)

		rows, err := p.conn.Query(query, args...)
		if err != nil {
			return nil, fmt.Errorf("lookup embedding cache: %w", err)
		}

		for rows.Next() {
			var hash string
			var vecIface interface{}
			if err := rows.Scan(&hash, &vecIface); err != nil {
				rows.Close()
				return nil, fmt.Errorf("scan embedding cache: %w", err)
			}
			vec := parseVector(vecIface)
			if vec != nil {
				result[hash] = vec
			}
		}
		rows.Close()
	}

	return result, nil
}

// parseVector converts DuckDB FLOAT[] return values to []float32.
// DuckDB may return []float32, []float64, or []interface{}.
func parseVector(iface interface{}) []float32 {
	switch v := iface.(type) {
	case []float32:
		return v
	case []float64:
		return Float64ToFloat32(v)
	case []interface{}:
		out := make([]float32, 0, len(v))
		for _, item := range v {
			switch n := item.(type) {
			case float32:
				out = append(out, n)
			case float64:
				out = append(out, float32(n))
			default:
				return nil
			}
		}
		return out
	default:
		return nil
	}
}

// StoreEmbeddingCache bulk-inserts embedding cache entries, skipping duplicates.
// Uses INSERT OR IGNORE to handle duplicate content hashes gracefully.
func (p *ProjectDB) StoreEmbeddingCache(entries []CacheEntry) error {
	if len(entries) == 0 {
		return nil
	}

	// Deduplicate entries by content hash (same batch may have identical chunks)
	seen := make(map[string]struct{}, len(entries))
	unique := make([]CacheEntry, 0, len(entries))
	for _, e := range entries {
		if _, exists := seen[e.ContentHash]; !exists {
			seen[e.ContentHash] = struct{}{}
			unique = append(unique, e)
		}
	}

	for _, e := range unique {
		_, err := p.conn.Exec(
			`INSERT OR IGNORE INTO embedding_cache (content_hash, vector, created_at) VALUES ($1, $2, $3)`,
			e.ContentHash, e.Vector, time.Now(),
		)
		if err != nil {
			return fmt.Errorf("insert embedding cache: %w", err)
		}
	}
	return nil
}
