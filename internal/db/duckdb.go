package db

import (
	"database/sql"
	"errors"
	"fmt"

	_ "github.com/duckdb/duckdb-go/v2"
)

type ProjectDB struct {
	Path string
	conn *sql.DB
}

func OpenProjectDB(path string) (*ProjectDB, error) {

	if path == "" {
		return nil, errors.New("path for project database is required")
	}

	conn, err := sql.Open("duckdb", fmt.Sprintf("%s?access_mode=read_write", path))
	if err != nil {
		return nil, fmt.Errorf("failed to open project database: %w", err)
	}

	pdb := &ProjectDB{
		Path: path,
		conn: conn,
	}

	// Load the VSS extension for vector similarity search
	if _, err := conn.Exec("INSTALL vss"); err != nil {
		conn.Close()
		return nil, fmt.Errorf("failed to install vss extension: %w", err)
	}
	if _, err := conn.Exec("LOAD vss"); err != nil {
		conn.Close()
		return nil, fmt.Errorf("failed to load vss extension: %w", err)
	}

	// Enable experimental persistence for HNSW indexes on file-based databases
	// This allows the index to be persisted to disk
	if _, err := conn.Exec("SET hnsw_enable_experimental_persistence = true"); err != nil {
		conn.Close()
		return nil, fmt.Errorf("failed to enable HNSW persistence: %w", err)
	}

	// loop through all statements and execute them
	for _, stmt := range buildSchema() {
		if _, err := conn.Exec(stmt); err != nil {
			conn.Close()
			return nil, fmt.Errorf("failed to create schema: %w", err)
		}
	}

	return pdb, nil
}

func (p *ProjectDB) Close() error {
	return p.conn.Close()
}

func (p *ProjectDB) Conn() *sql.DB {
	return p.conn
}
