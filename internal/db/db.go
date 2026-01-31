package db

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	_ "github.com/lib/pq"
)

// DB wraps a PostgreSQL connection with pgvector support
type DB struct {
	*sql.DB
	dimension int
}

// NewDB creates a new PostgreSQL connection with pgvector support
// connectionString format: postgresql://user:password@host:port/dbname?sslmode=disable
func NewDB(ctx context.Context, connectionString string, dimension int) (*DB, error) {
	if dimension <= 0 {
		return nil, fmt.Errorf("dimension must be positive, got %d", dimension)
	}

	// Open database connection
	sqlDB, err := sql.Open("postgres", connectionString)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// Configure connection pool with sensible defaults
	sqlDB.SetMaxOpenConns(25)
	sqlDB.SetMaxIdleConns(5)
	sqlDB.SetConnMaxLifetime(time.Hour)
	sqlDB.SetConnMaxIdleTime(15 * time.Minute)

	// Verify connection
	if err := sqlDB.PingContext(ctx); err != nil {
		sqlDB.Close()
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	db := &DB{
		DB:        sqlDB,
		dimension: dimension,
	}

	// Initialize schema and extensions
	if err := db.initSchema(ctx); err != nil {
		sqlDB.Close()
		return nil, fmt.Errorf("failed to initialize schema: %w", err)
	}

	// Check or set dimension metadata
	if err := db.ensureDimension(ctx); err != nil {
		sqlDB.Close()
		return nil, fmt.Errorf("failed to ensure dimension: %w", err)
	}

	return db, nil
}

// initSchema creates necessary extensions and tables
func (db *DB) initSchema(ctx context.Context) error {
	// Enable pgvector extension
	if _, err := db.ExecContext(ctx, "CREATE EXTENSION IF NOT EXISTS vector"); err != nil {
		return fmt.Errorf("failed to create vector extension: %w", err)
	}

	// Create tables using schema builder
	for _, stmt := range buildSchema(db.dimension) {
		if _, err := db.ExecContext(ctx, stmt); err != nil {
			return fmt.Errorf("failed to execute schema statement: %w", err)
		}
	}

	return nil
}

// ensureDimension checks existing dimension or sets it for new database
func (db *DB) ensureDimension(ctx context.Context) error {
	var existingDimStr string
	err := db.QueryRowContext(ctx,
		"SELECT value FROM metadata WHERE key = 'dimension'").Scan(&existingDimStr)

	if err == sql.ErrNoRows {
		// New database, set dimension
		return db.SetMetadata(ctx, "dimension", fmt.Sprintf("%d", db.dimension))
	} else if err != nil {
		return fmt.Errorf("failed to query dimension: %w", err)
	}

	// Parse existing dimension
	var existingDim int
	if _, err := fmt.Sscanf(existingDimStr, "%d", &existingDim); err != nil {
		return fmt.Errorf("invalid dimension in metadata: %s", existingDimStr)
	}

	if existingDim != db.dimension {
		return fmt.Errorf("dimension mismatch: database has %d, requested %d", existingDim, db.dimension)
	}

	return nil
}

// SetMetadata stores a key-value pair in the metadata table
func (db *DB) SetMetadata(ctx context.Context, key, value string) error {
	query := `
		INSERT INTO metadata (key, value)
		VALUES ($1, $2)
		ON CONFLICT (key)
		DO UPDATE SET value = EXCLUDED.value`

	_, err := db.ExecContext(ctx, query, key, value)
	if err != nil {
		return fmt.Errorf("failed to set metadata %s: %w", key, err)
	}

	return nil
}

// GetMetadata retrieves a value from the metadata table
func (db *DB) GetMetadata(ctx context.Context, key string) (string, error) {
	var value string
	err := db.QueryRowContext(ctx, "SELECT value FROM metadata WHERE key = $1", key).Scan(&value)
	if err != nil {
		return "", fmt.Errorf("failed to get metadata %s: %w", key, err)
	}
	return value, nil
}

// Dimension returns the embedding dimension
func (db *DB) Dimension() int {
	return db.dimension
}

// GetDimension retrieves the dimension from an existing PostgreSQL database
func GetDimension(ctx context.Context, connectionString string) (int, error) {
	sqlDB, err := sql.Open("postgres", connectionString)
	if err != nil {
		return 0, fmt.Errorf("failed to open database: %w", err)
	}
	defer sqlDB.Close()

	var dimStr string
	err = sqlDB.QueryRowContext(ctx, "SELECT value FROM metadata WHERE key = 'dimension'").Scan(&dimStr)
	if err != nil {
		return 0, fmt.Errorf("failed to query dimension: %w", err)
	}

	var dim int
	if _, err := fmt.Sscanf(dimStr, "%d", &dim); err != nil {
		return 0, fmt.Errorf("invalid dimension value: %s", dimStr)
	}

	return dim, nil
}

// Compatibility aliases for gradual migration
type ProjectDB = DB
type PostgresDB = DB
type PostgresConfig struct {
	Host     string
	Port     int
	User     string
	Password string
	DBName   string
	SSLMode  string
	MaxConns int
	MaxIdle  int
}

// NewPostgresDB creates a DB from PostgresConfig (compatibility wrapper)
func NewPostgresDB(ctx context.Context, cfg PostgresConfig, dimension int) (*DB, error) {
	connStr := fmt.Sprintf(
		"postgresql://%s:%s@%s:%d/%s?sslmode=%s",
		cfg.User, cfg.Password, cfg.Host, cfg.Port, cfg.DBName, cfg.SSLMode,
	)
	return NewDB(ctx, connStr, dimension)
}

// OpenProjectDB opens a database (compatibility wrapper for DuckDB -> PostgreSQL migration)
// NOTE: This is deprecated - use NewDB with a connection string instead
func OpenProjectDB(connStringOrPath string, dimension int) (*DB, error) {
	// This is a temporary compatibility layer
	// In the new system, we expect a connection string
	return NewDB(context.Background(), connStringOrPath, dimension)
}

// GetDimensionFromDB is a compatibility wrapper
func GetDimensionFromDB(dbPathOrConnString string) (int, error) {
	return GetDimension(context.Background(), dbPathOrConnString)
}

// GetDimensionFromPostgres is a compatibility wrapper
func GetDimensionFromPostgres(ctx context.Context, cfg PostgresConfig) (int, error) {
	connStr := fmt.Sprintf(
		"postgresql://%s:%s@%s:%d/%s?sslmode=%s",
		cfg.User, cfg.Password, cfg.Host, cfg.Port, cfg.DBName, cfg.SSLMode,
	)
	return GetDimension(ctx, connStr)
}
