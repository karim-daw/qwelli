package config

import (
	"os"
	"testing"
)

func TestGetEnv(t *testing.T) {
	tests := []struct {
		name         string
		key          string
		envValue     string
		setEnv       bool
		defaultValue string
		expected     string
	}{
		{
			name:         "returns env value when set",
			key:          "TEST_GET_ENV_SET",
			envValue:     "custom_value",
			setEnv:       true,
			defaultValue: "default",
			expected:     "custom_value",
		},
		{
			name:         "returns default when not set",
			key:          "TEST_GET_ENV_UNSET",
			setEnv:       false,
			defaultValue: "default",
			expected:     "default",
		},
		{
			name:         "returns default when empty",
			key:          "TEST_GET_ENV_EMPTY",
			envValue:     "",
			setEnv:       true,
			defaultValue: "default",
			expected:     "default",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.setEnv {
				os.Setenv(tt.key, tt.envValue)
				defer os.Unsetenv(tt.key)
			} else {
				os.Unsetenv(tt.key)
			}

			result := getEnv(tt.key, tt.defaultValue)
			if result != tt.expected {
				t.Errorf("getEnv(%q, %q) = %q, want %q", tt.key, tt.defaultValue, result, tt.expected)
			}
		})
	}
}

func TestGetEnvAsInt(t *testing.T) {
	tests := []struct {
		name         string
		key          string
		envValue     string
		setEnv       bool
		defaultValue int
		expected     int
	}{
		{
			name:         "returns parsed int when set",
			key:          "TEST_INT_SET",
			envValue:     "9090",
			setEnv:       true,
			defaultValue: 8080,
			expected:     9090,
		},
		{
			name:         "returns default when not set",
			key:          "TEST_INT_UNSET",
			setEnv:       false,
			defaultValue: 8080,
			expected:     8080,
		},
		{
			name:         "returns default when invalid",
			key:          "TEST_INT_INVALID",
			envValue:     "not_a_number",
			setEnv:       true,
			defaultValue: 8080,
			expected:     8080,
		},
		{
			name:         "handles zero",
			key:          "TEST_INT_ZERO",
			envValue:     "0",
			setEnv:       true,
			defaultValue: 8080,
			expected:     0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.setEnv {
				os.Setenv(tt.key, tt.envValue)
				defer os.Unsetenv(tt.key)
			} else {
				os.Unsetenv(tt.key)
			}

			result := getEnvAsInt(tt.key, tt.defaultValue)
			if result != tt.expected {
				t.Errorf("getEnvAsInt(%q, %d) = %d, want %d", tt.key, tt.defaultValue, result, tt.expected)
			}
		})
	}
}

func TestGetEnvAsBool(t *testing.T) {
	tests := []struct {
		name         string
		key          string
		envValue     string
		setEnv       bool
		defaultValue bool
		expected     bool
	}{
		{
			name:         "true string",
			key:          "TEST_BOOL_TRUE",
			envValue:     "true",
			setEnv:       true,
			defaultValue: false,
			expected:     true,
		},
		{
			name:         "1 string",
			key:          "TEST_BOOL_ONE",
			envValue:     "1",
			setEnv:       true,
			defaultValue: false,
			expected:     true,
		},
		{
			name:         "false string",
			key:          "TEST_BOOL_FALSE",
			envValue:     "false",
			setEnv:       true,
			defaultValue: true,
			expected:     false,
		},
		{
			name:         "returns default when not set",
			key:          "TEST_BOOL_UNSET",
			setEnv:       false,
			defaultValue: true,
			expected:     true,
		},
		{
			name:         "invalid value treated as false",
			key:          "TEST_BOOL_INVALID",
			envValue:     "yes",
			setEnv:       true,
			defaultValue: true,
			expected:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.setEnv {
				os.Setenv(tt.key, tt.envValue)
				defer os.Unsetenv(tt.key)
			} else {
				os.Unsetenv(tt.key)
			}

			result := getEnvAsBool(tt.key, tt.defaultValue)
			if result != tt.expected {
				t.Errorf("getEnvAsBool(%q, %v) = %v, want %v", tt.key, tt.defaultValue, result, tt.expected)
			}
		})
	}
}

func TestLoad_MissingDatabaseURL(t *testing.T) {
	// Clear relevant env vars
	os.Unsetenv("DATABASE_URL")
	os.Setenv("VOYAGE_API_KEY", "test-key")
	defer os.Unsetenv("VOYAGE_API_KEY")

	_, err := Load()
	if err == nil {
		t.Error("Load() should return error when DATABASE_URL is missing")
	}
}

func TestLoad_MissingVoyageAPIKey(t *testing.T) {
	os.Setenv("DATABASE_URL", "postgresql://localhost/test")
	os.Unsetenv("VOYAGE_API_KEY")
	defer os.Unsetenv("DATABASE_URL")

	_, err := Load()
	if err == nil {
		t.Error("Load() should return error when VOYAGE_API_KEY is missing")
	}
}

func TestLoad_ValidConfig(t *testing.T) {
	os.Setenv("DATABASE_URL", "postgresql://localhost/test")
	os.Setenv("VOYAGE_API_KEY", "test-key")
	defer os.Unsetenv("DATABASE_URL")
	defer os.Unsetenv("VOYAGE_API_KEY")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() returned unexpected error: %v", err)
	}

	if cfg.DatabaseURL != "postgresql://localhost/test" {
		t.Errorf("DatabaseURL = %q, want %q", cfg.DatabaseURL, "postgresql://localhost/test")
	}
	if cfg.VoyageAPIKey != "test-key" {
		t.Errorf("VoyageAPIKey = %q, want %q", cfg.VoyageAPIKey, "test-key")
	}
	if cfg.VoyageModel != "voyage-multimodal-3.5" {
		t.Errorf("VoyageModel = %q, want default %q", cfg.VoyageModel, "voyage-multimodal-3.5")
	}
	if cfg.Port != 8080 {
		t.Errorf("Port = %d, want default %d", cfg.Port, 8080)
	}
	if cfg.EnableReranker != true {
		t.Error("EnableReranker should default to true")
	}
}

func TestLoad_CustomPort(t *testing.T) {
	os.Setenv("DATABASE_URL", "postgresql://localhost/test")
	os.Setenv("VOYAGE_API_KEY", "test-key")
	os.Setenv("PORT", "3000")
	defer os.Unsetenv("DATABASE_URL")
	defer os.Unsetenv("VOYAGE_API_KEY")
	defer os.Unsetenv("PORT")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() returned unexpected error: %v", err)
	}

	if cfg.Port != 3000 {
		t.Errorf("Port = %d, want %d", cfg.Port, 3000)
	}
}
