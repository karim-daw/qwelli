package differ

import (
	"crypto/md5"
	"encoding/hex"
	"io/fs"
	"path/filepath"
	"strings"

	"github.com/karim-daw/qwelli/internal/engine/chunker"
	"github.com/karim-daw/qwelli/internal/engine/fileprocessor"
)

// ScanFolder scans a folder and returns all supported file paths
func ScanFolder(root string) ([]string, error) {
	var files []string
	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			name := d.Name()
			if strings.HasPrefix(name, ".") || name == "node_modules" || name == "vendor" {
				return filepath.SkipDir
			}
			return nil
		}
		if strings.HasPrefix(d.Name(), ".") || !IsTextFile(path) {
			return nil
		}
		files = append(files, path)
		return nil
	})
	return files, err
}

// IsTextFile returns true if the file path points to a supported file type
func IsTextFile(path string) bool {
	fileType := GetFileTypeFromPath(path)
	return fileprocessor.IsSupported(fileType)
}

// GenerateFileID generates a unique file ID from a file path
func GenerateFileID(path string) string {
	hash := md5.Sum([]byte(path))
	return hex.EncodeToString(hash[:])
}

// GenerateChunkID delegates to chunker package to avoid duplication
func GenerateChunkID(fileID string, chunkIndex int) string {
	return chunker.GenerateChunkID(fileID, chunkIndex)
}

// GetFileTypeFromPath extracts the file type from a file path
func GetFileTypeFromPath(path string) string {
	ext := strings.ToLower(filepath.Ext(path))
	if ext == "" {
		return "unknown"
	}
	return ext[1:]
}
