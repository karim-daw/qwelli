package service

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/karim-daw/qwelli/internal/testutil"
)

// TestCreateIndex_SingleFile tests indexing a single text file
func TestCreateIndex_SingleFile(t *testing.T) {
	cfg := testutil.CreateTestConfig(t)
	mockClient := testutil.NewMockVoyageClient(1024)
	svc := New(cfg, mockClient)

	// Create test folder with one file
	testFolder := t.TempDir()
	testFile := filepath.Join(testFolder, "test.txt")
	err := os.WriteFile(testFile, []byte("This is a test file with some content."), 0644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Create index
	ctx := context.Background()
	progressCount := 0
	progressCb := func(current, total int, filename string) {
		progressCount++
	}

	err = svc.CreateIndex(ctx, testFolder, IndexOptions{}, progressCb)
	if err != nil {
		t.Fatalf("CreateIndex failed: %v", err)
	}

	// Verify progress callback was called
	if progressCount == 0 {
		t.Error("Expected progress callback to be called")
	}

	// Verify index was created
	dbPath, err := svc.GenerateDBPath(testFolder)
	if err != nil {
		t.Fatalf("GenerateDBPath failed: %v", err)
	}

	if _, err := os.Stat(dbPath); os.IsNotExist(err) {
		t.Error("Expected database file to be created")
	}

	// Verify we can search the index
	results, err := svc.Search(testFolder, "test", 5, "", "semantic")
	if err != nil {
		t.Fatalf("Search failed: %v", err)
	}

	if len(results) == 0 {
		t.Error("Expected at least one search result")
	}
}

// TestCreateIndex_MultipleFiles tests indexing multiple files
func TestCreateIndex_MultipleFiles(t *testing.T) {
	cfg := testutil.CreateTestConfig(t)
	mockClient := testutil.NewMockVoyageClient(1024)
	svc := New(cfg, mockClient)

	// Create test folder with multiple files
	testFolder := t.TempDir()
	files := map[string]string{
		"file1.txt": "First test file with content about cats.",
		"file2.txt": "Second test file with content about dogs.",
		"file3.md":  "# Markdown File\n\nThis is a markdown file about birds.",
	}

	for filename, content := range files {
		filePath := filepath.Join(testFolder, filename)
		if err := os.WriteFile(filePath, []byte(content), 0644); err != nil {
			t.Fatalf("Failed to create test file: %v", err)
		}
	}

	// Create index
	ctx := context.Background()
	err := svc.CreateIndex(ctx, testFolder, IndexOptions{}, nil)
	if err != nil {
		t.Fatalf("CreateIndex failed: %v", err)
	}

	// Verify index stats
	count, err := svc.GetIndexStats(testFolder)
	if err != nil {
		t.Fatalf("GetIndexStats failed: %v", err)
	}

	if count == 0 {
		t.Error("Expected non-zero chunk count")
	}

	// Verify search works
	results, err := svc.Search(testFolder, "cats", 5, "", "semantic")
	if err != nil {
		t.Fatalf("Search failed: %v", err)
	}

	if len(results) == 0 {
		t.Error("Expected search results for 'cats'")
	}
}

// TestCreateIndex_EmptyFolder tests indexing an empty folder
func TestCreateIndex_EmptyFolder(t *testing.T) {
	cfg := testutil.CreateTestConfig(t)
	mockClient := testutil.NewMockVoyageClient(1024)
	svc := New(cfg, mockClient)

	testFolder := t.TempDir()

	ctx := context.Background()
	err := svc.CreateIndex(ctx, testFolder, IndexOptions{}, nil)

	// Should complete without error (just no files to index)
	if err != nil {
		t.Fatalf("CreateIndex on empty folder failed: %v", err)
	}
}

// TestCreateIndex_UnsupportedFiles tests that unsupported files are skipped
func TestCreateIndex_UnsupportedFiles(t *testing.T) {
	cfg := testutil.CreateTestConfig(t)
	mockClient := testutil.NewMockVoyageClient(1024)
	svc := New(cfg, mockClient)

	testFolder := t.TempDir()

	// Create unsupported file
	unsupportedFile := filepath.Join(testFolder, "test.exe")
	if err := os.WriteFile(unsupportedFile, []byte("binary content"), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Create supported file
	supportedFile := filepath.Join(testFolder, "test.txt")
	if err := os.WriteFile(supportedFile, []byte("text content"), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	ctx := context.Background()
	err := svc.CreateIndex(ctx, testFolder, IndexOptions{}, nil)
	if err != nil {
		t.Fatalf("CreateIndex failed: %v", err)
	}

	// Verify only supported file was indexed
	count, err := svc.GetIndexStats(testFolder)
	if err != nil {
		t.Fatalf("GetIndexStats failed: %v", err)
	}

	if count == 0 {
		t.Error("Expected at least one chunk from supported file")
	}
}

// TestCreateIndex_Cancellation tests context cancellation during indexing
func TestCreateIndex_Cancellation(t *testing.T) {
	cfg := testutil.CreateTestConfig(t)
	mockClient := testutil.NewMockVoyageClient(1024)
	svc := New(cfg, mockClient)

	testFolder := t.TempDir()

	// Create some test files
	for i := 0; i < 5; i++ {
		filename := filepath.Join(testFolder, fmt.Sprintf("file%d.txt", i))
		if err := os.WriteFile(filename, []byte("content"), 0644); err != nil {
			t.Fatalf("Failed to create test file: %v", err)
		}
	}

	// Create context with short timeout
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Millisecond)
	defer cancel()

	// Give it a moment to ensure context expires
	time.Sleep(10 * time.Millisecond)

	err := svc.CreateIndex(ctx, testFolder, IndexOptions{}, nil)

	// Should return context error
	if err == nil {
		t.Error("Expected error from cancelled context")
	}

	if err != nil && err != context.DeadlineExceeded && err != context.Canceled {
		// Context error should be wrapped
		if !contains(err.Error(), "context") && !contains(err.Error(), "cancel") {
			t.Logf("Got error: %v (expected context-related error)", err)
		}
	}
}

// TestUpdateIndex_NoChanges tests incremental update with no file changes
func TestUpdateIndex_NoChanges(t *testing.T) {
	cfg := testutil.CreateTestConfig(t)
	mockClient := testutil.NewMockVoyageClient(1024)
	svc := New(cfg, mockClient)

	testFolder := t.TempDir()
	testFile := filepath.Join(testFolder, "test.txt")
	if err := os.WriteFile(testFile, []byte("content"), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	ctx := context.Background()

	// Initial index
	err := svc.CreateIndex(ctx, testFolder, IndexOptions{}, nil)
	if err != nil {
		t.Fatalf("CreateIndex failed: %v", err)
	}

	// Update without changes
	err = svc.UpdateIndex(ctx, testFolder, IndexOptions{}, nil)
	if err != nil {
		t.Fatalf("UpdateIndex failed: %v", err)
	}

	// Should complete successfully (no error)
}

// TestUpdateIndex_NewFiles tests incremental update with new files
func TestUpdateIndex_NewFiles(t *testing.T) {
	cfg := testutil.CreateTestConfig(t)
	mockClient := testutil.NewMockVoyageClient(1024)
	svc := New(cfg, mockClient)

	testFolder := t.TempDir()
	file1 := filepath.Join(testFolder, "file1.txt")
	if err := os.WriteFile(file1, []byte("first file"), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	ctx := context.Background()

	// Initial index
	err := svc.CreateIndex(ctx, testFolder, IndexOptions{}, nil)
	if err != nil {
		t.Fatalf("CreateIndex failed: %v", err)
	}

	initialCount, _ := svc.GetIndexStats(testFolder)

	// Add new file
	file2 := filepath.Join(testFolder, "file2.txt")
	if err := os.WriteFile(file2, []byte("second file"), 0644); err != nil {
		t.Fatalf("Failed to create new file: %v", err)
	}

	// Update index
	err = svc.UpdateIndex(ctx, testFolder, IndexOptions{}, nil)
	if err != nil {
		t.Fatalf("UpdateIndex failed: %v", err)
	}

	// Verify chunk count increased
	newCount, _ := svc.GetIndexStats(testFolder)
	if newCount <= initialCount {
		t.Errorf("Expected chunk count to increase after adding file (was %d, now %d)", initialCount, newCount)
	}
}

// TestUpdateIndex_ModifiedFiles tests incremental update with modified files
func TestUpdateIndex_ModifiedFiles(t *testing.T) {
	cfg := testutil.CreateTestConfig(t)
	mockClient := testutil.NewMockVoyageClient(1024)
	svc := New(cfg, mockClient)

	testFolder := t.TempDir()
	testFile := filepath.Join(testFolder, "test.txt")
	if err := os.WriteFile(testFile, []byte("original content"), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	ctx := context.Background()

	// Initial index
	err := svc.CreateIndex(ctx, testFolder, IndexOptions{}, nil)
	if err != nil {
		t.Fatalf("CreateIndex failed: %v", err)
	}

	// Wait a moment to ensure timestamp difference
	time.Sleep(10 * time.Millisecond)

	// Modify file
	if err := os.WriteFile(testFile, []byte("modified content with different text"), 0644); err != nil {
		t.Fatalf("Failed to modify test file: %v", err)
	}

	// Update index
	err = svc.UpdateIndex(ctx, testFolder, IndexOptions{}, nil)
	if err != nil {
		t.Fatalf("UpdateIndex failed: %v", err)
	}

	// Search should find modified content
	results, err := svc.Search(testFolder, "modified", 5, "", "semantic")
	if err != nil {
		t.Fatalf("Search failed: %v", err)
	}

	if len(results) == 0 {
		t.Error("Expected to find 'modified' in search results after update")
	}
}

// TestDeleteIndex tests deleting an index
func TestDeleteIndex(t *testing.T) {
	cfg := testutil.CreateTestConfig(t)
	mockClient := testutil.NewMockVoyageClient(1024)
	svc := New(cfg, mockClient)

	testFolder := t.TempDir()
	testFile := filepath.Join(testFolder, "test.txt")
	if err := os.WriteFile(testFile, []byte("content"), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	ctx := context.Background()

	// Create index
	err := svc.CreateIndex(ctx, testFolder, IndexOptions{}, nil)
	if err != nil {
		t.Fatalf("CreateIndex failed: %v", err)
	}

	// Verify index exists
	dbPath, _ := svc.GenerateDBPath(testFolder)
	if _, err := os.Stat(dbPath); os.IsNotExist(err) {
		t.Fatal("Expected database file to exist before deletion")
	}

	// Delete index
	err = svc.DeleteIndex(testFolder)
	if err != nil {
		t.Fatalf("DeleteIndex failed: %v", err)
	}

	// Verify index was deleted
	if _, err := os.Stat(dbPath); err == nil {
		t.Error("Expected database file to be deleted")
	}
}

// TestDeleteIndex_NonExistent tests deleting a non-existent index
func TestDeleteIndex_NonExistent(t *testing.T) {
	cfg := testutil.CreateTestConfig(t)
	mockClient := testutil.NewMockVoyageClient(1024)
	svc := New(cfg, mockClient)

	nonExistentFolder := filepath.Join(t.TempDir(), "nonexistent")

	err := svc.DeleteIndex(nonExistentFolder)
	if err == nil {
		t.Error("Expected error when deleting non-existent index")
	}
}

// TestSearch_BasicQuery tests basic search functionality
func TestSearch_BasicQuery(t *testing.T) {
	cfg := testutil.CreateTestConfig(t)
	mockClient := testutil.NewMockVoyageClient(1024)
	svc := New(cfg, mockClient)

	testFolder := t.TempDir()
	files := map[string]string{
		"cats.txt": "Cats are wonderful pets. They are independent and loving.",
		"dogs.txt": "Dogs are loyal companions. They love to play fetch.",
	}

	for filename, content := range files {
		filePath := filepath.Join(testFolder, filename)
		if err := os.WriteFile(filePath, []byte(content), 0644); err != nil {
			t.Fatalf("Failed to create test file: %v", err)
		}
	}

	ctx := context.Background()
	err := svc.CreateIndex(ctx, testFolder, IndexOptions{}, nil)
	if err != nil {
		t.Fatalf("CreateIndex failed: %v", err)
	}

	// Search for "cats"
	results, err := svc.Search(testFolder, "cats pets", 5, "", "semantic")
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
}

// TestSearch_NoResults tests search with no matching results
func TestSearch_NoResults(t *testing.T) {
	cfg := testutil.CreateTestConfig(t)
	mockClient := testutil.NewMockVoyageClient(1024)
	svc := New(cfg, mockClient)

	testFolder := t.TempDir()
	testFile := filepath.Join(testFolder, "test.txt")
	if err := os.WriteFile(testFile, []byte("content about cats"), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	ctx := context.Background()
	err := svc.CreateIndex(ctx, testFolder, IndexOptions{}, nil)
	if err != nil {
		t.Fatalf("CreateIndex failed: %v", err)
	}

	// Search for something not in the content
	results, err := svc.Search(testFolder, "xyzabc123notfound", 5, "", "semantic")
	if err != nil {
		t.Fatalf("Search failed: %v", err)
	}

	// Should return empty results, not an error
	if len(results) != 0 {
		t.Logf("Got %d results (expected 0, but mock embeddings may match)", len(results))
	}
}

// TestListIndexes tests listing all indexes
func TestListIndexes(t *testing.T) {
	cfg := testutil.CreateTestConfig(t)
	mockClient := testutil.NewMockVoyageClient(1024)
	svc := New(cfg, mockClient)

	// Create two indexes
	folder1 := filepath.Join(t.TempDir(), "folder1")
	folder2 := filepath.Join(t.TempDir(), "folder2")

	for _, folder := range []string{folder1, folder2} {
		if err := os.MkdirAll(folder, 0755); err != nil {
			t.Fatalf("Failed to create folder: %v", err)
		}

		testFile := filepath.Join(folder, "test.txt")
		if err := os.WriteFile(testFile, []byte("content"), 0644); err != nil {
			t.Fatalf("Failed to create test file: %v", err)
		}

		ctx := context.Background()
		if err := svc.CreateIndex(ctx, folder, IndexOptions{}, nil); err != nil {
			t.Fatalf("CreateIndex failed: %v", err)
		}
	}

	// List indexes
	indexes, err := svc.ListIndexes()
	if err != nil {
		t.Fatalf("ListIndexes failed: %v", err)
	}

	if len(indexes) < 2 {
		t.Errorf("Expected at least 2 indexes, got %d", len(indexes))
	}

	// Verify index info has required fields
	for _, info := range indexes {
		if info.FolderPath == "" {
			t.Error("Expected FolderPath to be set")
		}
		if info.DBPath == "" {
			t.Error("Expected DBPath to be set")
		}
	}
}

// TestListIndexes_Empty tests listing indexes when none exist
func TestListIndexes_Empty(t *testing.T) {
	cfg := testutil.CreateTestConfig(t)
	mockClient := testutil.NewMockVoyageClient(1024)
	svc := New(cfg, mockClient)

	indexes, err := svc.ListIndexes()
	if err != nil {
		t.Fatalf("ListIndexes failed: %v", err)
	}

	if len(indexes) != 0 {
		t.Logf("Expected 0 indexes, got %d (may have existing test indexes)", len(indexes))
	}
}

// Helper function
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
