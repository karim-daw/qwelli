package service

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/karim-daw/qwelli/internal/db"
	"github.com/karim-daw/qwelli/internal/engine"
	"github.com/karim-daw/qwelli/internal/engine/differ"
)

const statusCacheTTL = 60 * time.Second

// statusCacheEntry holds a cached GetIndexStatus result.
type statusCacheEntry struct {
	status   *differ.IndexStatus
	cachedAt time.Time
}

// IndexInfo represents metadata about an indexed folder.
type IndexInfo struct {
	FolderPath    string
	DBPath        string
	DocumentCount int
	SizeBytes     int64
	LastModified  time.Time
}

// IndexOptions contains options for indexing operations.
type IndexOptions struct {
	Incremental bool
	Multimodal  bool
}

// CreateIndex indexes a folder from scratch.
func (s *Service) CreateIndex(ctx context.Context, folderPath string, opts IndexOptions, progressCb func(int, int, string), phaseCb ...engine.PhaseCallback) error {
	return s.indexFolder(ctx, folderPath, false, progressCb, phaseCb...)
}

// UpdateIndex performs an incremental re-index.
func (s *Service) UpdateIndex(ctx context.Context, folderPath string, opts IndexOptions, progressCb func(int, int, string), phaseCb ...engine.PhaseCallback) error {
	return s.indexFolder(ctx, folderPath, true, progressCb, phaseCb...)
}

func (s *Service) indexFolder(ctx context.Context, folderPath string, incremental bool, progressCb func(int, int, string), phaseCb ...engine.PhaseCallback) error {
	absPath, err := filepath.Abs(folderPath)
	if err != nil {
		return fmt.Errorf("invalid path: %w", err)
	}
	if _, err := os.Stat(absPath); os.IsNotExist(err) {
		return fmt.Errorf("folder does not exist: %s", absPath)
	}

	dbPath, err := s.GenerateDBPath(absPath)
	if err != nil {
		return err
	}

	if incremental {
		if _, err := os.Stat(dbPath); os.IsNotExist(err) {
			return fmt.Errorf("index not found for %s — create index first", absPath)
		}
	}

	// Detect dimension & handle model changes
	dimension, err := s.engine.DetectDimension()
	if err != nil {
		return err
	}
	s.handleModelChange(dbPath, dimension)

	projectDB, err := db.OpenProjectDB(dbPath, dimension)
	if err != nil {
		return err
	}
	defer projectDB.Close()

	projectDB.SetMetadata("dimension", fmt.Sprintf("%d", dimension))
	projectDB.SetMetadata("model", s.engine.EmbeddingModel())
	projectDB.SetMetadata("folder_path", absPath)

	var phase engine.PhaseCallback
	if len(phaseCb) > 0 {
		phase = phaseCb[0]
	}
	if err := s.engine.IndexFolder(ctx, projectDB, absPath, incremental, progressCb, phase); err != nil {
		return err
	}
	// Refresh column statistics so the query planner makes optimal choices
	// on subsequent searches. Runs in <1s on typical indexes.
	_ = projectDB.AnalyzeStats()
	// Bust the status cache so the next GetIndexStatus call reflects the fresh index.
	s.invalidateStatusCache(absPath)
	return nil
}

// handleModelChange checks if the embedding model changed and recreates the DB if needed.
func (s *Service) handleModelChange(dbPath string, dimension int) {
	currentModel := s.engine.EmbeddingModel()
	if dim, err := db.GetDimensionFromDB(dbPath); err == nil {
		if tmpDB, _ := db.OpenProjectDB(dbPath, dim); tmpDB != nil {
			stored, _ := tmpDB.GetMetadata("model")
			tmpDB.Close()
			if stored != "" && stored != currentModel {
				log.Printf("⚠️  Model changed from '%s' to '%s' — recreating database...", stored, currentModel)
				os.Remove(dbPath)
			}
		}
	}
}

// DeleteIndex removes the database file for a folder.
func (s *Service) DeleteIndex(folderPath string) error {
	dbPath, err := s.GenerateDBPath(folderPath)
	if err != nil {
		return err
	}
	if _, err := os.Stat(dbPath); os.IsNotExist(err) {
		return fmt.Errorf("index not found for %s", folderPath)
	}
	return os.Remove(dbPath)
}

// Search performs a search across an indexed folder.
func (s *Service) Search(folderPath, query string, topK int, contentType, strategy string) ([]engine.SearchResult, error) {
	projectDB, err := s.OpenDB(folderPath)
	if err != nil {
		return nil, err
	}
	defer projectDB.Close()
	return s.engine.Search(projectDB, query, topK, contentType, strategy)
}

// SearchByDBPath performs a search using a direct DB path (used by server).
func (s *Service) SearchByDBPath(dbPath, query string, topK int, contentType, strategy string) ([]engine.SearchResult, error) {
	release := s.lockDB(dbPath)
	defer release()
	dim, err := db.GetDimensionFromDB(dbPath)
	if err != nil {
		return nil, err
	}
	projectDB, err := db.OpenProjectDB(dbPath, dim)
	if err != nil {
		return nil, err
	}
	defer projectDB.Close()
	return s.engine.Search(projectDB, query, topK, contentType, strategy)
}

// GetIndexStatus returns pending changes for an indexed folder.
// Results are cached for statusCacheTTL to avoid repeated expensive filesystem scans.
func (s *Service) GetIndexStatus(folderPath string) (*differ.IndexStatus, error) {
	absPath, err := filepath.Abs(folderPath)
	if err != nil {
		return nil, fmt.Errorf("invalid path: %w", err)
	}

	s.statusCacheMu.RLock()
	entry, ok := s.statusCache[absPath]
	s.statusCacheMu.RUnlock()
	if ok && time.Since(entry.cachedAt) < statusCacheTTL {
		return entry.status, nil
	}

	dbPath, err := s.GenerateDBPath(absPath)
	if err != nil {
		return nil, err
	}
	release := s.lockDB(dbPath)
	projectDB, err := s.OpenDB(absPath)
	if err != nil {
		release()
		return nil, err
	}
	defer release()
	defer projectDB.Close()

	status, err := s.engine.GetIndexStatus(projectDB, absPath)
	if err != nil {
		return nil, err
	}

	s.statusCacheMu.Lock()
	s.statusCache[absPath] = statusCacheEntry{status: status, cachedAt: time.Now()}
	s.statusCacheMu.Unlock()

	return status, nil
}

// invalidateStatusCache clears the cached status for a folder so the next call rescans.
func (s *Service) invalidateStatusCache(absPath string) {
	s.statusCacheMu.Lock()
	delete(s.statusCache, absPath)
	s.statusCacheMu.Unlock()
}

// lockDB acquires a per-dbPath mutex and returns a release function.
// DuckDB allows only one open connection to a file at a time, so callers
// must hold this lock for the duration of open → use → close.
func (s *Service) lockDB(dbPath string) func() {
	s.dbMuMu.Lock()
	mu, ok := s.dbMu[dbPath]
	if !ok {
		mu = &sync.Mutex{}
		s.dbMu[dbPath] = mu
	}
	s.dbMuMu.Unlock()
	mu.Lock()
	return mu.Unlock
}

// OpenDBLocked opens the project database and returns it along with a release function.
// The caller must call release() after Close() to allow other goroutines to open the same DB.
func (s *Service) OpenDBLocked(folderPath string) (*db.ProjectDB, func(), error) {
	dbPath, err := s.GenerateDBPath(folderPath)
	if err != nil {
		return nil, nil, err
	}
	release := s.lockDB(dbPath)
	dim, err := db.GetDimensionFromDB(dbPath)
	if err != nil {
		release()
		return nil, nil, fmt.Errorf("index not found for %s", folderPath)
	}
	pdb, err := db.OpenProjectDB(dbPath, dim)
	if err != nil {
		release()
		return nil, nil, err
	}
	return pdb, release, nil
}

// GetIndexStats returns the number of indexed chunks for a folder.
func (s *Service) GetIndexStats(folderPath string) (int, error) {
	dbPath, err := s.GenerateDBPath(folderPath)
	if err != nil {
		return 0, err
	}
	release := s.lockDB(dbPath)
	defer release()
	projectDB, err := s.OpenDB(folderPath)
	if err != nil {
		return 0, err
	}
	defer projectDB.Close()
	return projectDB.CountChunks()
}

// ListIndexes returns metadata for all indexed folders.
func (s *Service) ListIndexes() ([]IndexInfo, error) {
	files, err := filepath.Glob(filepath.Join(s.config.IndexDir, "*.db"))
	if err != nil {
		return nil, fmt.Errorf("list indexes: %w", err)
	}

	var (
		mu      sync.Mutex
		wg      sync.WaitGroup
		indexes = make([]IndexInfo, 0, len(files))
	)

	for _, dbFile := range files {
		wg.Add(1)
		go func(dbFile string) {
			defer wg.Done()

			meta, err := db.ReadIndexMeta(dbFile)
			if err != nil {
				// DB temporarily inaccessible — include stub so it doesn't vanish from the sidebar
				folder := strings.TrimSuffix(filepath.Base(dbFile), ".db")
				if info, err2 := os.Stat(dbFile); err2 == nil {
					mu.Lock()
					indexes = append(indexes, IndexInfo{
						FolderPath: folder, DBPath: dbFile,
						SizeBytes: info.Size(), LastModified: info.ModTime(),
					})
					mu.Unlock()
				}
				return
			}

			folder := meta.FolderPath
			if folder == "" {
				folder = strings.TrimSuffix(filepath.Base(dbFile), ".db")
			}

			var size int64
			var modTime time.Time
			if info, err := os.Stat(dbFile); err == nil {
				size = info.Size()
				modTime = info.ModTime()
			}

			mu.Lock()
			indexes = append(indexes, IndexInfo{
				FolderPath: folder, DBPath: dbFile,
				DocumentCount: meta.ChunkCount, SizeBytes: size, LastModified: modTime,
			})
			mu.Unlock()
		}(dbFile)
	}

	wg.Wait()
	return indexes, nil
}
