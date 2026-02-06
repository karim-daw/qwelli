package db

import (
	"database/sql"
	"errors"
	"fmt"

	_ "github.com/duckdb/duckdb-go/v2"
)

type ProjectDB struct {
	Path      string
	Dimension int
	conn      *sql.DB
}

func OpenProjectDB(path string, dimension int) (*ProjectDB, error) {
	if path == "" {
		return nil, errors.New("path is required")
	}

	conn, err := sql.Open("duckdb", fmt.Sprintf("%s?access_mode=read_write", path))
	if err != nil {
		return nil, err
	}

	// Check existing dimension
	var existingDim int
	if err := conn.QueryRow("SELECT value FROM metadata WHERE key = 'dimension'").Scan(&existingDim); err == nil {
		dimension = existingDim
	} else if dimension <= 0 {
		conn.Close()
		return nil, errors.New("dimension required for new database")
	}

	// Load VSS extension
	for _, stmt := range []string{
		"INSTALL vss",
		"LOAD vss",
		"SET hnsw_enable_experimental_persistence = true",
	} {
		if _, err := conn.Exec(stmt); err != nil {
			conn.Close()
			return nil, err
		}
	}

	// Create schema
	for _, stmt := range buildSchema(dimension) {
		if _, err := conn.Exec(stmt); err != nil {
			conn.Close()
			return nil, err
		}
	}

	return &ProjectDB{Path: path, Dimension: dimension, conn: conn}, nil
}

func (p *ProjectDB) SetMetadata(key, value string) error {
	_, err := p.conn.Exec(`INSERT OR REPLACE INTO metadata (key, value) VALUES ($1, $2)`, key, value)
	return err
}

func (p *ProjectDB) GetMetadata(key string) (string, error) {
	var value string
	err := p.conn.QueryRow("SELECT value FROM metadata WHERE key = $1", key).Scan(&value)
	return value, err
}

func (p *ProjectDB) Close() error { return p.conn.Close() }

func (p *ProjectDB) Conn() *sql.DB { return p.conn }

// CountChunks returns the total number of chunks in the database
func (p *ProjectDB) CountChunks() (int, error) {
	var count int
	err := p.conn.QueryRow("SELECT COUNT(*) FROM chunks").Scan(&count)
	return count, err
}

func GetDimensionFromDB(dbPath string) (int, error) {
	conn, err := sql.Open("duckdb", fmt.Sprintf("%s?access_mode=read_only", dbPath))
	if err != nil {
		return 0, err
	}
	defer conn.Close()

	var dimStr string
	if err := conn.QueryRow("SELECT value FROM metadata WHERE key = 'dimension'").Scan(&dimStr); err != nil {
		return 0, err
	}

	var dim int
	fmt.Sscanf(dimStr, "%d", &dim)
	return dim, nil
}
