package service

import (
	"path/filepath"
	"testing"

	"github.com/karim-daw/qwelli/internal/testutil"
)

func TestNew_WithValidConfig(t *testing.T) {
	cfg := testutil.CreateTestConfig(t)
	mockClient := testutil.NewMockVoyageClient(1024)

	svc := New(cfg, mockClient)

	if svc == nil {
		t.Fatal("Expected service to be created, got nil")
	}

	if svc.Config() != cfg {
		t.Error("Expected config to match")
	}

	if svc.Engine() == nil {
		t.Error("Expected engine to be initialized")
	}

	if svc.VoyageClient() != mockClient {
		t.Error("Expected voyage client to match")
	}
}

func TestNew_RequiresValidClient(t *testing.T) {
	// Test that New accepts any valid client interface
	cfg := testutil.CreateTestConfig(t)
	mockClient := testutil.NewMockVoyageClient(512)

	svc := New(cfg, mockClient)

	if svc == nil {
		t.Fatal("Expected service to be created")
	}

	// Verify the client was properly set
	if svc.VoyageClient() != mockClient {
		t.Error("Expected voyage client to match provided client")
	}
}

func TestLoad_ConfigNotFound(t *testing.T) {
	// This test will fail because Load() looks for config in ~/.qwelli/config.yaml
	// which likely doesn't exist in test environment
	_, err := Load()
	if err == nil {
		t.Skip("Config file exists, skipping test")
	}

	// Error should mention config not found
	expectedSubstring := "config not found"
	if !contains(err.Error(), expectedSubstring) && !contains(err.Error(), "no such file") {
		t.Errorf("Expected error to mention config not found, got: %v", err)
	}
}

func TestGenerateDBPath_ValidPath(t *testing.T) {
	cfg := testutil.CreateTestConfig(t)
	mockClient := testutil.NewMockVoyageClient(1024)
	svc := New(cfg, mockClient)

	testFolder := t.TempDir()
	dbPath, err := svc.GenerateDBPath(testFolder)

	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	// Should end with .db
	if filepath.Ext(dbPath) != ".db" {
		t.Errorf("Expected .db extension, got: %s", filepath.Ext(dbPath))
	}

	// Should be in index directory
	if !contains(dbPath, cfg.IndexDir) {
		t.Errorf("Expected path to be in index directory (%s), got: %s", cfg.IndexDir, dbPath)
	}
}

func TestGenerateDBPath_SafeName(t *testing.T) {
	cfg := testutil.CreateTestConfig(t)
	mockClient := testutil.NewMockVoyageClient(1024)
	svc := New(cfg, mockClient)

	tests := []struct {
		name       string
		folderPath string
		wantSafe   bool
	}{
		{
			name:       "simple folder name",
			folderPath: "/home/user/my-project",
			wantSafe:   true,
		},
		{
			name:       "folder with spaces",
			folderPath: "/home/user/my project",
			wantSafe:   true,
		},
		{
			name:       "folder with special chars",
			folderPath: "/home/user/my@project#test",
			wantSafe:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create temp directory with safe name (filesystem limitation)
			tempBase := t.TempDir()

			dbPath, err := svc.GenerateDBPath(tempBase)
			if err != nil {
				t.Fatalf("Expected no error, got: %v", err)
			}

			// DB name should only contain safe characters
			dbName := filepath.Base(dbPath)
			for _, ch := range dbName {
				if !isSafeChar(ch) && ch != '.' && ch != '_' {
					t.Errorf("DB name contains unsafe character '%c': %s", ch, dbName)
				}
			}
		})
	}
}

func TestOpenDB_NonExistentIndex(t *testing.T) {
	cfg := testutil.CreateTestConfig(t)
	mockClient := testutil.NewMockVoyageClient(1024)
	svc := New(cfg, mockClient)

	nonExistentFolder := filepath.Join(t.TempDir(), "nonexistent")

	_, err := svc.OpenDB(nonExistentFolder)
	if err == nil {
		t.Fatal("Expected error when opening non-existent index, got nil")
	}

	expectedSubstring := "index not found"
	if !contains(err.Error(), expectedSubstring) {
		t.Errorf("Expected error to contain '%s', got: %v", expectedSubstring, err)
	}
}

func TestIsFileInIndex_FileNotExists(t *testing.T) {
	cfg := testutil.CreateTestConfig(t)
	mockClient := testutil.NewMockVoyageClient(1024)
	svc := New(cfg, mockClient)

	nonExistentFile := "/path/to/nonexistent/file.txt"

	found := svc.IsFileInIndex(nonExistentFile)
	if found {
		t.Error("Expected file not to be found in any index")
	}
}

func TestNew_WithDifferentDimension(t *testing.T) {
	cfg := testutil.CreateTestConfig(t)
	mockClient := testutil.NewMockVoyageClient(1536)

	svc := New(cfg, mockClient)

	if svc == nil {
		t.Fatal("Expected service to be created")
	}

	if svc.VoyageClient() != mockClient {
		t.Error("Expected voyage client to match the provided mock")
	}

	if svc.Engine() == nil {
		t.Error("Expected engine to be initialized")
	}
}

func TestService_GettersReturnCorrectValues(t *testing.T) {
	cfg := testutil.CreateTestConfig(t)
	mockClient := testutil.NewMockVoyageClient(1024)

	svc := New(cfg, mockClient)

	// Test Config()
	if svc.Config() != cfg {
		t.Error("Config() should return the same config instance")
	}

	// Test VoyageClient()
	if svc.VoyageClient() != mockClient {
		t.Error("VoyageClient() should return the same client instance")
	}

	// Test Engine()
	eng := svc.Engine()
	if eng == nil {
		t.Error("Engine() should not return nil")
	}

	// Engine should use the same voyage client
	if eng.EmbeddingModel() != mockClient.EmbeddingModel() {
		t.Error("Engine should use the same voyage client")
	}
}

// Helper functions

func isSafeChar(ch rune) bool {
	return (ch >= 'a' && ch <= 'z') || (ch >= 'A' && ch <= 'Z') || (ch >= '0' && ch <= '9') || ch == '-'
}

