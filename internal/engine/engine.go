package engine

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/karim-daw/qwelli/internal/db"
	"github.com/karim-daw/qwelli/internal/engine/differ"
	"github.com/karim-daw/qwelli/internal/engine/embeddings"
	"github.com/karim-daw/qwelli/internal/engine/extraction"
	"github.com/karim-daw/qwelli/internal/engine/fileprocessor"
	"github.com/karim-daw/qwelli/internal/engine/search"
	"github.com/karim-daw/qwelli/internal/voyage"
)

type SearchResult struct {
	FilePath     string
	FileName     string
	Distance     float64
	Content      string
	TextMetadata map[string]interface{}
	ImageData    []byte // Base64 encoded image data for image results
	PageNumbers  []int  // Page numbers for PDF chunks
}

type Engine struct {
	voyageClient          *voyage.Client
	enableMultimodal      bool
	contentTypeMode       fileprocessor.ContentTypeMode
	fileProcessingService *fileprocessor.FileProcessingService
}

// NewEngine creates an engine using a Voyage client
func NewEngine(voyageClient *voyage.Client, enableMultimodal bool) *Engine {
	processingConfig := fileprocessor.DefaultProcessingConfig()
	processingConfig.EnableMultimodal = enableMultimodal

	return &Engine{
		voyageClient:          voyageClient,
		enableMultimodal:      enableMultimodal,
		contentTypeMode:       fileprocessor.ContentTypeBoth,
		fileProcessingService: fileprocessor.NewFileProcessingService(processingConfig),
	}
}

// getEmbedder returns an embedder using the voyage client
func (e *Engine) getEmbedder() (*embeddings.Embedder, error) {
	return embeddings.NewEmbedder(e.voyageClient)
}

// SetContentTypeMode sets the content type mode for indexing
func (e *Engine) SetContentTypeMode(mode fileprocessor.ContentTypeMode) {
	e.contentTypeMode = mode
	// Update the file processing service config
	e.fileProcessingService.SetContentTypeMode(mode)
}

func (e *Engine) IndexFolder(ctx context.Context, folderPath, dbPath string, incremental bool, progressCallback func(current, total int, filename string), phaseCallback ...func(phase, message string, current, total int)) error {
	// Helper to emit phase updates
	emitPhase := func(phase, message string, current, total int) {
		if len(phaseCallback) > 0 && phaseCallback[0] != nil {
			phaseCallback[0](phase, message, current, total)
		}
	}

	// Detect dimension
	embedder, err := e.getEmbedder()
	if err != nil {
		return err
	}
	testEmbed, err := embedder.Embed("test")
	if err != nil {
		return err
	}
	dimension := len(testEmbed)

	// Check if model changed - if so, delete DB to start fresh
	currentModel := e.voyageClient.EmbeddingModel()
	if existingDim, err := db.GetDimensionFromDB(dbPath); err == nil {
		tempDB, _ := db.OpenProjectDB(dbPath, existingDim)
		if tempDB != nil {
			storedModel, _ := tempDB.GetMetadata("model")
			tempDB.Close()
			if storedModel != "" && storedModel != currentModel {
				fmt.Printf("⚠️  Model changed from '%s' to '%s' - recreating database...\n", storedModel, currentModel)
				os.Remove(dbPath)
			}
		}
	}

	// Open project database
	projectDB, err := db.OpenProjectDB(dbPath, dimension)
	if err != nil {
		return err
	}
	defer projectDB.Close()

	// Store project metadata
	projectDB.SetMetadata("dimension", fmt.Sprintf("%d", dimension))
	projectDB.SetMetadata("model", currentModel)
	projectDB.SetMetadata("folder_path", folderPath)

	// Determine which files to process
	var filesToProcess []string
	var needsRebuild bool

	// if incremental, detect changes
	if incremental {
		// Detect changes
		changes, err := differ.DetectChanges(projectDB, folderPath)
		if err != nil {
			return fmt.Errorf("failed to detect changes: %w", err)
		}

		// Delete removed files
		for _, path := range changes.ToDelete {
			file, err := projectDB.GetFileByPath(path)
			if err == nil {
				if err := projectDB.DeleteFile(file.FileID); err != nil {
					log.Printf("⚠️  Failed to delete file %s: %v", path, err)
				} else {
					needsRebuild = true
				}
			}
		}

		// Process files to add and update
		filesToProcess = append(filesToProcess, changes.ToAdd...)
		filesToProcess = append(filesToProcess, changes.ToUpdate...)

		// For updated files, delete existing data first
		for _, path := range changes.ToUpdate {
			file, err := projectDB.GetFileByPath(path)
			if err == nil {
				if err := projectDB.DeleteFile(file.FileID); err != nil {
					log.Printf("⚠️  Failed to delete old file data %s: %v", path, err)
				} else {
					needsRebuild = true
				}
			}
		}

		if len(filesToProcess) == 0 {
			// No changes detected
			return nil
		}
	} else {
		// Full index: process all files
		allFiles, err := differ.ScanFolder(folderPath)
		if err != nil {
			return err
		}
		if len(allFiles) == 0 {
			return fmt.Errorf("no files found in %s", folderPath)
		}
		filesToProcess = allFiles
		needsRebuild = true
	}

	// Process files
	var allChunks []db.Chunk
	skippedFiles := 0
	oneDriveSkipped := 0

	for i, f := range filesToProcess {
		// Check for cancellation
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		if progressCallback != nil {
			progressCallback(i+1, len(filesToProcess), f)
		}

		// Normalize path to absolute
		absPath, err := filepath.Abs(f)
		if err != nil {
			skippedFiles++
			continue
		}

		info, err := os.Stat(absPath)
		if err != nil {
			log.Printf("⚠️  Failed to stat file %s: %v", absPath, err)
			skippedFiles++
			continue
		}

		// Compute file hash
		fileHash, err := extraction.ComputeSHA256(absPath)
		if err != nil {
			// Check if it's an OneDrive I/O error
			if strings.Contains(err.Error(), "OneDrive placeholder") || strings.Contains(err.Error(), "input/output error") {
				oneDriveSkipped++
				if oneDriveSkipped <= 3 {
					// Only show first 3 OneDrive errors to avoid spam
					log.Printf("⚠️  Skipping OneDrive placeholder file (not downloaded): %s", filepath.Base(absPath))
				}
			} else {
				log.Printf("⚠️  Failed to compute hash for %s: %v", absPath, err)
			}
			skippedFiles++
			continue
		}

		// Create file record
		fileID := differ.GenerateFileID(absPath)
		file := db.File{
			FileID:     fileID,
			Path:       absPath,
			FileType:   differ.GetFileTypeFromPath(absPath),
			FileHash:   fileHash,
			ModifiedAt: info.ModTime(),
			Size:       info.Size(),
			IndexedAt:  time.Now(),
		}

		if err := projectDB.InsertFile(file); err != nil {
			log.Printf("⚠️  Failed to insert file %s: %v", absPath, err)
			continue
		}

		// Check if file type is supported
		if !e.fileProcessingService.CanProcess(file.FileType) {
			log.Printf("⚠️  Unsupported file type %s: %s", file.FileType, absPath)
			continue
		}

		var chunks []db.Chunk
		var processErr error

		// Process based on file type
		if file.FileType == "pdf" {
			// Process PDF
			chunks, _, processErr = e.fileProcessingService.ProcessPDF(file)
			if processErr != nil {
				log.Printf("⚠️  Failed to process PDF %s: %v", filepath.Base(absPath), processErr)
				continue
			}
		} else {
			// Process text file
			// Skip very large text files
			if info.Size() > 500*1024 {
				continue
			}

			content, readErr := os.ReadFile(absPath)
			if readErr != nil {
				log.Printf("⚠️  Failed to read file %s: %v", absPath, readErr)
				continue
			}

			chunks, _, processErr = e.fileProcessingService.ProcessText(file, string(content))
			if processErr != nil {
				log.Printf("⚠️  Failed to process text file %s: %v", filepath.Base(absPath), processErr)
				continue
			}
		}

		allChunks = append(allChunks, chunks...)
	}

	// Generate embeddings for all chunks using EmbeddingGenerator
	if len(allChunks) > 0 {
		// Check for cancellation before starting embeddings
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		log.Printf("🔄 Generating embeddings for %d chunks...", len(allChunks))
		emitPhase("embedding", fmt.Sprintf("Generating embeddings for %d chunks", len(allChunks)), 0, len(allChunks))
		embeddingStart := time.Now()

		// Reset progress bar when embedding starts
		// Send a progress message with current=0 to reset the bar
		if progressCallback != nil {
			progressCallback(0, len(allChunks), "Generating embeddings...")
		}

		// Create progress callback for embedding batches
		embeddingProgressCallback := func(current, total int) {
			// Check for cancellation
			select {
			case <-ctx.Done():
				return
			default:
			}

			// Send progress update through progress callback to update the progress bar
			if progressCallback != nil {
				progressCallback(current, total, fmt.Sprintf("Generating embeddings: %d/%d chunks", current, total))
			}
			// Also send phase update for the message
			emitPhase("embedding", fmt.Sprintf("Generating embeddings: %d/%d chunks", current, total), current, total)
		}

		embeddingGen := embeddings.NewEmbeddingGenerator(embedder, e.enableMultimodal)
		embeddingMap, err := embeddingGen.GenerateEmbeddings(ctx, allChunks, embeddingProgressCallback)
		if err != nil {
			// Check if error is due to cancellation
			if ctx.Err() != nil {
				return ctx.Err()
			}
			return fmt.Errorf("failed to generate embeddings: %w", err)
		}

		log.Printf("✅ Generated %d embeddings in %v", len(embeddingMap), time.Since(embeddingStart))
		emitPhase("embedding", fmt.Sprintf("Generated %d embeddings", len(embeddingMap)), len(embeddingMap), len(allChunks))

		// Store chunks and embeddings
		log.Printf("💾 Storing chunks and embeddings in database...")
		emitPhase("storing", fmt.Sprintf("Storing %d chunks and embeddings", len(allChunks)), 0, len(allChunks))
		storeStart := time.Now()
		if err := embeddings.StoreChunksAndEmbeddings(projectDB, allChunks, embeddingMap); err != nil {
			return fmt.Errorf("failed to store chunks and embeddings: %w", err)
		}
		log.Printf("✅ Stored in %v", time.Since(storeStart))
		emitPhase("storing", fmt.Sprintf("Stored %d chunks", len(allChunks)), len(allChunks), len(allChunks))
	}

	// Rebuild HNSW index if needed (after any embedding changes)
	if (needsRebuild || len(allChunks) > 0) && len(allChunks) > 0 {
		log.Printf("🔨 Rebuilding HNSW index...")
		emitPhase("hnsw", "Rebuilding HNSW index", 0, 1)
		indexStart := time.Now()
		if err := projectDB.RebuildHNSWIndex(); err != nil {
			return fmt.Errorf("failed to rebuild HNSW index: %w", err)
		}
		log.Printf("✅ HNSW index rebuilt in %v", time.Since(indexStart))
		emitPhase("hnsw", "HNSW index rebuilt", 1, 1)
	}

	// Print summary
	successfulFiles := len(filesToProcess) - skippedFiles
	emitPhase("complete", fmt.Sprintf("Processed %d files successfully", successfulFiles), successfulFiles, len(filesToProcess))
	log.Printf("📊 Indexing Summary: %d files processed successfully, %d files skipped", successfulFiles, skippedFiles)
	if oneDriveSkipped > 0 {
		log.Printf("💡 Note: %d OneDrive placeholder files were skipped (not downloaded locally)", oneDriveSkipped)
		log.Printf("   To index these files, ensure they are downloaded in OneDrive settings:")
		log.Printf("   Right-click folder → 'Always keep on this device'")
	}

	return nil
}

// Search performs a search using the default (semantic) strategy
func (e *Engine) Search(query string, dbPath string, topK int) ([]SearchResult, error) {
	return e.SearchWithStrategy(query, dbPath, topK, "", "semantic")
}

// SearchWithFilter performs a search with content type filtering using the default (semantic) strategy
func (e *Engine) SearchWithFilter(query string, dbPath string, topK int, contentType string) ([]SearchResult, error) {
	return e.SearchWithStrategy(query, dbPath, topK, contentType, "semantic")
}

// SearchWithStrategy performs a search using the specified strategy
// strategy can be "semantic", "keyword", or "hybrid"
func (e *Engine) SearchWithStrategy(query string, dbPath string, topK int, contentType string, strategyName string) ([]SearchResult, error) {
	// Get the search strategy
	var strategy search.SearchStrategy

	// Helper to create semantic strategy using the voyage client
	createSemanticStrategy := func() *search.SemanticSearchStrategy {
		return search.NewSemanticSearchStrategy(e.voyageClient)
	}

	switch strategyName {
	case "semantic":
		strategy = createSemanticStrategy()
	case "keyword":
		strategy = search.NewKeywordSearchStrategy()
	case "hybrid":
		semantic := createSemanticStrategy()
		keyword := search.NewKeywordSearchStrategy()
		strategy = search.NewHybridSearchStrategy(semantic, keyword)
	default:
		// Fall back to semantic
		strategy = createSemanticStrategy()
	}

	// Open database - we need dimension, so try to get it from DB first
	dim, err := db.GetDimensionFromDB(dbPath)
	if err != nil {
		// If dimension not found, we need to create embedder to get dimension
		embedder, err2 := e.getEmbedder()
		if err2 != nil {
			return nil, fmt.Errorf("failed to get dimension: %w", err)
		}
		testEmbed, err2 := embedder.Embed("test")
		if err2 != nil {
			return nil, fmt.Errorf("failed to get dimension: %w", err)
		}
		dim = len(testEmbed)
	}

	projectDB, err := db.OpenProjectDB(dbPath, dim)
	if err != nil {
		return nil, err
	}
	defer projectDB.Close()

	// Perform search using strategy
	results, err := strategy.Search(query, projectDB, topK, contentType)
	if err != nil {
		return nil, err
	}

	// Convert db.SearchResult to engine.SearchResult
	var out []SearchResult
	for _, r := range results {
		// Build metadata from available fields
		metadata := make(map[string]interface{})
		metadata["chunk_index"] = r.ChunkIndex
		metadata["total_chunks"] = r.TotalChunks
		if len(r.PageNumbers) > 0 {
			metadata["page_numbers"] = r.PageNumbers
		}

		metadata["content_type"] = r.ContentType
		if r.ContentType == "image" && len(r.ImageData) > 0 {
			metadata["has_image"] = true
		}

		out = append(out, SearchResult{
			FilePath:     r.FilePath,
			FileName:     filepath.Base(r.FilePath),
			Distance:     r.Distance,
			Content:      strings.TrimSpace(r.Content),
			TextMetadata: metadata,
			ImageData:    r.ImageData,   // Include image data for previews
			PageNumbers:  r.PageNumbers, // Include page numbers directly
		})
	}
	return out, nil
}

func (e *Engine) GetIndexStats(dbPath string) (int, error) {
	dim, err := db.GetDimensionFromDB(dbPath)
	if err != nil {
		return 0, err
	}
	projectDB, err := db.OpenProjectDB(dbPath, dim)
	if err != nil {
		return 0, err
	}
	defer projectDB.Close()

	// Count chunks (which have embeddings)
	files, err := projectDB.GetAllFiles()
	if err != nil {
		return 0, err
	}

	totalChunks := 0
	for _, file := range files {
		chunks, err := projectDB.GetChunksForFile(file.FileID)
		if err != nil {
			continue
		}
		totalChunks += len(chunks)
	}

	return totalChunks, nil
}

func (e *Engine) GetFolderPath(dbPath string) (string, error) {
	dim, err := db.GetDimensionFromDB(dbPath)
	if err != nil {
		return "", err
	}
	projectDB, err := db.OpenProjectDB(dbPath, dim)
	if err != nil {
		return "", err
	}
	defer projectDB.Close()
	return projectDB.GetMetadata("folder_path")
}

// GetIndexStatus returns the status of an index showing pending changes
func (e *Engine) GetIndexStatus(dbPath, folderPath string) (*differ.IndexStatus, error) {
	dim, err := db.GetDimensionFromDB(dbPath)
	if err != nil {
		return nil, err
	}
	projectDB, err := db.OpenProjectDB(dbPath, dim)
	if err != nil {
		return nil, err
	}
	defer projectDB.Close()

	// Detect changes
	changes, err := differ.DetectChanges(projectDB, folderPath)
	if err != nil {
		return nil, fmt.Errorf("failed to detect changes: %w", err)
	}

	status := &differ.IndexStatus{
		ToAdd:    make([]differ.FileStatus, 0),
		ToUpdate: make([]differ.FileStatus, 0),
		ToDelete: make([]differ.FileStatus, 0),
	}

	// Populate ToAdd
	for _, path := range changes.ToAdd {
		info, err := os.Stat(path)
		if err != nil {
			continue
		}
		status.ToAdd = append(status.ToAdd, differ.FileStatus{
			Path:       path,
			FileType:   differ.GetFileTypeFromPath(path),
			Size:       info.Size(),
			ModifiedAt: info.ModTime(),
			Reason:     "new",
		})
	}

	// Populate ToUpdate
	for _, path := range changes.ToUpdate {
		info, err := os.Stat(path)
		if err != nil {
			continue
		}
		status.ToUpdate = append(status.ToUpdate, differ.FileStatus{
			Path:       path,
			FileType:   differ.GetFileTypeFromPath(path),
			Size:       info.Size(),
			ModifiedAt: info.ModTime(),
			Reason:     "modified",
		})
	}

	// Populate ToDelete
	for _, path := range changes.ToDelete {
		file, err := projectDB.GetFileByPath(path)
		if err != nil {
			continue
		}
		status.ToDelete = append(status.ToDelete, differ.FileStatus{
			Path:       path,
			FileType:   file.FileType,
			Size:       file.Size,
			ModifiedAt: file.ModifiedAt,
			Reason:     "deleted",
		})
	}

	// Get total counts
	allFiles, err := projectDB.GetAllFiles()
	if err != nil {
		return nil, err
	}
	status.Total = len(allFiles)
	status.UpToDate = status.Total - len(changes.ToUpdate) - len(changes.ToDelete)

	return status, nil
}
