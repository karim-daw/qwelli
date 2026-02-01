package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"

	"github.com/joho/godotenv"
)

// Config holds the application configuration
type Config struct {
	DatabaseURL             string // PostgreSQL connection string
	VoyageAPIKey            string // Voyage AI API key
	VoyageModel             string // Embedding model (default: voyage-multimodal-3.5)
	VoyageEmbeddingEndpoint string // Embedding API endpoint (default: Voyage AI)
	VoyageRerankModel       string // Rerank model (default: rerank-2)
	VoyageRerankEndpoint    string // Rerank API endpoint (default: Voyage AI)
	Port                    int    // Server port (default: 8080)
	EnableReranker          bool   // Enable reranking of search results (default: true)
}

// Load loads configuration from environment variables
// Automatically loads .env file if it exists in current directory or parent directories
func Load() (*Config, error) {
	// Try to load .env file (silently ignore if not found)
	loadDotEnv()

	cfg := &Config{
		DatabaseURL:             os.Getenv("DATABASE_URL"),
		VoyageAPIKey:            os.Getenv("VOYAGE_API_KEY"),
		VoyageModel:             getEnv("VOYAGE_MODEL", "voyage-multimodal-3.5"),
		VoyageEmbeddingEndpoint: getEnv("VOYAGE_EMBEDDING_ENDPOINT", "https://api.voyageai.com/v1/multimodalembeddings"),
		VoyageRerankModel:       getEnv("VOYAGE_RERANK_MODEL", "rerank-2"),
		VoyageRerankEndpoint:    getEnv("VOYAGE_RERANK_ENDPOINT", "https://api.voyageai.com/v1/rerank"),
		Port:                    getEnvAsInt("PORT", 8080),
		EnableReranker:          getEnvAsBool("ENABLE_RERANKER", true),
	}

	// Validate required fields
	if cfg.DatabaseURL == "" {
		return nil, fmt.Errorf("DATABASE_URL environment variable is required\n\nTip: Create a .env file with:\n  DATABASE_URL=postgresql://user:pass@host:5432/dbname\n  VOYAGE_API_KEY=your_key")
	}
	if cfg.VoyageAPIKey == "" {
		return nil, fmt.Errorf("VOYAGE_API_KEY environment variable is required\n\nTip: Create a .env file with:\n  DATABASE_URL=postgresql://user:pass@host:5432/dbname\n  VOYAGE_API_KEY=your_key")
	}

	return cfg, nil
}

// loadDotEnv tries to load .env file from current directory or parent directories
// It silently ignores if the file doesn't exist
func loadDotEnv() {
	// Try current directory first
	if err := godotenv.Load(); err == nil {
		return
	}

	// Try finding .env in parent directories (up to 5 levels)
	cwd, err := os.Getwd()
	if err != nil {
		return
	}

	dir := cwd
	for i := 0; i < 5; i++ {
		envPath := filepath.Join(dir, ".env")
		if _, err := os.Stat(envPath); err == nil {
			_ = godotenv.Load(envPath)
			return
		}

		parent := filepath.Dir(dir)
		if parent == dir {
			break // reached root
		}
		dir = parent
	}
}

// Helper functions

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getEnvAsInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if intVal, err := strconv.Atoi(value); err == nil {
			return intVal
		}
	}
	return defaultValue
}

func getEnvAsBool(key string, defaultValue bool) bool {
	if value := os.Getenv(key); value != "" {
		return value == "true" || value == "1"
	}
	return defaultValue
}
