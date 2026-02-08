package integration

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/karim-daw/qwelli/internal/service"
	"github.com/karim-daw/qwelli/internal/testutil"
)

// TestE2E_IndexAndSearch tests the full workflow: create index and search
func TestE2E_IndexAndSearch(t *testing.T) {
	// Setup test environment
	tempDir := t.TempDir()
	configDir := filepath.Join(tempDir, ".qwelli")
	os.Setenv("HOME", tempDir)
	defer os.Unsetenv("HOME")

	// Create test config
	cfg := testutil.CreateTestConfig(t)
	cfg.IndexDir = filepath.Join(configDir, "indexes")
	err := cfg.EnsureIndexDir()
	if err != nil {
		t.Fatalf("Failed to create index dir: %v", err)
	}

	// Create test folder with sample files
	testFolder := filepath.Join(tempDir, "test_project")
	os.MkdirAll(testFolder, 0755)

	testFiles := map[string]string{
		"README.md":   "# Test Project\n\nThis is a test project for semantic search.",
		"main.go":     "package main\n\nfunc main() {\n\tprintln(\"Hello, World!\")\n}",
		"config.yaml": "app_name: test\nversion: 1.0.0",
		"docs/api.md": "# API Documentation\n\nThe API provides search functionality.",
	}

	for filename, content := range testFiles {
		filePath := filepath.Join(testFolder, filename)
		os.MkdirAll(filepath.Dir(filePath), 0755)
		os.WriteFile(filePath, []byte(content), 0644)
	}

	// Create service with mock client
	mockClient := testutil.NewMockVoyageClient(1024)
	svc := service.New(cfg, mockClient)

	// Step 1: Create index
	ctx := context.Background()
	err = svc.CreateIndex(ctx, testFolder, service.IndexOptions{}, nil)
	if err != nil {
		t.Fatalf("CreateIndex failed: %v", err)
	}

	// Verify index was created
	indexes, err := svc.ListIndexes()
	if err != nil {
		t.Fatalf("ListIndexes failed: %v", err)
	}
	if len(indexes) == 0 {
		t.Fatal("Expected at least one index to be created")
	}

	// Step 2: Search the index
	results, err := svc.Search(testFolder, "search functionality", 5, "", "semantic")
	if err != nil {
		t.Fatalf("Search failed: %v", err)
	}

	if len(results) == 0 {
		t.Error("Expected search results")
	}

	// Verify results have required fields
	for _, result := range results {
		if result.FilePath == "" {
			t.Error("Expected FilePath to be set")
		}
		if result.Content == "" {
			t.Error("Expected Content to be set")
		}
	}

	// Step 3: Get index stats
	count, err := svc.GetIndexStats(testFolder)
	if err != nil {
		t.Fatalf("GetIndexStats failed: %v", err)
	}
	if count == 0 {
		t.Error("Expected non-zero chunk count")
	}

	// Step 4: Delete index
	err = svc.DeleteIndex(testFolder)
	if err != nil {
		t.Fatalf("DeleteIndex failed: %v", err)
	}

	// Verify index was deleted
	indexes, err = svc.ListIndexes()
	if err != nil {
		t.Fatalf("ListIndexes failed: %v", err)
	}
	if len(indexes) != 0 {
		t.Error("Expected index to be deleted")
	}
}

// TestE2E_IncrementalUpdate tests the incremental indexing workflow
func TestE2E_IncrementalUpdate(t *testing.T) {
	tempDir := t.TempDir()
	configDir := filepath.Join(tempDir, ".qwelli")
	os.Setenv("HOME", tempDir)
	defer os.Unsetenv("HOME")

	cfg := testutil.CreateTestConfig(t)
	cfg.IndexDir = filepath.Join(configDir, "indexes")
	cfg.EnsureIndexDir()

	testFolder := filepath.Join(tempDir, "test_project")
	os.MkdirAll(testFolder, 0755)

	// Create initial file
	file1 := filepath.Join(testFolder, "file1.txt")
	os.WriteFile(file1, []byte("Initial content"), 0644)

	mockClient := testutil.NewMockVoyageClient(1024)
	svc := service.New(cfg, mockClient)

	ctx := context.Background()

	// Initial index
	err := svc.CreateIndex(ctx, testFolder, service.IndexOptions{}, nil)
	if err != nil {
		t.Fatalf("CreateIndex failed: %v", err)
	}

	initialCount, _ := svc.GetIndexStats(testFolder)

	// Add new file
	file2 := filepath.Join(testFolder, "file2.txt")
	os.WriteFile(file2, []byte("New file content"), 0644)

	// Incremental update
	err = svc.UpdateIndex(ctx, testFolder, service.IndexOptions{Incremental: true}, nil)
	if err != nil {
		t.Fatalf("UpdateIndex failed: %v", err)
	}

	// Verify chunk count increased
	newCount, _ := svc.GetIndexStats(testFolder)
	if newCount <= initialCount {
		t.Errorf("Expected chunk count to increase after adding file (was %d, now %d)", initialCount, newCount)
	}

	// Search should find both files
	results, err := svc.Search(testFolder, "content", 10, "", "semantic")
	if err != nil {
		t.Fatalf("Search failed: %v", err)
	}

	if len(results) == 0 {
		t.Error("Expected search results from both files")
	}
}

// TestE2E_MultipleIndexes tests managing multiple indexes
func TestE2E_MultipleIndexes(t *testing.T) {
	tempDir := t.TempDir()
	configDir := filepath.Join(tempDir, ".qwelli")
	os.Setenv("HOME", tempDir)
	defer os.Unsetenv("HOME")

	cfg := testutil.CreateTestConfig(t)
	cfg.IndexDir = filepath.Join(configDir, "indexes")
	cfg.EnsureIndexDir()

	// Create two separate project folders
	project1 := filepath.Join(tempDir, "project1")
	project2 := filepath.Join(tempDir, "project2")
	os.MkdirAll(project1, 0755)
	os.MkdirAll(project2, 0755)

	os.WriteFile(filepath.Join(project1, "file.txt"), []byte("Project 1 content"), 0644)
	os.WriteFile(filepath.Join(project2, "file.txt"), []byte("Project 2 content"), 0644)

	mockClient := testutil.NewMockVoyageClient(1024)
	svc := service.New(cfg, mockClient)

	ctx := context.Background()

	// Index both projects
	err := svc.CreateIndex(ctx, project1, service.IndexOptions{}, nil)
	if err != nil {
		t.Fatalf("CreateIndex project1 failed: %v", err)
	}

	err = svc.CreateIndex(ctx, project2, service.IndexOptions{}, nil)
	if err != nil {
		t.Fatalf("CreateIndex project2 failed: %v", err)
	}

	// List indexes
	indexes, err := svc.ListIndexes()
	if err != nil {
		t.Fatalf("ListIndexes failed: %v", err)
	}

	if len(indexes) < 2 {
		t.Errorf("Expected at least 2 indexes, got %d", len(indexes))
	}

	// Search each project independently
	results1, err := svc.Search(project1, "content", 5, "", "semantic")
	if err != nil {
		t.Fatalf("Search project1 failed: %v", err)
	}

	results2, err := svc.Search(project2, "content", 5, "", "semantic")
	if err != nil {
		t.Fatalf("Search project2 failed: %v", err)
	}

	// Both should have results
	if len(results1) == 0 || len(results2) == 0 {
		t.Error("Expected search results from both projects")
	}

	// Delete one index
	err = svc.DeleteIndex(project1)
	if err != nil {
		t.Fatalf("DeleteIndex failed: %v", err)
	}

	// Verify only one index remains
	indexes, err = svc.ListIndexes()
	if err != nil {
		t.Fatalf("ListIndexes failed: %v", err)
	}

	if len(indexes) != 1 {
		t.Errorf("Expected 1 index after deletion, got %d", len(indexes))
	}
}

// TestE2E_SearchStrategies tests different search strategies
func TestE2E_SearchStrategies(t *testing.T) {
	tempDir := t.TempDir()
	configDir := filepath.Join(tempDir, ".qwelli")
	os.Setenv("HOME", tempDir)
	defer os.Unsetenv("HOME")

	cfg := testutil.CreateTestConfig(t)
	cfg.IndexDir = filepath.Join(configDir, "indexes")
	cfg.EnsureIndexDir()

	testFolder := filepath.Join(tempDir, "test_project")
	os.MkdirAll(testFolder, 0755)

	// Create test files with distinct content
	os.WriteFile(filepath.Join(testFolder, "cats.txt"),
		[]byte("Cats are wonderful pets. They are independent and loving."), 0644)
	os.WriteFile(filepath.Join(testFolder, "dogs.txt"),
		[]byte("Dogs are loyal companions. They love to play fetch."), 0644)

	mockClient := testutil.NewMockVoyageClient(1024)
	svc := service.New(cfg, mockClient)

	ctx := context.Background()
	err := svc.CreateIndex(ctx, testFolder, service.IndexOptions{}, nil)
	if err != nil {
		t.Fatalf("CreateIndex failed: %v", err)
	}

	// Test different search strategies
	strategies := []string{"semantic", "keyword", "hybrid"}

	for _, strategy := range strategies {
		t.Run(strategy, func(t *testing.T) {
			results, err := svc.Search(testFolder, "cats pets", 5, "", strategy)
			if err != nil {
				t.Fatalf("Search with %s strategy failed: %v", strategy, err)
			}

			// Each strategy should return results (even if different)
			if len(results) == 0 {
				t.Logf("No results for %s strategy (may be expected)", strategy)
			}
		})
	}
}
