package db

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	_ "github.com/lib/pq"
)

// PostgresDB wraps a PostgreSQL connection with pgvector support
type PostgresDB struct {
	db        *sql.DB
	dimension int
	dbName    string
}

// PostgresConfig holds PostgreSQL connection configuration
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

// NewPostgresDB creates a new PostgreSQL connection with pgvector support
func NewPostgresDB(ctx context.Context, cfg PostgresConfig, dimension int) (*PostgresDB, error) {
	if dimension <= 0 {
		return nil, fmt.Errorf("dimension must be positive, got %d", dimension)
	}

	// Build connection string
	dsn := fmt.Sprintf(
		"host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
		cfg.Host, cfg.Port, cfg.User, cfg.Password, cfg.DBName, cfg.SSLMode,
	)

	// Open database connection
	db, err := sql.Open("postgres", dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// Configure connection pool
	db.SetMaxOpenConns(cfg.MaxConns)
	db.SetMaxIdleConns(cfg.MaxIdle)
	db.SetConnMaxLifetime(time.Hour)
	db.SetConnMaxIdleTime(15 * time.Minute)

	// Verify connection
	if err := db.PingContext(ctx); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	pdb := &PostgresDB{
		db:        db,
		dimension: dimension,
		dbName:    cfg.DBName,
	}

	// Initialize schema and extensions
	if err := pdb.initSchema(ctx); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to initialize schema: %w", err)
	}

	// Check or set dimension metadata
	if err := pdb.ensureDimension(ctx); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to ensure dimension: %w", err)
	}

	return pdb, nil
}

// initSchema creates necessary extensions and tables
func (p *PostgresDB) initSchema(ctx context.Context) error {
	// Enable pgvector extension
	if _, err := p.db.ExecContext(ctx, "CREATE EXTENSION IF NOT EXISTS vector"); err != nil {
		return fmt.Errorf("failed to create vector extension: %w", err)
	}

	// Create tables using schema builder
	for _, stmt := range buildPostgresSchema(p.dimension) {
		if _, err := p.db.ExecContext(ctx, stmt); err != nil {
			return fmt.Errorf("failed to execute schema statement: %w", err)
		}
	}

	return nil
}

// ensureDimension checks existing dimension or sets it for new database
func (p *PostgresDB) ensureDimension(ctx context.Context) error {
	var existingDimStr string
	err := p.db.QueryRowContext(ctx,
		"SELECT value FROM metadata WHERE key = 'dimension'").Scan(&existingDimStr)

	if err == sql.ErrNoRows {
		// New database, set dimension
		return p.SetMetadata(ctx, "dimension", fmt.Sprintf("%d", p.dimension))
	} else if err != nil {
		return fmt.Errorf("failed to query dimension: %w", err)
	}

	// Parse existing dimension
	var existingDim int
	if _, err := fmt.Sscanf(existingDimStr, "%d", &existingDim); err != nil {
		return fmt.Errorf("invalid dimension in metadata: %s", existingDimStr)
	}

	if existingDim != p.dimension {
		return fmt.Errorf("dimension mismatch: database has %d, requested %d", existingDim, p.dimension)
	}

	return nil
}

// SetMetadata stores a key-value pair in the metadata table
func (p *PostgresDB) SetMetadata(ctx context.Context, key, value string) error {
	query := `
		INSERT INTO metadata (key, value)
		VALUES ($1, $2)
		ON CONFLICT (key)
		DO UPDATE SET value = EXCLUDED.value`

	_, err := p.db.ExecContext(ctx, query, key, value)
	if err != nil {
		return fmt.Errorf("failed to set metadata %s: %w", key, err)
	}

	return nil
}

// GetMetadata retrieves a value from the metadata table
func (p *PostgresDB) GetMetadata(ctx context.Context, key string) (string, error) {
	var value string
	err := p.db.QueryRowContext(ctx, "SELECT value FROM metadata WHERE key = $1", key).Scan(&value)
	if err != nil {
		return "", fmt.Errorf("failed to get metadata %s: %w", key, err)
	}
	return value, nil
}

// Close closes the database connection
func (p *PostgresDB) Close() error {
	return p.db.Close()
}

// Conn returns the underlying sql.DB connection
func (p *PostgresDB) Conn() *sql.DB {
	return p.db
}

// Dimension returns the embedding dimension
func (p *PostgresDB) Dimension() int {
	return p.dimension
}

// DBName returns the database name
func (p *PostgresDB) DBName() string {
	return p.dbName
}

// GetDimensionFromPostgres retrieves the dimension from an existing PostgreSQL database
func GetDimensionFromPostgres(ctx context.Context, cfg PostgresConfig) (int, error) {
	dsn := fmt.Sprintf(
		"host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
		cfg.Host, cfg.Port, cfg.User, cfg.Password, cfg.DBName, cfg.SSLMode,
	)

	db, err := sql.Open("postgres", dsn)
	if err != nil {
		return 0, fmt.Errorf("failed to open database: %w", err)
	}
	defer db.Close()

	var dimStr string
	err = db.QueryRowContext(ctx, "SELECT value FROM metadata WHERE key = 'dimension'").Scan(&dimStr)
	if err != nil {
		return 0, fmt.Errorf("failed to query dimension: %w", err)
	}

	var dim int
	if _, err := fmt.Sscanf(dimStr, "%d", &dim); err != nil {
		return 0, fmt.Errorf("invalid dimension value: %s", dimStr)
	}

	return dim, nil
}
