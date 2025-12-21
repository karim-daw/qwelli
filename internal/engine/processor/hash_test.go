package processor

import (
	"os"
	"path/filepath"
	"testing"
)

func TestComputeSHA256(t *testing.T) {
	tmpFile := filepath.Join(t.TempDir(), "test.txt")
	content := "test content"
	if err := os.WriteFile(tmpFile, []byte(content), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	hash1, err := ComputeSHA256(tmpFile)
	if err != nil {
		t.Fatalf("ComputeSHA256() error = %v", err)
	}

	if len(hash1) != 64 { // SHA256 hex string length
		t.Errorf("hash length = %d, want 64", len(hash1))
	}

	// Same file should produce same hash
	hash2, err := ComputeSHA256(tmpFile)
	if err != nil {
		t.Fatalf("ComputeSHA256() second call error = %v", err)
	}

	if hash1 != hash2 {
		t.Error("Hash should be deterministic")
	}
}

func TestComputeSHA256_ModifiedFile(t *testing.T) {
	tmpFile := filepath.Join(t.TempDir(), "test.txt")
	if err := os.WriteFile(tmpFile, []byte("original"), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	hash1, err := ComputeSHA256(tmpFile)
	if err != nil {
		t.Fatalf("ComputeSHA256() error = %v", err)
	}

	// Modify file
	if err := os.WriteFile(tmpFile, []byte("modified"), 0644); err != nil {
		t.Fatalf("Failed to modify test file: %v", err)
	}

	hash2, err := ComputeSHA256(tmpFile)
	if err != nil {
		t.Fatalf("ComputeSHA256() after modification error = %v", err)
	}

	if hash1 == hash2 {
		t.Error("Hash should change when file content changes")
	}
}

func TestComputeSHA256_NonexistentFile(t *testing.T) {
	_, err := ComputeSHA256("/nonexistent/file.txt")
	if err == nil {
		t.Error("Expected error for nonexistent file")
	}
}
