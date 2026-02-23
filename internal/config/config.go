package config

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"

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

	// Parallel processing settings
	EnableParallel          bool `yaml:"enable_parallel"`           // Enable parallel file processing (default: true)
	ParallelWorkers         int  `yaml:"parallel_workers"`          // Number of file processing workers (0 = auto: ~90% of CPU cores)
	EnableParallelPDF       bool `yaml:"enable_parallel_pdf"`       // Enable parallel PDF page processing (default: true)
	ParallelPDFWorkers      int  `yaml:"parallel_pdf_workers"`      // Number of PDF page workers (0 = auto: ~90% of CPU cores)
	MaxConcurrentEmbeddings int  `yaml:"max_concurrent_embeddings"` // Max parallel embedding API calls (default: 5)

	// Agent / Azure AI Foundry settings (Claude via Foundry)
	FoundryEndpoint string `yaml:"foundry_endpoint"` // e.g. https://resource.services.ai.azure.com/anthropic
	FoundryAPIKey   string `yaml:"foundry_api_key"`
	FoundryModel    string `yaml:"foundry_model"` // default: claude-sonnet-4-6

	// Local storage settings
	IndexDir string `yaml:"index_dir"` // Where to store .db files
}

// DefaultConfig returns the default configuration
func DefaultConfig() *Config {
	homeDir, _ := os.UserHomeDir()

	return &Config{
		EmbeddingProvider:  "voyage",
		APIKey:             os.Getenv("VOYAGE_API_KEY"),
		Model:              os.Getenv("VOYAGE_MODEL"),
		Endpoint:           os.Getenv("VOYAGE_EMBEDDING_ENDPOINT"),
		EnableMultimodal:   true, // Default to true for Voyage
		ImageQuality:       "medium",
		EnableReranker:     true, // Default to true (enabled by default)
		RerankProvider:     "voyage",
		RerankModel:        os.Getenv("VOYAGE_RERANK_MODEL"),
		RerankEndpoint:     os.Getenv("VOYAGE_RERANK_ENDPOINT"),
		EnableParallel:          true, // Parallel file processing enabled by default
		ParallelWorkers:         0,    // 0 = auto-detect (~90% of CPU cores)
		EnableParallelPDF:       true, // Parallel PDF page processing enabled by default
		ParallelPDFWorkers:      0,    // 0 = auto-detect (~90% of CPU cores)
		MaxConcurrentEmbeddings: 5,    // Max parallel embedding API calls
		FoundryModel:           "claude-sonnet-4-6",
		IndexDir:               filepath.Join(homeDir, ".qwelli", "indexes"),
	}
}

// ConfigPath returns the path to the config file
func ConfigPath() string {
	homeDir, _ := os.UserHomeDir()
	return filepath.Join(homeDir, ".qwelli", "config.yaml")
}

// Exists returns true if the config file exists on disk
func Exists() bool {
	_, err := os.Stat(ConfigPath())
	return err == nil
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

	// Start from defaults so missing fields get sensible values
	// (e.g., existing configs without parallel settings still get parallelism enabled)
	cfg := *DefaultConfig()
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
	if val := os.Getenv("MAX_CONCURRENT_EMBEDDINGS"); val != "" {
		var n int
		if _, err := fmt.Sscanf(val, "%d", &n); err == nil && n > 0 {
			cfg.MaxConcurrentEmbeddings = n
		}
	}

	// Agent / Azure AI Foundry overrides
	if v := os.Getenv("FOUNDRY_ENDPOINT"); v != "" {
		cfg.FoundryEndpoint = v
	}
	if v := os.Getenv("FOUNDRY_API_KEY"); v != "" {
		cfg.FoundryAPIKey = v
	}
	if v := os.Getenv("FOUNDRY_MODEL"); v != "" {
		cfg.FoundryModel = v
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

// DefaultWorkerCount returns a worker count targeting ~90% CPU utilization.
// It uses max(floor(NumCPU * 0.9), 1) so we always have at least 1 worker
// and leave a sliver of headroom for the OS, Go runtime GC, and the main goroutine.
func DefaultWorkerCount() int {
	cpus := runtime.NumCPU()
	workers := int(float64(cpus) * 0.9)
	if workers < 1 {
		workers = 1
	}
	return workers
}
