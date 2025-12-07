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
		return nil, errors.New("path for project database is required")
	}

	conn, err := sql.Open("duckdb", fmt.Sprintf("%s?access_mode=read_write", path))
	if err != nil {
		return nil, fmt.Errorf("failed to open project database: %w", err)
	}

	// Check if database already exists and has dimension stored
	var existingDimension int
	err = conn.QueryRow("SELECT value FROM metadata WHERE key = 'dimension' LIMIT 1").Scan(&existingDimension)
	if err == nil {
		// Database exists with stored dimension
		dimension = existingDimension
	} else if dimension <= 0 {
		// New database but no dimension provided
		return nil, errors.New("embedding dimension must be positive for new database")
	}

	pdb := &ProjectDB{
		Path:      path,
		Dimension: dimension,
		conn:      conn,
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
	for _, stmt := range buildSchema(dimension) {
		if _, err := conn.Exec(stmt); err != nil {
			conn.Close()
			return nil, fmt.Errorf("failed to create schema: %w", err)
		}
	}

	return pdb, nil
}

// SetMetadata stores a key-value pair in the metadata table
func (p *ProjectDB) SetMetadata(key, value string) error {
	_, err := p.conn.Exec(`INSERT OR REPLACE INTO metadata (key, value) VALUES ($1, $2)`, key, value)
	return err
}

// GetMetadata retrieves a value from the metadata table
func (p *ProjectDB) GetMetadata(key string) (string, error) {
	var value string
	err := p.conn.QueryRow("SELECT value FROM metadata WHERE key = $1", key).Scan(&value)
	return value, err
}

func (p *ProjectDB) Close() error {
	return p.conn.Close()
}

func (p *ProjectDB) Conn() *sql.DB {
	return p.conn
}

// GetDimensionFromDB retrieves the stored dimension from the database
func GetDimensionFromDB(dbPath string) (int, error) {
	conn, err := sql.Open("duckdb", fmt.Sprintf("%s?access_mode=read_only", dbPath))
	if err != nil {
		return 0, fmt.Errorf("failed to open database: %w", err)
	}
	defer conn.Close()

	var dimensionStr string
	err = conn.QueryRow("SELECT value FROM metadata WHERE key = 'dimension'").Scan(&dimensionStr)
	if err != nil {
		return 0, fmt.Errorf("failed to get dimension: %w", err)
	}

	var dimension int
	_, err = fmt.Sscanf(dimensionStr, "%d", &dimension)
	if err != nil {
		return 0, fmt.Errorf("failed to parse dimension: %w", err)
	}

	return dimension, nil
}

// GetOrCreateModel returns the model_id for a given model name, creating it if needed
func (p *ProjectDB) GetOrCreateModel(modelName string, dimension int) (int, error) {
	// Try to get existing model
	var modelID int
	var storedDimension int
	err := p.conn.QueryRow("SELECT model_id, dimension FROM models WHERE model_name = $1", modelName).Scan(&modelID, &storedDimension)

	if err == nil {
		// Model exists - verify dimension matches
		if storedDimension != dimension {
			return 0, fmt.Errorf("model '%s' exists with dimension %d, but current dimension is %d", modelName, storedDimension, dimension)
		}
		return modelID, nil
	}

	// Model doesn't exist - create it
	err = p.conn.QueryRow(`
		INSERT INTO models (model_name, dimension)
		VALUES ($1, $2)
		RETURNING model_id
	`, modelName, dimension).Scan(&modelID)

	if err != nil {
		return 0, fmt.Errorf("failed to create model: %w", err)
	}

	return modelID, nil
}
