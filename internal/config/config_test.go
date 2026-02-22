package config

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"gopkg.in/yaml.v3"
)

func TestDefaultConfig_Values(t *testing.T) {
	// Clear environment variables to test defaults
	oldAPIKey := os.Getenv("VOYAGE_API_KEY")
	oldModel := os.Getenv("VOYAGE_MODEL")
	oldEndpoint := os.Getenv("VOYAGE_EMBEDDING_ENDPOINT")
	defer func() {
		os.Setenv("VOYAGE_API_KEY", oldAPIKey)
		os.Setenv("VOYAGE_MODEL", oldModel)
		os.Setenv("VOYAGE_EMBEDDING_ENDPOINT", oldEndpoint)
	}()

	os.Unsetenv("VOYAGE_API_KEY")
	os.Unsetenv("VOYAGE_MODEL")
	os.Unsetenv("VOYAGE_EMBEDDING_ENDPOINT")

	cfg := DefaultConfig()

	if cfg.EmbeddingProvider != "voyage" {
		t.Errorf("Expected embedding provider 'voyage', got '%s'", cfg.EmbeddingProvider)
	}

	if cfg.EnableMultimodal != true {
		t.Error("Expected multimodal to be enabled by default")
	}

	if cfg.ImageQuality != "medium" {
		t.Errorf("Expected image quality 'medium', got '%s'", cfg.ImageQuality)
	}

	if cfg.EnableReranker != true {
		t.Error("Expected reranker to be enabled by default")
	}

	if cfg.RerankProvider != "voyage" {
		t.Errorf("Expected rerank provider 'voyage', got '%s'", cfg.RerankProvider)
	}

	homeDir, _ := os.UserHomeDir()
	expectedIndexDir := filepath.Join(homeDir, ".qwelli", "indexes")
	if cfg.IndexDir != expectedIndexDir {
		t.Errorf("Expected index dir '%s', got '%s'", expectedIndexDir, cfg.IndexDir)
	}
}

func TestDefaultConfig_EnvVars(t *testing.T) {
	// Set environment variables
	testAPIKey := "test-api-key-123"
	testModel := "test-model"
	testEndpoint := "http://test-endpoint.com"

	oldAPIKey := os.Getenv("VOYAGE_API_KEY")
	oldModel := os.Getenv("VOYAGE_MODEL")
	oldEndpoint := os.Getenv("VOYAGE_EMBEDDING_ENDPOINT")
	defer func() {
		os.Setenv("VOYAGE_API_KEY", oldAPIKey)
		os.Setenv("VOYAGE_MODEL", oldModel)
		os.Setenv("VOYAGE_EMBEDDING_ENDPOINT", oldEndpoint)
	}()

	os.Setenv("VOYAGE_API_KEY", testAPIKey)
	os.Setenv("VOYAGE_MODEL", testModel)
	os.Setenv("VOYAGE_EMBEDDING_ENDPOINT", testEndpoint)

	cfg := DefaultConfig()

	if cfg.APIKey != testAPIKey {
		t.Errorf("Expected API key '%s', got '%s'", testAPIKey, cfg.APIKey)
	}

	if cfg.Model != testModel {
		t.Errorf("Expected model '%s', got '%s'", testModel, cfg.Model)
	}

	if cfg.Endpoint != testEndpoint {
		t.Errorf("Expected endpoint '%s', got '%s'", testEndpoint, cfg.Endpoint)
	}
}

func TestConfigPath_ReturnsCorrectPath(t *testing.T) {
	path := ConfigPath()

	homeDir, _ := os.UserHomeDir()
	expectedPath := filepath.Join(homeDir, ".qwelli", "config.yaml")

	if path != expectedPath {
		t.Errorf("Expected config path '%s', got '%s'", expectedPath, path)
	}
}

func TestSave_CreatesFile(t *testing.T) {
	tempDir := t.TempDir()
	testConfigPath := filepath.Join(tempDir, "config.yaml")

	cfg := &Config{
		EmbeddingProvider: "voyage",
		APIKey:            "test-key",
		Model:             "voyage-3",
		Endpoint:          "http://test.com",
		EnableMultimodal:  true,
		ImageQuality:      "medium",
		EnableReranker:    true,
		RerankProvider:    "voyage",
		RerankModel:       "rerank-2",
		RerankEndpoint:    "http://test-rerank.com",
		IndexDir:          tempDir,
	}

	err := cfg.Save(testConfigPath)
	if err != nil {
		t.Fatalf("Failed to save config: %v", err)
	}

	if _, err := os.Stat(testConfigPath); os.IsNotExist(err) {
		t.Fatal("Config file was not created")
	}

	info, err := os.Stat(testConfigPath)
	if err != nil {
		t.Fatalf("Failed to stat config file: %v", err)
	}

	if runtime.GOOS != "windows" {
		mode := info.Mode()
		if mode.Perm() != 0600 {
			t.Errorf("Expected file permissions 0600, got %o", mode.Perm())
		}
	}
}

func TestSave_ValidYAML(t *testing.T) {
	tempDir := t.TempDir()
	testConfigPath := filepath.Join(tempDir, "config.yaml")

	originalCfg := &Config{
		EmbeddingProvider: "voyage",
		APIKey:            "test-key-abc",
		Model:             "voyage-multimodal-3",
		Endpoint:          "http://api.test.com",
		EnableMultimodal:  true,
		ImageQuality:      "high",
		EnableReranker:    false,
		RerankProvider:    "voyage",
		RerankModel:       "rerank-2",
		RerankEndpoint:    "",
		IndexDir:          "/path/to/indexes",
	}

	err := originalCfg.Save(testConfigPath)
	if err != nil {
		t.Fatalf("Failed to save config: %v", err)
	}

	data, err := os.ReadFile(testConfigPath)
	if err != nil {
		t.Fatalf("Failed to read config file: %v", err)
	}

	var loadedCfg Config
	err = yaml.Unmarshal(data, &loadedCfg)
	if err != nil {
		t.Fatalf("Failed to unmarshal saved config: %v", err)
	}

	if loadedCfg.APIKey != originalCfg.APIKey {
		t.Errorf("API key mismatch: expected '%s', got '%s'", originalCfg.APIKey, loadedCfg.APIKey)
	}

	if loadedCfg.Model != originalCfg.Model {
		t.Errorf("Model mismatch: expected '%s', got '%s'", originalCfg.Model, loadedCfg.Model)
	}

	if loadedCfg.EnableMultimodal != originalCfg.EnableMultimodal {
		t.Error("EnableMultimodal mismatch")
	}
}

func TestLoad_MissingConfig(t *testing.T) {
	tempDir := t.TempDir()
	testConfigPath := filepath.Join(tempDir, "nonexistent", "config.yaml")

	_, err := Load(testConfigPath)
	if err == nil {
		t.Fatal("Expected error when loading missing config, got nil")
	}

	expectedSubstring := "config not found"
	if !contains(err.Error(), expectedSubstring) {
		t.Errorf("Expected error to contain '%s', got: %v", expectedSubstring, err)
	}
}

func TestLoad_InvalidYAML(t *testing.T) {
	tempDir := t.TempDir()
	testConfigPath := filepath.Join(tempDir, "config.yaml")

	invalidYAML := "this is not: valid: yaml: content:"
	err := os.WriteFile(testConfigPath, []byte(invalidYAML), 0600)
	if err != nil {
		t.Fatalf("Failed to write test config: %v", err)
	}

	_, err = Load(testConfigPath)
	if err == nil {
		t.Fatal("Expected error when loading invalid YAML, got nil")
	}

	expectedSubstring := "failed to parse config"
	if !contains(err.Error(), expectedSubstring) {
		t.Errorf("Expected error to contain '%s', got: %v", expectedSubstring, err)
	}
}

func TestLoad_EnvOverrides(t *testing.T) {
	tempDir := t.TempDir()
	testConfigPath := filepath.Join(tempDir, "config.yaml")

	initialCfg := &Config{
		EmbeddingProvider: "voyage",
		APIKey:            "file-api-key",
		Model:             "file-model",
		Endpoint:          "http://file-endpoint.com",
		EnableMultimodal:  true,
		ImageQuality:      "medium",
		EnableReranker:    false,
		RerankProvider:    "voyage",
		RerankModel:       "file-rerank",
		RerankEndpoint:    "http://file-rerank.com",
		IndexDir:          tempDir,
	}

	data, _ := yaml.Marshal(initialCfg)
	err := os.WriteFile(testConfigPath, data, 0600)
	if err != nil {
		t.Fatalf("Failed to write test config: %v", err)
	}

	oldAPIKey := os.Getenv("VOYAGE_API_KEY")
	oldModel := os.Getenv("VOYAGE_MODEL")
	oldEndpoint := os.Getenv("VOYAGE_EMBEDDING_ENDPOINT")
	oldRerankModel := os.Getenv("VOYAGE_RERANK_MODEL")
	oldRerankEndpoint := os.Getenv("VOYAGE_RERANK_ENDPOINT")
	oldEnableReranker := os.Getenv("ENABLE_RERANKER")
	defer func() {
		os.Setenv("VOYAGE_API_KEY", oldAPIKey)
		os.Setenv("VOYAGE_MODEL", oldModel)
		os.Setenv("VOYAGE_EMBEDDING_ENDPOINT", oldEndpoint)
		os.Setenv("VOYAGE_RERANK_MODEL", oldRerankModel)
		os.Setenv("VOYAGE_RERANK_ENDPOINT", oldRerankEndpoint)
		os.Setenv("ENABLE_RERANKER", oldEnableReranker)
	}()

	os.Setenv("VOYAGE_API_KEY", "env-api-key")
	os.Setenv("VOYAGE_MODEL", "env-model")
	os.Setenv("VOYAGE_EMBEDDING_ENDPOINT", "http://env-endpoint.com")
	os.Setenv("VOYAGE_RERANK_MODEL", "env-rerank")
	os.Setenv("VOYAGE_RERANK_ENDPOINT", "http://env-rerank.com")
	os.Setenv("ENABLE_RERANKER", "true")

	cfg, err := Load(testConfigPath)
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	if cfg.APIKey != "env-api-key" {
		t.Errorf("Expected env override for API key, got '%s'", cfg.APIKey)
	}

	if cfg.Model != "env-model" {
		t.Errorf("Expected env override for model, got '%s'", cfg.Model)
	}

	if cfg.Endpoint != "http://env-endpoint.com" {
		t.Errorf("Expected env override for endpoint, got '%s'", cfg.Endpoint)
	}

	if cfg.RerankModel != "env-rerank" {
		t.Errorf("Expected env override for rerank model, got '%s'", cfg.RerankModel)
	}

	if cfg.EnableReranker != true {
		t.Error("Expected env override for enable reranker")
	}
}

func TestEnsureIndexDir_CreatesDirectory(t *testing.T) {
	tempDir := t.TempDir()
	indexDir := filepath.Join(tempDir, "indexes")

	cfg := &Config{IndexDir: indexDir}

	err := cfg.EnsureIndexDir()
	if err != nil {
		t.Fatalf("Failed to ensure index directory: %v", err)
	}

	if _, err := os.Stat(indexDir); os.IsNotExist(err) {
		t.Error("Index directory was not created")
	}
}

func TestEnsureIndexDir_ExistingDirectory(t *testing.T) {
	tempDir := t.TempDir()
	indexDir := filepath.Join(tempDir, "existing-indexes")

	err := os.MkdirAll(indexDir, 0755)
	if err != nil {
		t.Fatalf("Failed to create test directory: %v", err)
	}

	cfg := &Config{IndexDir: indexDir}

	err = cfg.EnsureIndexDir()
	if err != nil {
		t.Fatalf("Failed to ensure existing index directory: %v", err)
	}
}

func TestSave_CreatesDirectory(t *testing.T) {
	tempDir := t.TempDir()
	testConfigPath := filepath.Join(tempDir, "nested", "deep", "config.yaml")

	cfg := &Config{
		EmbeddingProvider: "voyage",
		APIKey:            "test-key",
		IndexDir:          tempDir,
	}

	err := cfg.Save(testConfigPath)
	if err != nil {
		t.Fatalf("Failed to save config: %v", err)
	}

	if _, err := os.Stat(filepath.Dir(testConfigPath)); os.IsNotExist(err) {
		t.Error("Config directory was not created")
	}

	if _, err := os.Stat(testConfigPath); os.IsNotExist(err) {
		t.Error("Config file was not created")
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0 ||
		(len(s) > 0 && len(substr) > 0 && findSubstring(s, substr)))
}

func findSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
