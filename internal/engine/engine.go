package engine

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/karim-daw/qwelli/internal/db"
	"github.com/karim-daw/qwelli/internal/engine/differ"
	"github.com/karim-daw/qwelli/internal/engine/embeddings"
	"github.com/karim-daw/qwelli/internal/engine/fileprocessor"
	"github.com/karim-daw/qwelli/internal/engine/search"
	"github.com/karim-daw/qwelli/internal/voyage"
)

// SearchResult is the engine-level search result returned to callers.
type SearchResult struct {
	FilePath     string
	FileName     string
	Distance     float64
	Content      string
	TextMetadata map[string]interface{}
	ImageData    []byte
	PageNumbers  []int
}

// Engine contains the core business logic for indexing and search.
// It does NOT manage database connections — callers provide *db.ProjectDB.
type Engine struct {
	voyageClient            voyage.ClientInterface
	enableMultimodal        bool
	contentTypeMode         fileprocessor.ContentTypeMode
	fileProcessingService   *fileprocessor.FileProcessingService
	enableParallel          bool
	numWorkers              int
	maxConcurrentEmbeddings int
}

func NewEngine(voyageClient voyage.ClientInterface, enableMultimodal bool) *Engine {
	cfg := fileprocessor.DefaultProcessingConfig()
	cfg.EnableMultimodal = enableMultimodal
	return &Engine{
		voyageClient:          voyageClient,
		enableMultimodal:      enableMultimodal,
		contentTypeMode:       fileprocessor.ContentTypeAll,
		fileProcessingService: fileprocessor.NewFileProcessingService(cfg),
	}
}

// SetParallelProcessing enables or disables parallel file processing.
// When enabled, files are processed concurrently using a worker pool
// of the specified size. Set numWorkers to 0 to use default (4).
// maxConcurrentEmbeddings controls how many embedding API calls run in parallel.
func (e *Engine) SetParallelProcessing(enabled bool, numWorkers, maxConcurrentEmbeddings int) {
	e.enableParallel = enabled
	e.numWorkers = numWorkers
	e.maxConcurrentEmbeddings = maxConcurrentEmbeddings
}

// SetParallelPDFProcessing enables or disables parallel page processing within PDFs.
// This parallelizes text extraction and image extraction across pages.
// Useful for large PDFs with many pages. Set numWorkers to 0 for auto-detect.
func (e *Engine) SetParallelPDFProcessing(enabled bool, numWorkers int) {
	e.fileProcessingService.SetParallelPDFProcessing(enabled, numWorkers)
}

func (e *Engine) SetContentTypeMode(mode fileprocessor.ContentTypeMode) {
	e.contentTypeMode = mode
	e.fileProcessingService.SetContentTypeMode(mode)
}

// EmbeddingModel returns the configured model name (used for model-change detection).
func (e *Engine) EmbeddingModel() string {
	return e.voyageClient.EmbeddingModel()
}

// DetectDimension makes a test embedding call and returns the vector dimension.
func (e *Engine) DetectDimension() (int, error) {
	embedder, err := embeddings.NewEmbedder(e.voyageClient)
	if err != nil {
		return 0, err
	}
	vec, err := embedder.Embed("test")
	if err != nil {
		return 0, err
	}
	return len(vec), nil
}

type PhaseCallback func(phase, message string, current, total int)

func (e *Engine) IndexFolder(ctx context.Context, projectDB *db.ProjectDB, folderPath string, incremental bool, progressCb func(int, int, string), phaseCb PhaseCallback) error {
	// Check context before starting
	if err := ctx.Err(); err != nil {
		return err
	}

	emitPhase := func(phase, msg string, cur, tot int) {
		if phaseCb != nil {
			phaseCb(phase, msg, cur, tot)
		}
	}

	// Resolve files
	filesToProcess, needsRebuild, err := e.resolveFiles(projectDB, folderPath, incremental)
	if err != nil {
		return err
	}
	if len(filesToProcess) == 0 {
		return nil
	}

	// Check context before processing files
	if err := ctx.Err(); err != nil {
		return err
	}

	// For full (non-incremental) re-indexes, clear existing data so the bulk
	// Appender API doesn't hit duplicate key errors and fall back to slow
	// row-by-row inserts.
	if !incremental {
		if count, _ := projectDB.CountChunks(); count > 0 {
			log.Printf("🗑️  Full re-index: clearing %d existing chunks", count)
			if err := projectDB.ClearAllData(); err != nil {
				return fmt.Errorf("clear existing data: %w", err)
			}
		}
	}

	prevEmbeddingCount, _ := projectDB.CountEmbeddings()

	log.Printf("⚡ Indexing %d files via streaming pipeline (workers=%d, embed_concurrency=%d, CPUs=%d)",
		len(filesToProcess), e.numWorkers, e.maxConcurrentEmbeddings, runtime.NumCPU())

	totalChunks, skipped, onedriveSkipped, pipelineErr := e.indexFolderStreaming(
		ctx, projectDB, filesToProcess, e.maxConcurrentEmbeddings, progressCb, emitPhase,
	)
	if pipelineErr != nil {
		return pipelineErr
	}

	if err := e.rebuildHNSWAfterIndex(projectDB, needsRebuild, prevEmbeddingCount, totalChunks > 0, emitPhase); err != nil {
		return err
	}

	ok := len(filesToProcess) - skipped
	emitPhase("complete", fmt.Sprintf("Processed %d files successfully", ok), ok, len(filesToProcess))
	log.Printf("📊 Indexing: %d processed, %d skipped", ok, skipped)
	if onedriveSkipped > 0 {
		log.Printf("💡 %d OneDrive placeholder files skipped", onedriveSkipped)
	}
	return nil
}

// rebuildHNSWAfterIndex handles HNSW index rebuilding after indexing completes.
// It rebuilds unconditionally after file deletions, or conditionally when new embeddings were added.
func (e *Engine) rebuildHNSWAfterIndex(projectDB *db.ProjectDB, needsRebuild bool, prevEmbeddingCount int, hasNewEmbeddings bool, emitPhase func(string, string, int, int)) error {
	if needsRebuild {
		log.Printf("🔨 Rebuilding HNSW index (after file deletions)...")
		emitPhase("hnsw", "Rebuilding HNSW index", 0, 1)
		start := time.Now()
		if err := projectDB.RebuildHNSWIndex(); err != nil {
			return fmt.Errorf("rebuild HNSW index: %w", err)
		}
		log.Printf("✅ HNSW index rebuilt in %v", time.Since(start))
		emitPhase("hnsw", "HNSW index rebuilt", 1, 1)
	} else if hasNewEmbeddings {
		emitPhase("hnsw", "Rebuilding HNSW index", 0, 1)
		start := time.Now()
		rebuilt, err := projectDB.BuildHNSWIndexIfNeeded(prevEmbeddingCount)
		if err != nil {
			return fmt.Errorf("rebuild HNSW index: %w", err)
		}
		if rebuilt {
			log.Printf("🔨 Rebuilding HNSW index...")
			log.Printf("✅ HNSW index rebuilt in %v", time.Since(start))
		} else if projectDB.IsHNSWStale() {
			log.Printf("⏳ HNSW rebuild deferred (small incremental update)")
		}
		emitPhase("hnsw", "HNSW index rebuilt", 1, 1)
	}
	return nil
}

func (e *Engine) resolveFiles(projectDB *db.ProjectDB, folderPath string, incremental bool) ([]string, bool, error) {
	if !incremental {
		files, err := differ.ScanFolder(folderPath)
		if err != nil {
			return nil, false, err
		}
		if len(files) == 0 {
			// Empty folder is not an error - just return empty list
			return nil, false, nil
		}
		return files, true, nil
	}

	changes, err := differ.DetectChanges(projectDB, folderPath)
	if err != nil {
		return nil, false, fmt.Errorf("detect changes: %w", err)
	}

	needsRebuild := false
	for _, paths := range [][]string{changes.ToDelete, changes.ToUpdate} {
		for _, p := range paths {
			if f, err := projectDB.GetFileByPath(p); err == nil {
				if err := projectDB.DeleteFile(f.FileID); err != nil {
					log.Printf("⚠️  Failed to delete %s: %v", p, err)
				} else {
					needsRebuild = true
				}
			}
		}
	}

	var files []string
	files = append(files, changes.ToAdd...)
	files = append(files, changes.ToUpdate...)
	return files, needsRebuild, nil
}

func (e *Engine) Search(projectDB *db.ProjectDB, query string, topK int, contentType, strategyName string) ([]SearchResult, error) {
	strategy := e.buildStrategy(strategyName)
	results, err := strategy.Search(query, projectDB, topK, contentType)
	if err != nil {
		return nil, err
	}
	return toSearchResults(results), nil
}

func (e *Engine) buildStrategy(name string) search.SearchStrategy {
	newSemantic := func() *search.SemanticSearchStrategy {
		return search.NewSemanticSearchStrategy(e.voyageClient)
	}
	switch name {
	case "keyword":
		return search.NewKeywordSearchStrategy()
	case "hybrid":
		return search.NewHybridSearchStrategy(newSemantic(), search.NewKeywordSearchStrategy())
	default:
		return newSemantic()
	}
}

func (e *Engine) GetIndexStatus(projectDB *db.ProjectDB, folderPath string) (*differ.IndexStatus, error) {
	changes, err := differ.DetectChanges(projectDB, folderPath)
	if err != nil {
		return nil, fmt.Errorf("detect changes: %w", err)
	}

	status := &differ.IndexStatus{
		ToAdd:    make([]differ.FileStatus, 0),
		ToUpdate: make([]differ.FileStatus, 0),
		ToDelete: make([]differ.FileStatus, 0),
	}

	for _, entry := range []struct {
		paths  []string
		target *[]differ.FileStatus
		reason string
	}{
		{changes.ToAdd, &status.ToAdd, "new"},
		{changes.ToUpdate, &status.ToUpdate, "modified"},
	} {
		for _, p := range entry.paths {
			if info, err := os.Stat(p); err == nil {
				*entry.target = append(*entry.target, differ.FileStatus{
					Path: p, FileType: fileprocessor.GetFileTypeFromPath(p),
					Size: info.Size(), ModifiedAt: info.ModTime(), Reason: entry.reason,
				})
			}
		}
	}
	for _, p := range changes.ToDelete {
		if f, err := projectDB.GetFileByPath(p); err == nil {
			status.ToDelete = append(status.ToDelete, differ.FileStatus{
				Path: p, FileType: f.FileType, Size: f.Size,
				ModifiedAt: f.ModifiedAt, Reason: "deleted",
			})
		}
	}

	allFiles, _ := projectDB.GetAllFiles()
	status.Total = len(allFiles)
	status.UpToDate = status.Total - len(changes.ToUpdate) - len(changes.ToDelete)
	return status, nil
}

func toSearchResults(dbResults []db.SearchResult) []SearchResult {
	out := make([]SearchResult, len(dbResults))
	for i, r := range dbResults {
		meta := map[string]interface{}{
			"chunk_index": r.ChunkIndex, "total_chunks": r.TotalChunks,
			"content_type": r.ContentType,
		}
		if len(r.PageNumbers) > 0 {
			meta["page_numbers"] = r.PageNumbers
		}
		if r.ContentType == "image" && len(r.ImageData) > 0 {
			meta["has_image"] = true
		}
		out[i] = SearchResult{
			FilePath: r.FilePath, FileName: filepath.Base(r.FilePath),
			Distance: r.Distance, Content: strings.TrimSpace(r.Content),
			TextMetadata: meta, ImageData: r.ImageData, PageNumbers: r.PageNumbers,
		}
	}
	return out
}
