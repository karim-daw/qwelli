package config

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// Config holds the application configuration
type Config struct {
	// Embedding provider settings
	EmbeddingProvider string `yaml:"embedding_provider"` // "openai", etc.
	APIKey            string `yaml:"api_key"`
	Model             string `yaml:"model"`
	Endpoint          string `yaml:"endpoint"`

	// Local storage settings
	IndexDir string `yaml:"index_dir"` // Where to store .db files
}

// DefaultConfig returns the default configuration
func DefaultConfig() *Config {
	homeDir, _ := os.UserHomeDir()

	return &Config{
		EmbeddingProvider: "voyagerAI",
		APIKey:            os.Getenv("QWELLI_EMBEDDING_KEY"),
		Model:             os.Getenv("QWELLI_EMBEDDING_MODEL"),
		Endpoint:          os.Getenv("QWELLI_EMBEDDING_ENDPOINT"),
		IndexDir:          filepath.Join(homeDir, ".qwelli", "indexes"),
	}
}

// ConfigPath returns the path to the config file
func ConfigPath() string {
	homeDir, _ := os.UserHomeDir()
	return filepath.Join(homeDir, ".qwelli", "config.yaml")
}

// Load loads the configuration from disk
func Load() (*Config, error) {
	configPath := ConfigPath()

	// If config doesn't exist, return default
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		return nil, fmt.Errorf("config not found. Run 'qwelli init' first")
	}

	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read config: %w", err)
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("failed to parse config: %w", err)
	}

	return &cfg, nil
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

// EnsureIndexDir creates the index directory if it doesn't exist
func (c *Config) EnsureIndexDir() error {
	return os.MkdirAll(c.IndexDir, 0755)
}
