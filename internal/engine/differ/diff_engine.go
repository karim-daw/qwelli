package differ

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"runtime"
	"sync"
	"time"

	"github.com/karim-daw/qwelli/internal/db"
	"github.com/karim-daw/qwelli/internal/engine/extraction"
)

// ChangeSet represents files that need to be added, updated, or deleted
type ChangeSet struct {
	ToAdd    []string
	ToUpdate []string
	ToDelete []string
}

// IndexStatus represents the status of an index with pending changes
type IndexStatus struct {
	ToAdd    []FileStatus `json:"toAdd"`    // New files not yet indexed
	ToUpdate []FileStatus `json:"toUpdate"` // Changed files needing re-index
	ToDelete []FileStatus `json:"toDelete"` // Files deleted from filesystem but still in DB
	Total    int          `json:"total"`    // Total files in index
	UpToDate int          `json:"upToDate"` // Files that are current
}

// FileStatus represents a file with its status information
type FileStatus struct {
	Path       string    `json:"path"`
	FileType   string    `json:"fileType"`
	Size       int64     `json:"size"`
	ModifiedAt time.Time `json:"modifiedAt"`
	Reason     string    `json:"reason"` // "new", "modified", "deleted"
}

// DetectChanges compares filesystem state with database state to identify changes
func DetectChanges(projectDB *db.ProjectDB, folderPath string) (*ChangeSet, error) {
	startTime := time.Now()
	ctx := context.Background()

	// Get all files from database
	dbFiles, err := projectDB.GetAllFiles(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get files from database: %w", err)
	}

	log.Printf("🔍 DetectChanges: Found %d files in database for folder: %s", len(dbFiles), folderPath)

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

	log.Printf("🔍 DetectChanges: Found %d files in filesystem", len(fsFiles))

	changes := &ChangeSet{
		ToAdd:    []string{},
		ToUpdate: []string{},
		ToDelete: []string{},
	}

	// Track which files we've seen in filesystem
	seenInFS := make(map[string]bool)

	// Use parallel processing for large file sets
	numWorkers := runtime.NumCPU()
	if len(fsFiles) < 100 {
		numWorkers = 1 // Don't parallelize for small sets
	}

	// Channels for parallel processing
	type fileCheckResult struct {
		absPath string
		status  string // "new", "updated", "unchanged", "error"
		oldSize int64
		newSize int64
	}

	fileChan := make(chan string, len(fsFiles))
	resultChan := make(chan fileCheckResult, len(fsFiles))

	// Start workers
	var wg sync.WaitGroup
	for i := 0; i < numWorkers; i++ {
		wg.Go(func() {
			for fsPath := range fileChan {
				// Normalize path to absolute
				absPath, err := filepath.Abs(fsPath)
				if err != nil {
					continue
				}

				dbFile, exists := dbFileMap[absPath]
				if !exists {
					// New file
					resultChan <- fileCheckResult{absPath: absPath, status: "new"}
					continue
				}

				// Check if file changed
				info, err := os.Stat(absPath)
				if err != nil {
					resultChan <- fileCheckResult{absPath: absPath, status: "error"}
					continue
				}

				// Quick check: size changed
				if info.Size() != dbFile.Size {
					resultChan <- fileCheckResult{
						absPath: absPath,
						status:  "updated",
						oldSize: dbFile.Size,
						newSize: info.Size(),
					}
					continue
				}

				// Size same - check modification time first (optimization)
				// If mtime is exactly the same, file likely unchanged
				if info.ModTime().Equal(dbFile.ModifiedAt) {
					resultChan <- fileCheckResult{absPath: absPath, status: "unchanged"}
					continue
				}

				// Mtime changed - verify content hash to be sure
				currentHash, err := extraction.ComputeSHA256(absPath)
				if err != nil {
					log.Printf("⚠️  Failed to compute hash for %s: %v", filepath.Base(absPath), err)
					resultChan <- fileCheckResult{absPath: absPath, status: "updated"}
					continue
				}

				if currentHash != dbFile.FileHash {
					log.Printf("📝 File modified (hash changed): %s", filepath.Base(absPath))
					resultChan <- fileCheckResult{absPath: absPath, status: "updated"}
					continue
				}

				// Hash same despite mtime change (e.g., touch command)
				resultChan <- fileCheckResult{absPath: absPath, status: "unchanged"}
			}
		})
	}

	// Send files to workers
	for _, fsPath := range fsFiles {
		fileChan <- fsPath
	}
	close(fileChan)

	// Close result channel when all workers done
	go func() {
		wg.Wait()
		close(resultChan)
	}()

	// Collect results
	for result := range resultChan {
		seenInFS[result.absPath] = true

		switch result.status {
		case "new":
			changes.ToAdd = append(changes.ToAdd, result.absPath)
		case "updated":
			if result.newSize > 0 {
				log.Printf("📝 File modified (size change): %s (old: %d, new: %d)",
					result.absPath, result.oldSize, result.newSize)
			} else {
				log.Printf("📝 File modified (hash change): %s", result.absPath)
			}
			changes.ToUpdate = append(changes.ToUpdate, result.absPath)
		}
	}

	// Find deleted files (in DB but not in filesystem)
	for path := range dbFileMap {
		if !seenInFS[path] {
			changes.ToDelete = append(changes.ToDelete, path)
		}
	}

	elapsed := time.Since(startTime)
	log.Printf("📊 DetectChanges Results: ToAdd=%d, ToUpdate=%d, ToDelete=%d (took %v)",
		len(changes.ToAdd), len(changes.ToUpdate), len(changes.ToDelete), elapsed)

	return changes, nil
}
