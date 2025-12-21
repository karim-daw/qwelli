package scanner

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/karim-daw/qwelli/internal/db"
	"github.com/karim-daw/qwelli/internal/engine/processor"
)

// ChangeSet represents files that need to be added, updated, or deleted
type ChangeSet struct {
	ToAdd    []string
	ToUpdate []string
	ToDelete []string
}

// IndexStatus represents the status of an index with pending changes
type IndexStatus struct {
	ToAdd    []FileStatus // New files not yet indexed
	ToUpdate []FileStatus // Changed files needing re-index
	ToDelete []FileStatus // Files deleted from filesystem but still in DB
	Total    int          // Total files in index
	UpToDate int          // Files that are current
}

// FileStatus represents a file with its status information
type FileStatus struct {
	Path       string
	FileType   string
	Size       int64
	ModifiedAt time.Time
	Reason     string // "new", "modified", "deleted"
}

// DetectChanges compares filesystem state with database state to identify changes
func DetectChanges(projectDB *db.ProjectDB, folderPath string) (*ChangeSet, error) {
	// Get all files from database
	dbFiles, err := projectDB.GetAllFiles()
	if err != nil {
		return nil, fmt.Errorf("failed to get files from database: %w", err)
	}

	// Build map of database files by path
	dbFileMap := make(map[string]*db.File)
	for i := range dbFiles {
		dbFileMap[dbFiles[i].Path] = &dbFiles[i]
	}

	// Scan filesystem
	fsFiles, err := ScanFolder(folderPath)
	if err != nil {
		return nil, fmt.Errorf("failed to scan folder: %w", err)
	}

	changes := &ChangeSet{
		ToAdd:    []string{},
		ToUpdate: []string{},
		ToDelete: []string{},
	}

	// Track which files we've seen in filesystem
	seenInFS := make(map[string]bool)

	// Check each filesystem file
	for _, fsPath := range fsFiles {
		seenInFS[fsPath] = true

		// Normalize path to absolute
		absPath, err := filepath.Abs(fsPath)
		if err != nil {
			continue
		}

		dbFile, exists := dbFileMap[absPath]
		if !exists {
			// New file
			changes.ToAdd = append(changes.ToAdd, absPath)
			continue
		}

		// Check if file changed (compare size first, then hash if needed)
		info, err := os.Stat(absPath)
		if err != nil {
			// File might have been deleted between scan and stat
			continue
		}

		// Quick check: size changed
		if info.Size() != dbFile.Size {
			changes.ToUpdate = append(changes.ToUpdate, absPath)
			continue
		}

		// Size same, check modification time
		if !info.ModTime().Equal(dbFile.ModifiedAt) {
			// mtime different, compute hash to verify content change
			currentHash, err := processor.ComputeSHA256(absPath)
			if err != nil {
				// If we can't compute hash, assume changed
				changes.ToUpdate = append(changes.ToUpdate, absPath)
				continue
			}

			if currentHash != dbFile.FileHash {
				changes.ToUpdate = append(changes.ToUpdate, absPath)
			}
		}
	}

	// Find deleted files (in DB but not in filesystem)
	for path := range dbFileMap {
		if !seenInFS[path] {
			changes.ToDelete = append(changes.ToDelete, path)
		}
	}

	return changes, nil
}
