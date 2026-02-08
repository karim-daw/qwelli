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
	EmbeddingProvider string `yaml:"embedding_provider"` // "voyage" (default)
	APIKey            string `yaml:"api_key"`
	Model             string `yaml:"model"`
	Endpoint          string `yaml:"endpoint"`

	// Multimodal settings
	EnableMultimodal bool   `yaml:"enable_multimodal"` // Enable multimodal embeddings (images)
	ImageQuality     string `yaml:"image_quality"`     // "low", "medium", "high" (default: "medium")

	// Reranker settings
	EnableReranker   bool   `yaml:"enable_reranker"`   // Enable reranking of search results
	RerankProvider   string `yaml:"rerank_provider"`   // "voyage" (default)
	RerankModel      string `yaml:"rerank_model"`      // Reranker model to use
	RerankEndpoint   string `yaml:"rerank_endpoint"`   // Custom reranker endpoint (optional)

	// Local storage settings
	IndexDir string `yaml:"index_dir"` // Where to store .db files
}

// DefaultConfig returns the default configuration
func DefaultConfig() *Config {
	homeDir, _ := os.UserHomeDir()

	return &Config{
		EmbeddingProvider: "voyage",
		APIKey:            os.Getenv("VOYAGE_API_KEY"),
		Model:             os.Getenv("VOYAGE_MODEL"),
		Endpoint:          os.Getenv("VOYAGE_EMBEDDING_ENDPOINT"),
		EnableMultimodal:  true, // Default to true for Voyage
		ImageQuality:      "medium",
		EnableReranker:    true, // Default to true (enabled by default)
		RerankProvider:    "voyage",
		RerankModel:       os.Getenv("VOYAGE_RERANK_MODEL"),
		RerankEndpoint:    os.Getenv("VOYAGE_RERANK_ENDPOINT"),
		IndexDir:          filepath.Join(homeDir, ".qwelli", "indexes"),
	}
}

// ConfigPath returns the path to the config file
func ConfigPath() string {
	homeDir, _ := os.UserHomeDir()
	return filepath.Join(homeDir, ".qwelli", "config.yaml")
}

// Load loads the configuration from disk and applies environment variable overrides.
// If path is provided, loads from that path; otherwise uses the default config path.
func Load(path ...string) (*Config, error) {
	configPath := ConfigPath()
	if len(path) > 0 && path[0] != "" {
		configPath = path[0]
	}

	// If config doesn't exist, return error
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

	// Apply environment variable overrides (env vars take precedence over config file)
	if apiKey := os.Getenv("VOYAGE_API_KEY"); apiKey != "" {
		cfg.APIKey = apiKey
	}
	if model := os.Getenv("VOYAGE_MODEL"); model != "" {
		cfg.Model = model
	}
	if endpoint := os.Getenv("VOYAGE_EMBEDDING_ENDPOINT"); endpoint != "" {
		cfg.Endpoint = endpoint
	}
	if rerankModel := os.Getenv("VOYAGE_RERANK_MODEL"); rerankModel != "" {
		cfg.RerankModel = rerankModel
	}
	if rerankEndpoint := os.Getenv("VOYAGE_RERANK_ENDPOINT"); rerankEndpoint != "" {
		cfg.RerankEndpoint = rerankEndpoint
	}
	if val := os.Getenv("ENABLE_RERANKER"); val != "" {
		cfg.EnableReranker = val == "true" || val == "1" || val == "yes"
	}

	return &cfg, nil
}

// Save saves the configuration to disk.
// If path is provided, saves to that path; otherwise uses the default config path.
func (c *Config) Save(path ...string) error {
	configPath := ConfigPath()
	if len(path) > 0 && path[0] != "" {
		configPath = path[0]
	}

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
