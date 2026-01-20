package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"

	"gopkg.in/yaml.v3"
)

// Config holds the application configuration
type Config struct {
	// Database settings
	Database DatabaseConfig `yaml:"database"`

	// Embedding provider settings
	EmbeddingProvider string `yaml:"embedding_provider"` // "voyage" (default)
	APIKey            string `yaml:"api_key"`
	Model             string `yaml:"model"`
	Endpoint          string `yaml:"endpoint"`

	// Multimodal settings
	EnableMultimodal bool   `yaml:"enable_multimodal"` // Enable multimodal embeddings (images)
	ImageQuality     string `yaml:"image_quality"`     // "low", "medium", "high" (default: "medium")

	// Reranker settings
	EnableReranker bool   `yaml:"enable_reranker"` // Enable reranking of search results
	RerankProvider string `yaml:"rerank_provider"` // "voyage" (default)
	RerankModel    string `yaml:"rerank_model"`    // Reranker model to use
	RerankEndpoint string `yaml:"rerank_endpoint"` // Custom reranker endpoint (optional)

	// Server settings
	ServerHost string `yaml:"server_host"` // Server bind host (default: "0.0.0.0")
	ServerPort int    `yaml:"server_port"` // Server port (default: 8080)

	// Local storage settings (deprecated in favor of PostgreSQL)
	IndexDir string `yaml:"index_dir"` // Where to store .db files (for local mode)
}

// DatabaseConfig holds database connection configuration
type DatabaseConfig struct {
	Type     string `yaml:"type"`     // "postgres" or "duckdb" (default: "postgres")
	Host     string `yaml:"host"`     // PostgreSQL host
	Port     int    `yaml:"port"`     // PostgreSQL port
	User     string `yaml:"user"`     // PostgreSQL user
	Password string `yaml:"password"` // PostgreSQL password
	DBName   string `yaml:"dbname"`   // PostgreSQL database name
	SSLMode  string `yaml:"sslmode"`  // PostgreSQL SSL mode
	MaxConns int    `yaml:"max_conns"` // Max open connections
	MaxIdle  int    `yaml:"max_idle"`  // Max idle connections
}

// DefaultConfig returns the default configuration
func DefaultConfig() *Config {
	homeDir, _ := os.UserHomeDir()

	return &Config{
		Database: DatabaseConfig{
			Type:     getEnvOrDefault("DB_TYPE", "postgres"),
			Host:     getEnvOrDefault("DB_HOST", "localhost"),
			Port:     getEnvAsInt("DB_PORT", 5432),
			User:     getEnvOrDefault("DB_USER", "qwelli"),
			Password: os.Getenv("DB_PASSWORD"),
			DBName:   getEnvOrDefault("DB_NAME", "qwelli"),
			SSLMode:  getEnvOrDefault("DB_SSLMODE", "disable"),
			MaxConns: getEnvAsInt("DB_MAX_CONNS", 25),
			MaxIdle:  getEnvAsInt("DB_MAX_IDLE", 5),
		},
		EmbeddingProvider: "voyage",
		APIKey:            os.Getenv("QWELLI_EMBEDDING_KEY"),
		Model:             getEnvOrDefault("QWELLI_EMBEDDING_MODEL", "voyage-multimodal-3"),
		Endpoint:          getEnvOrDefault("QWELLI_EMBEDDING_ENDPOINT", "https://api.voyageai.com/v1/multimodalembeddings"),
		EnableMultimodal:  getEnvAsBool("ENABLE_MULTIMODAL", true),
		ImageQuality:      getEnvOrDefault("IMAGE_QUALITY", "medium"),
		EnableReranker:    getEnvAsBool("ENABLE_RERANKER", true),
		RerankProvider:    "voyage",
		RerankModel:       os.Getenv("QWELLI_RERANK_MODEL"),
		RerankEndpoint:    os.Getenv("QWELLI_RERANK_ENDPOINT"),
		ServerHost:        getEnvOrDefault("SERVER_HOST", "0.0.0.0"),
		ServerPort:        getEnvAsInt("SERVER_PORT", 8080),
		IndexDir:          filepath.Join(homeDir, ".qwelli", "indexes"),
	}
}

// ConfigPath returns the path to the config file
func ConfigPath() string {
	homeDir, _ := os.UserHomeDir()
	return filepath.Join(homeDir, ".qwelli", "config.yaml")
}

// Load loads the configuration from disk or environment
func Load() (*Config, error) {
	cfg := DefaultConfig()
	configPath := ConfigPath()

	// Try to load from file if it exists
	if _, err := os.Stat(configPath); err == nil {
		data, err := os.ReadFile(configPath)
		if err != nil {
			return nil, fmt.Errorf("failed to read config: %w", err)
		}

		if err := yaml.Unmarshal(data, cfg); err != nil {
			return nil, fmt.Errorf("failed to parse config: %w", err)
		}
	}

	// Environment variables always override file config
	cfg.overrideFromEnv()

	return cfg, nil
}

// overrideFromEnv overrides configuration with environment variables
func (c *Config) overrideFromEnv() {
	// Database configuration
	if v := os.Getenv("DB_TYPE"); v != "" {
		c.Database.Type = v
	}
	if v := os.Getenv("DB_HOST"); v != "" {
		c.Database.Host = v
	}
	if v := os.Getenv("DB_PORT"); v != "" {
		if port, err := strconv.Atoi(v); err == nil {
			c.Database.Port = port
		}
	}
	if v := os.Getenv("DB_USER"); v != "" {
		c.Database.User = v
	}
	if v := os.Getenv("DB_PASSWORD"); v != "" {
		c.Database.Password = v
	}
	if v := os.Getenv("DB_NAME"); v != "" {
		c.Database.DBName = v
	}
	if v := os.Getenv("DB_SSLMODE"); v != "" {
		c.Database.SSLMode = v
	}
	if v := os.Getenv("DB_MAX_CONNS"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			c.Database.MaxConns = n
		}
	}
	if v := os.Getenv("DB_MAX_IDLE"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			c.Database.MaxIdle = n
		}
	}

	// Embedding configuration
	if v := os.Getenv("QWELLI_EMBEDDING_KEY"); v != "" {
		c.APIKey = v
	}
	if v := os.Getenv("QWELLI_EMBEDDING_MODEL"); v != "" {
		c.Model = v
	}
	if v := os.Getenv("QWELLI_EMBEDDING_ENDPOINT"); v != "" {
		c.Endpoint = v
	}

	// Multimodal settings
	if v := os.Getenv("ENABLE_MULTIMODAL"); v != "" {
		c.EnableMultimodal = v == "true" || v == "1"
	}
	if v := os.Getenv("IMAGE_QUALITY"); v != "" {
		c.ImageQuality = v
	}

	// Reranker settings
	if v := os.Getenv("ENABLE_RERANKER"); v != "" {
		c.EnableReranker = v == "true" || v == "1"
	}
	if v := os.Getenv("QWELLI_RERANK_MODEL"); v != "" {
		c.RerankModel = v
	}
	if v := os.Getenv("QWELLI_RERANK_ENDPOINT"); v != "" {
		c.RerankEndpoint = v
	}

	// Server settings
	if v := os.Getenv("SERVER_HOST"); v != "" {
		c.ServerHost = v
	}
	if v := os.Getenv("SERVER_PORT"); v != "" {
		if port, err := strconv.Atoi(v); err == nil {
			c.ServerPort = port
		}
	}
}

// Save saves the configuration to disk
func (c *Config) Save() error {
	configPath := ConfigPath()

	// Ensure directory exists
	if err := os.MkdirAll(filepath.Dir(configPath), 0755); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	data, err := yaml.Marshal(c)
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	if err := os.WriteFile(configPath, data, 0600); err != nil {
		return fmt.Errorf("failed to write config: %w", err)
	}

	return nil
}

// EnsureIndexDir creates the index directory if it doesn't exist (for local DuckDB mode)
func (c *Config) EnsureIndexDir() error {
	return os.MkdirAll(c.IndexDir, 0755)
}

// Helper functions for environment variable parsing

func getEnvOrDefault(key, defaultValue string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return defaultValue
}

func getEnvAsInt(key string, defaultValue int) int {
	if v := os.Getenv(key); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			return n
		}
	}
	return defaultValue
}

func getEnvAsBool(key string, defaultValue bool) bool {
	if v := os.Getenv(key); v != "" {
		return v == "true" || v == "1"
	}
	return defaultValue
}
