package differ

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/karim-daw/qwelli/internal/db"
)

// TestDetectChanges_NewFiles tests detection of new files
func TestDetectChanges_NewFiles(t *testing.T) {
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "test.db")

	// Create database
	projectDB, err := db.OpenProjectDB(dbPath, 1024)
	if err != nil {
		t.Fatalf("Failed to create database: %v", err)
	}
	defer projectDB.Close()

	// Create test files
	testFolder := filepath.Join(tempDir, "test_folder")
	os.MkdirAll(testFolder, 0755)

	file1 := filepath.Join(testFolder, "new1.txt")
	file2 := filepath.Join(testFolder, "new2.txt")
	os.WriteFile(file1, []byte("content1"), 0644)
	os.WriteFile(file2, []byte("content2"), 0644)

	// Detect changes
	changes, err := DetectChanges(projectDB, testFolder)
	if err != nil {
		t.Fatalf("DetectChanges failed: %v", err)
	}

	if len(changes.ToAdd) != 2 {
		t.Errorf("Expected 2 files to add, got %d", len(changes.ToAdd))
	}
	if len(changes.ToUpdate) != 0 {
		t.Errorf("Expected 0 files to update, got %d", len(changes.ToUpdate))
	}
	if len(changes.ToDelete) != 0 {
		t.Errorf("Expected 0 files to delete, got %d", len(changes.ToDelete))
	}
}

// TestDetectChanges_ModifiedFiles tests detection of modified files
func TestDetectChanges_ModifiedFiles(t *testing.T) {
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "test.db")

	projectDB, err := db.OpenProjectDB(dbPath, 1024)
	if err != nil {
		t.Fatalf("Failed to create database: %v", err)
	}
	defer projectDB.Close()

	testFolder := filepath.Join(tempDir, "test_folder")
	os.MkdirAll(testFolder, 0755)

	file1 := filepath.Join(testFolder, "file1.txt")
	os.WriteFile(file1, []byte("original content"), 0644)

	// Index the file
	absPath, _ := filepath.Abs(file1)
	stat, _ := os.Stat(file1)
	projectDB.InsertFile(db.File{
		FileID:     GenerateFileID(absPath),
		Path:       absPath,
		FileHash:   "old-hash",
		Size:       16,
		ModifiedAt: stat.ModTime(),
	})

	// Wait and modify file
	time.Sleep(10 * time.Millisecond)
	os.WriteFile(file1, []byte("modified content with more text"), 0644)

	// Detect changes
	changes, err := DetectChanges(projectDB, testFolder)
	if err != nil {
		t.Fatalf("DetectChanges failed: %v", err)
	}

	if len(changes.ToUpdate) == 0 {
		t.Error("Expected at least 1 file to update")
	}
	if len(changes.ToAdd) != 0 {
		t.Errorf("Expected 0 files to add, got %d", len(changes.ToAdd))
	}
}

// TestDetectChanges_DeletedFiles tests detection of deleted files
func TestDetectChanges_DeletedFiles(t *testing.T) {
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "test.db")

	projectDB, err := db.OpenProjectDB(dbPath, 1024)
	if err != nil {
		t.Fatalf("Failed to create database: %v", err)
	}
	defer projectDB.Close()

	testFolder := filepath.Join(tempDir, "test_folder")
	os.MkdirAll(testFolder, 0755)

	// Index a file that doesn't exist in filesystem
	deletedPath := filepath.Join(testFolder, "deleted.txt")
	projectDB.InsertFile(db.File{
		FileID:   GenerateFileID(deletedPath),
		Path:     deletedPath,
		FileHash: "some-hash",
		Size:     100,
	})

	// Detect changes
	changes, err := DetectChanges(projectDB, testFolder)
	if err != nil {
		t.Fatalf("DetectChanges failed: %v", err)
	}

	if len(changes.ToDelete) != 1 {
		t.Errorf("Expected 1 file to delete, got %d", len(changes.ToDelete))
	}
}

// TestDetectChanges_EmptyFolder tests detecting changes in an empty folder
func TestDetectChanges_EmptyFolder(t *testing.T) {
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "test.db")

	projectDB, err := db.OpenProjectDB(dbPath, 1024)
	if err != nil {
		t.Fatalf("Failed to create database: %v", err)
	}
	defer projectDB.Close()

	testFolder := filepath.Join(tempDir, "test_folder")
	os.MkdirAll(testFolder, 0755)

	// Detect changes in empty folder
	changes, err := DetectChanges(projectDB, testFolder)
	if err != nil {
		t.Fatalf("DetectChanges failed: %v", err)
	}

	if len(changes.ToAdd) != 0 || len(changes.ToUpdate) != 0 || len(changes.ToDelete) != 0 {
		t.Errorf("Expected no changes in empty folder, got: ToAdd=%d, ToUpdate=%d, ToDelete=%d",
			len(changes.ToAdd), len(changes.ToUpdate), len(changes.ToDelete))
	}
}

// TestScanFolder_SupportsFiltering tests folder scanning with file filtering
func TestScanFolder_SupportsFiltering(t *testing.T) {
	testFolder := t.TempDir()

	// Create various files
	os.WriteFile(filepath.Join(testFolder, "test.txt"), []byte("text"), 0644)
	os.WriteFile(filepath.Join(testFolder, "test.md"), []byte("markdown"), 0644)
	os.WriteFile(filepath.Join(testFolder, "test.go"), []byte("code"), 0644)
	os.WriteFile(filepath.Join(testFolder, "test.exe"), []byte("binary"), 0644) // Not supported

	files, err := ScanFolder(testFolder)
	if err != nil {
		t.Fatalf("ScanFolder failed: %v", err)
	}

	if len(files) < 3 {
		t.Errorf("Expected at least 3 supported files, got %d", len(files))
	}

	// Verify .exe was not included
	for _, file := range files {
		if filepath.Ext(file) == ".exe" {
			t.Error("Unsupported .exe file should not be included")
		}
	}
}

// TestScanFolder_IgnoresHidden tests that hidden files are ignored
func TestScanFolder_IgnoresHidden(t *testing.T) {
	testFolder := t.TempDir()

	os.WriteFile(filepath.Join(testFolder, "visible.txt"), []byte("text"), 0644)
	os.WriteFile(filepath.Join(testFolder, ".hidden.txt"), []byte("hidden"), 0644)

	files, err := ScanFolder(testFolder)
	if err != nil {
		t.Fatalf("ScanFolder failed: %v", err)
	}

	for _, file := range files {
		if filepath.Base(file) == ".hidden.txt" {
			t.Error("Hidden file should not be included")
		}
	}
}

// TestScanFolder_IgnoresDirectories tests that certain directories are skipped
func TestScanFolder_IgnoresDirectories(t *testing.T) {
	testFolder := t.TempDir()

	// Create ignored directories
	os.MkdirAll(filepath.Join(testFolder, "node_modules"), 0755)
	os.MkdirAll(filepath.Join(testFolder, "vendor"), 0755)
	os.MkdirAll(filepath.Join(testFolder, ".git"), 0755)

	// Add files in ignored directories
	os.WriteFile(filepath.Join(testFolder, "node_modules", "package.json"), []byte("{}"), 0644)
	os.WriteFile(filepath.Join(testFolder, "vendor", "lib.go"), []byte("code"), 0644)
	os.WriteFile(filepath.Join(testFolder, ".git", "config"), []byte("git"), 0644)

	// Add files in root
	os.WriteFile(filepath.Join(testFolder, "main.go"), []byte("code"), 0644)

	files, err := ScanFolder(testFolder)
	if err != nil {
		t.Fatalf("ScanFolder failed: %v", err)
	}

	// Should only find main.go
	if len(files) != 1 {
		t.Errorf("Expected 1 file, got %d", len(files))
	}
}

// TestScanFolder_NestedDirectories tests recursive directory scanning
func TestScanFolder_NestedDirectories(t *testing.T) {
	testFolder := t.TempDir()

	// Create nested structure
	os.MkdirAll(filepath.Join(testFolder, "src", "components"), 0755)
	os.WriteFile(filepath.Join(testFolder, "main.go"), []byte("main"), 0644)
	os.WriteFile(filepath.Join(testFolder, "src", "app.go"), []byte("app"), 0644)
	os.WriteFile(filepath.Join(testFolder, "src", "components", "button.go"), []byte("btn"), 0644)

	files, err := ScanFolder(testFolder)
	if err != nil {
		t.Fatalf("ScanFolder failed: %v", err)
	}

	if len(files) != 3 {
		t.Errorf("Expected 3 files, got %d", len(files))
	}
}

// TestIsTextFile tests file type detection
func TestIsTextFile(t *testing.T) {
	tests := []struct {
		path     string
		expected bool
	}{
		{"test.txt", true},
		{"test.md", true},
		{"test.go", true},
		{"test.py", true},
		{"test.exe", false},
		{"test.bin", false},
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			result := IsTextFile(tt.path)
			if result != tt.expected {
				t.Errorf("IsTextFile(%s) = %v, want %v", tt.path, result, tt.expected)
			}
		})
	}
}

// TestGenerateFileID tests file ID generation
func TestGenerateFileID(t *testing.T) {
	path1 := "/path/to/file.txt"
	path2 := "/path/to/file.txt"
	path3 := "/path/to/other.txt"

	id1 := GenerateFileID(path1)
	id2 := GenerateFileID(path2)
	id3 := GenerateFileID(path3)

	// Same path should generate same ID
	if id1 != id2 {
		t.Error("Same path should generate same file ID")
	}

	// Different paths should generate different IDs
	if id1 == id3 {
		t.Error("Different paths should generate different file IDs")
	}

	// ID should be non-empty
	if id1 == "" {
		t.Error("File ID should not be empty")
	}
}

// TestGenerateChunkID tests chunk ID generation
func TestGenerateChunkID(t *testing.T) {
	fileID := "test-file-id"
	id1 := GenerateChunkID(fileID, 0)
	id2 := GenerateChunkID(fileID, 1)

	if id1 == id2 {
		t.Error("Different chunk indices should generate different IDs")
	}

	if id1 == "" {
		t.Error("Chunk ID should not be empty")
	}
}

