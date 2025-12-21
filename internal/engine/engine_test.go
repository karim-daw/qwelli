package engine

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/karim-daw/qwelli/internal/db"
	"github.com/karim-daw/qwelli/internal/processor"
)

func TestDetectChanges(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	pdb, err := db.OpenProjectDB(dbPath, 4)
	if err != nil {
		t.Fatalf("OpenProjectDB() error = %v", err)
	}
	defer pdb.Close()

	// Create test files
	file1 := filepath.Join(tmpDir, "file1.txt")
	file2 := filepath.Join(tmpDir, "file2.txt")
	if err := os.WriteFile(file1, []byte("content1"), 0644); err != nil {
		t.Fatalf("Failed to create file1: %v", err)
	}
	if err := os.WriteFile(file2, []byte("content2"), 0644); err != nil {
		t.Fatalf("Failed to create file2: %v", err)
	}

	// Get absolute paths
	absFile1, _ := filepath.Abs(file1)
	absFile2, _ := filepath.Abs(file2)

	// Index file1 only
	hash1, _ := processor.ComputeSHA256(absFile1)
	info1, _ := os.Stat(absFile1)
	file := db.File{
		FileID:     "f1",
		Path:       absFile1,
		FileType:   "txt",
		FileHash:   hash1,
		ModifiedAt: info1.ModTime(),
		Size:       info1.Size(),
		IndexedAt:  time.Now(),
	}
	if err := pdb.InsertFile(file); err != nil {
		t.Fatalf("InsertFile() error = %v", err)
	}

	// Detect changes
	changes, err := detectChanges(pdb, tmpDir)
	if err != nil {
		t.Fatalf("detectChanges() error = %v", err)
	}

	// file2 should be in ToAdd
	if len(changes.ToAdd) != 1 {
		t.Errorf("ToAdd = %d, want 1", len(changes.ToAdd))
	}
	if len(changes.ToAdd) > 0 && changes.ToAdd[0] != absFile2 {
		t.Errorf("ToAdd[0] = %v, want %v", changes.ToAdd[0], absFile2)
	}

	// file1 should not be in ToUpdate or ToAdd
	if len(changes.ToUpdate) != 0 {
		t.Errorf("ToUpdate = %d, want 0", len(changes.ToUpdate))
	}
}

func TestDetectChanges_ModifiedFile(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	pdb, err := db.OpenProjectDB(dbPath, 4)
	if err != nil {
		t.Fatalf("OpenProjectDB() error = %v", err)
	}
	defer pdb.Close()

	testFile := filepath.Join(tmpDir, "test.txt")
	if err := os.WriteFile(testFile, []byte("original"), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	absFile, _ := filepath.Abs(testFile)

	// Index file
	hash1, _ := processor.ComputeSHA256(absFile)
	info1, _ := os.Stat(absFile)
	file := db.File{
		FileID:     "f1",
		Path:       absFile,
		FileType:   "txt",
		FileHash:   hash1,
		ModifiedAt: info1.ModTime(),
		Size:       info1.Size(),
		IndexedAt:  time.Now(),
	}
	if err := pdb.InsertFile(file); err != nil {
		t.Fatalf("InsertFile() error = %v", err)
	}

	// Modify file
	time.Sleep(10 * time.Millisecond) // Ensure mtime changes
	if err := os.WriteFile(testFile, []byte("modified"), 0644); err != nil {
		t.Fatalf("Failed to modify test file: %v", err)
	}

	// Detect changes
	changes, err := detectChanges(pdb, tmpDir)
	if err != nil {
		t.Fatalf("DetectChanges() error = %v", err)
	}

	if len(changes.ToUpdate) != 1 {
		t.Errorf("ToUpdate = %d, want 1", len(changes.ToUpdate))
	}
	if len(changes.ToUpdate) > 0 && changes.ToUpdate[0] != absFile {
		t.Errorf("ToUpdate[0] = %v, want %v", changes.ToUpdate[0], absFile)
	}
}

func TestDetectChanges_DeletedFile(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	pdb, err := db.OpenProjectDB(dbPath, 4)
	if err != nil {
		t.Fatalf("OpenProjectDB() error = %v", err)
	}
	defer pdb.Close()

	testFile := filepath.Join(tmpDir, "test.txt")
	if err := os.WriteFile(testFile, []byte("content"), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	absFile, _ := filepath.Abs(testFile)

	// Index file
	hash1, _ := processor.ComputeSHA256(absFile)
	info1, _ := os.Stat(absFile)
	file := db.File{
		FileID:     "f1",
		Path:       absFile,
		FileType:   "txt",
		FileHash:   hash1,
		ModifiedAt: info1.ModTime(),
		Size:       info1.Size(),
		IndexedAt:  time.Now(),
	}
	if err := pdb.InsertFile(file); err != nil {
		t.Fatalf("InsertFile() error = %v", err)
	}

	// Delete file
	if err := os.Remove(testFile); err != nil {
		t.Fatalf("Failed to delete test file: %v", err)
	}

	// Detect changes
	changes, err := detectChanges(pdb, tmpDir)
	if err != nil {
		t.Fatalf("DetectChanges() error = %v", err)
	}

	if len(changes.ToDelete) != 1 {
		t.Errorf("ToDelete = %d, want 1", len(changes.ToDelete))
	}
	if len(changes.ToDelete) > 0 && changes.ToDelete[0] != absFile {
		t.Errorf("ToDelete[0] = %v, want %v", changes.ToDelete[0], absFile)
	}
}
