package testutil

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/karim-daw/qwelli/internal/config"
	"github.com/karim-daw/qwelli/internal/db"
	"github.com/karim-daw/qwelli/internal/engine"
)

// CreateTestConfig creates a test configuration with mock Voyage client settings
func CreateTestConfig(t *testing.T) *config.Config {
	t.Helper()

	// Create temp index directory
	tempDir := t.TempDir()
	indexDir := filepath.Join(tempDir, "indexes")
	if err := os.MkdirAll(indexDir, 0755); err != nil {
		t.Fatalf("Failed to create test index directory: %v", err)
	}

	return &config.Config{
		EmbeddingProvider: "voyage",
		APIKey:            "test-api-key",
		Model:             "mock-voyage-model",
		Endpoint:          "http://mock-endpoint.test",
		EnableMultimodal:  true,
		ImageQuality:      "medium",
		EnableReranker:    true,
		RerankProvider:    "voyage",
		RerankModel:       "mock-rerank-model",
		RerankEndpoint:    "http://mock-rerank-endpoint.test",
		IndexDir:          indexDir,
	}
}

// CreateTestConfigWithDir creates a test configuration with a specific index directory
func CreateTestConfigWithDir(t *testing.T, indexDir string) *config.Config {
	t.Helper()

	return &config.Config{
		EmbeddingProvider: "voyage",
		APIKey:            "test-api-key",
		Model:             "mock-voyage-model",
		Endpoint:          "http://mock-endpoint.test",
		EnableMultimodal:  true,
		ImageQuality:      "medium",
		EnableReranker:    true,
		RerankProvider:    "voyage",
		RerankModel:       "mock-rerank-model",
		RerankEndpoint:    "http://mock-rerank-endpoint.test",
		IndexDir:          indexDir,
	}
}

// CreateTestEngine creates a test engine with mock Voyage client
func CreateTestEngine(t *testing.T, dimension int) (*engine.Engine, *MockVoyageClient) {
	t.Helper()

	mockClient := NewMockVoyageClient(dimension)
	return engine.NewEngine(mockClient, true), mockClient
}

// CreateTestFiles creates a set of test files in the specified directory
func CreateTestFiles(t *testing.T, dir string, files map[string]string) {
	t.Helper()

	for filename, content := range files {
		filePath := filepath.Join(dir, filename)

		// Create parent directory if needed
		parentDir := filepath.Dir(filePath)
		if err := os.MkdirAll(parentDir, 0755); err != nil {
			t.Fatalf("Failed to create directory %s: %v", parentDir, err)
		}

		// Write file
		if err := os.WriteFile(filePath, []byte(content), 0644); err != nil {
			t.Fatalf("Failed to create test file %s: %v", filename, err)
		}
	}
}

// CreateTestIndex creates and populates a test index with sample files
func CreateTestIndex(t *testing.T, indexDir, folderPath string, files map[string]string, dimension int) *db.ProjectDB {
	t.Helper()

	// Create test files
	CreateTestFiles(t, folderPath, files)

	// Generate DB path
	absPath, err := filepath.Abs(folderPath)
	if err != nil {
		t.Fatalf("Failed to get absolute path: %v", err)
	}

	dbPath := filepath.Join(indexDir, safeDBName(absPath))

	// Create database
	projectDB, err := db.OpenProjectDB(dbPath, dimension)
	if err != nil {
		t.Fatalf("Failed to create test database: %v", err)
	}

	// Store folder path in metadata
	if err := projectDB.SetMetadata("folder_path", absPath); err != nil {
		t.Fatalf("Failed to set folder path metadata: %v", err)
	}

	return projectDB
}

// safeDBName creates a safe database filename from a folder path
// This mirrors the implementation in service/service.go
func safeDBName(folderPath string) string {
	name := filepath.Base(folderPath)
	if name == "" || name == "." || name == "/" {
		name = "root"
	}

	// Replace spaces and special chars with underscores
	safe := ""
	for _, r := range name {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '-' {
			safe += string(r)
		} else {
			safe += "_"
		}
	}

	return safe + ".db"
}

// AssertSearchResults verifies that search results match expectations
func AssertSearchResults(t *testing.T, results []engine.SearchResult, expectedCount int, expectedInContent string) {
	t.Helper()

	if len(results) != expectedCount {
		t.Errorf("Expected %d results, got %d", expectedCount, len(results))
	}

	if expectedInContent != "" {
		found := false
		for _, result := range results {
			if contains(result.Content, expectedInContent) {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Expected to find '%s' in results, but it was not found", expectedInContent)
		}
	}
}

// AssertSearchResultsContain verifies that search results contain all expected strings
func AssertSearchResultsContain(t *testing.T, results []engine.SearchResult, expectedStrings []string) {
	t.Helper()

	for _, expected := range expectedStrings {
		found := false
		for _, result := range results {
			if contains(result.Content, expected) {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Expected to find '%s' in results, but it was not found", expected)
		}
	}
}

// AssertNoError asserts that no error occurred
func AssertNoError(t *testing.T, err error, message string) {
	t.Helper()

	if err != nil {
		t.Fatalf("%s: %v", message, err)
	}
}

// AssertError asserts that an error occurred
func AssertError(t *testing.T, err error, message string) {
	t.Helper()

	if err == nil {
		t.Fatalf("%s: expected error but got nil", message)
	}
}

// AssertErrorContains asserts that an error occurred and contains the expected substring
func AssertErrorContains(t *testing.T, err error, expectedSubstring string) {
	t.Helper()

	if err == nil {
		t.Fatalf("Expected error containing '%s' but got nil", expectedSubstring)
	}

	if !contains(err.Error(), expectedSubstring) {
		t.Fatalf("Expected error to contain '%s', but got: %v", expectedSubstring, err)
	}
}

// CaptureStdout captures stdout during function execution
func CaptureStdout(t *testing.T, fn func()) string {
	t.Helper()

	// Save original stdout
	oldStdout := os.Stdout

	// Create pipe
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("Failed to create pipe: %v", err)
	}

	// Replace stdout
	os.Stdout = w

	// Channel to capture output
	outC := make(chan string)

	// Copy goroutine
	go func() {
		var buf bytes.Buffer
		io.Copy(&buf, r)
		outC <- buf.String()
	}()

	// Execute function
	fn()

	// Restore stdout
	w.Close()
	os.Stdout = oldStdout

	// Get captured output
	output := <-outC

	return output
}

// IndexTestData creates a complete test index with files and embeddings
func IndexTestData(t *testing.T, eng *engine.Engine, projectDB *db.ProjectDB, folderPath string, mockClient *MockVoyageClient) {
	t.Helper()

	ctx := context.Background()

	// Use engine to index the folder (this will use the mock client)
	progressCallback := func(current, total int, filename string) {
		// Silent progress callback for tests
	}

	phaseCallback := func(phase, message string, current, total int) {
		// Silent phase callback for tests
	}

	err := eng.IndexFolder(ctx, projectDB, folderPath, false, progressCallback, phaseCallback)
	if err != nil {
		t.Fatalf("Failed to index test data: %v", err)
	}
}

// CleanupTestDB closes and removes a test database
func CleanupTestDB(t *testing.T, projectDB *db.ProjectDB, dbPath string) {
	t.Helper()

	if projectDB != nil {
		if err := projectDB.Close(); err != nil {
			t.Logf("Warning: Failed to close test database: %v", err)
		}
	}

	if dbPath != "" {
		if err := os.Remove(dbPath); err != nil && !os.IsNotExist(err) {
			t.Logf("Warning: Failed to remove test database: %v", err)
		}
	}
}

// WaitForCompletion waits for a function to return true or times out
func WaitForCompletion(t *testing.T, checkFn func() bool, timeoutMsg string) {
	t.Helper()

	// Simple synchronous check for tests
	if !checkFn() {
		t.Fatal(timeoutMsg)
	}
}

// CreateSampleTextFiles returns a map of sample text files for testing
func CreateSampleTextFiles() map[string]string {
	return map[string]string{
		"hello.txt":   "Hello, world! This is a test file.",
		"readme.md":   "# README\n\nThis is a sample readme file with markdown content.",
		"code.go":     "package main\n\nimport \"fmt\"\n\nfunc main() {\n\tfmt.Println(\"Hello, Go!\")\n}",
		"script.py":   "#!/usr/bin/env python3\n\nprint('Hello, Python!')",
		"data.json":   "{\"name\": \"test\", \"value\": 123}",
		"config.yaml": "version: 1.0\nname: test-config",
	}
}

// CreateLargeTextFile creates a file with enough content to be chunked
func CreateLargeTextFile(sentences int) string {
	content := ""
	for i := 0; i < sentences; i++ {
		content += fmt.Sprintf("This is sentence number %d in a large document. ", i+1)
		if (i+1)%10 == 0 {
			content += "\n\n" // Add paragraph breaks
		}
	}
	return content
}

// contains checks if a string contains a substring (case-sensitive)
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

// MockProgressCallback creates a mock progress callback for testing
func MockProgressCallback() func(current, total int, filename string) {
	return func(current, total int, filename string) {
		// Silent for tests
	}
}

// MockPhaseCallback creates a mock phase callback for testing
func MockPhaseCallback() func(phase, message string, current, total int) {
	return func(phase, message string, current, total int) {
		// Silent for tests
	}
}

// CreateTestFolder creates a temporary folder with the given name
func CreateTestFolder(t *testing.T, name string) string {
	t.Helper()

	tempDir := t.TempDir()
	folderPath := filepath.Join(tempDir, name)

	if err := os.MkdirAll(folderPath, 0755); err != nil {
		t.Fatalf("Failed to create test folder: %v", err)
	}

	return folderPath
}

// AssertFileExists checks that a file exists
func AssertFileExists(t *testing.T, path string) {
	t.Helper()

	if _, err := os.Stat(path); os.IsNotExist(err) {
		t.Fatalf("Expected file to exist: %s", path)
	}
}

// AssertFileNotExists checks that a file does not exist
func AssertFileNotExists(t *testing.T, path string) {
	t.Helper()

	if _, err := os.Stat(path); err == nil {
		t.Fatalf("Expected file to not exist: %s", path)
	}
}

// AssertDirEmpty checks that a directory is empty
func AssertDirEmpty(t *testing.T, dir string) {
	t.Helper()

	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("Failed to read directory: %v", err)
	}

	if len(entries) > 0 {
		t.Fatalf("Expected directory to be empty, but found %d entries", len(entries))
	}
}
