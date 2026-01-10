package extraction

import (
	"crypto/sha256"
	"fmt"
	"io"
	"os"
)

// ComputeSHA256 computes the SHA256 hash of a file's content
func ComputeSHA256(filePath string) (string, error) {
	file, err := os.Open(filePath)
	if err != nil {
		// Check if it's an I/O error (common with OneDrive Files On-Demand)
		if isIOError(err) {
			return "", fmt.Errorf("file unavailable (OneDrive placeholder or I/O error): %w", err)
		}
		return "", fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	hash := sha256.New()
	if _, err := io.Copy(hash, file); err != nil {
		// Check if it's an I/O error during reading
		if isIOError(err) {
			return "", fmt.Errorf("file unavailable (OneDrive placeholder or I/O error): %w", err)
		}
		return "", fmt.Errorf("failed to read file: %w", err)
	}

	return fmt.Sprintf("%x", hash.Sum(nil)), nil
}

// isIOError checks if an error is an input/output error
func isIOError(err error) bool {
	return err != nil && (
	// Check for common I/O error messages
	containsString(err.Error(), "input/output error") ||
		containsString(err.Error(), "i/o error") ||
		containsString(err.Error(), "invalid argument"))
}

func containsString(s, substr string) bool {
	return len(s) >= len(substr) &&
		(s == substr ||
			(len(s) > len(substr) &&
				(s[:len(substr)] == substr ||
					s[len(s)-len(substr):] == substr ||
					indexOf(s, substr) >= 0)))
}

func indexOf(s, substr string) int {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return i
		}
	}
	return -1
}
