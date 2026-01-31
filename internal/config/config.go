package config

import (
	"fmt"
	"os"
	"strconv"
)

// Config holds the application configuration
type Config struct {
	DatabaseURL    string // PostgreSQL connection string
	VoyageAPIKey   string // Voyage AI API key
	VoyageModel    string // Embedding model (default: voyage-multimodal-3)
	Port           int    // Server port (default: 8080)
	EnableReranker bool   // Enable reranking of search results (default: true)
}

// Load loads configuration from environment variables
func Load() (*Config, error) {
	cfg := &Config{
		DatabaseURL:    os.Getenv("DATABASE_URL"),
		VoyageAPIKey:   os.Getenv("VOYAGE_API_KEY"),
		VoyageModel:    getEnv("VOYAGE_MODEL", "voyage-multimodal-3"),
		Port:           getEnvAsInt("PORT", 8080),
		EnableReranker: getEnvAsBool("ENABLE_RERANKER", true),
	}

	// Validate required fields
	if cfg.DatabaseURL == "" {
		return nil, fmt.Errorf("DATABASE_URL environment variable is required")
	}
	if cfg.VoyageAPIKey == "" {
		return nil, fmt.Errorf("VOYAGE_API_KEY environment variable is required")
	}

	return cfg, nil
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
