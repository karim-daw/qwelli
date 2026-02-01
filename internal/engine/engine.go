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
	fileProcessingService *fileprocessor.FileProcessingService
}

// NewEngine creates an engine using a Voyage client (always multimodal now)
func NewEngine(voyageClient *voyage.Client, _ bool) *Engine {
	processingConfig := fileprocessor.DefaultProcessingConfig()
	processingConfig.EnableMultimodal = true // Always enabled

	return &Engine{
		voyageClient:          voyageClient,
		fileProcessingService: fileprocessor.NewFileProcessingService(processingConfig),
	}
}

// getEmbedder returns an embedder using the voyage client
func (e *Engine) getEmbedder() (*embeddings.Embedder, error) {
	return embeddings.NewEmbedder(e.voyageClient)
}

// SetContentTypeMode is deprecated - multimodal is always enabled
func (e *Engine) SetContentTypeMode(mode fileprocessor.ContentTypeMode) {
	// No-op: multimodal is always enabled now
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
			storedModel, _ := tempDB.GetMetadata(context.Background(), "model")
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
	projectDB.SetMetadata(context.Background(), "dimension", fmt.Sprintf("%d", dimension))
	projectDB.SetMetadata(context.Background(), "model", currentModel)
	projectDB.SetMetadata(context.Background(), "folder_path", folderPath)

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
			file, err := projectDB.GetFileByPath(context.Background(), path)
			if err == nil {
				if err := projectDB.DeleteFile(context.Background(), file.FileID); err != nil {
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
			file, err := projectDB.GetFileByPath(context.Background(), path)
			if err == nil {
				if err := projectDB.DeleteFile(context.Background(), file.FileID); err != nil {
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

		if err := projectDB.InsertFile(context.Background(), file); err != nil {
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

		embeddingGen := embeddings.NewEmbeddingGenerator(embedder, true)
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
		if err := projectDB.RebuildHNSWIndex(context.Background()); err != nil {
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

// SearchWithFilter is deprecated - use SearchWithStrategy instead (content type filtering removed)
func (e *Engine) SearchWithFilter(query string, dbPath string, topK int, contentType string) ([]SearchResult, error) {
	return e.SearchWithStrategy(query, dbPath, topK, "", "semantic")
}

// SearchWithStrategy performs a search using the specified strategy
// strategy can be "semantic", "keyword", or "hybrid"
// Note: contentType parameter is deprecated (always searches all content in multimodal mode)
func (e *Engine) SearchWithStrategy(query string, dbPath string, topK int, contentType string, strategyName string) ([]SearchResult, error) {
	// Open database
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

	ctx := context.Background()
	var results []db.SearchResult

	// Perform search based on strategy (inlined for simplicity)
	switch strategyName {
	case "keyword":
		// Full-text search
		results, err = projectDB.SearchFTS(ctx, query, topK, "")

	case "hybrid":
		// Hybrid: combine semantic + keyword
		embedder, err := e.getEmbedder()
		if err != nil {
			return nil, err
		}
		queryVec, err := embedder.Embed(query)
		if err != nil {
			return nil, err
		}

		// Get both semantic and keyword results
		semanticResults, err := projectDB.SearchANN(ctx, queryVec, topK*2)
		if err != nil {
			return nil, fmt.Errorf("semantic search failed: %w", err)
		}

		keywordResults, err := projectDB.SearchFTS(ctx, query, topK*2, "")
		if err != nil {
			return nil, fmt.Errorf("keyword search failed: %w", err)
		}

		// Merge results with weighted scoring (70% semantic, 30% keyword)
		results = e.mergeSearchResults(semanticResults, keywordResults, topK)

	default: // "semantic" or fallback
		// Vector similarity search
		embedder, err := e.getEmbedder()
		if err != nil {
			return nil, err
		}
		queryVec, err := embedder.Embed(query)
		if err != nil {
			return nil, err
		}
		results, err = projectDB.SearchANN(ctx, queryVec, topK)
	}

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
	files, err := projectDB.GetAllFiles(context.Background())
	if err != nil {
		return 0, err
	}

	totalChunks := 0
	for _, file := range files {
		chunks, err := projectDB.GetChunksForFile(context.Background(), file.FileID)
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
	return projectDB.GetMetadata(context.Background(), "folder_path")
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
		file, err := projectDB.GetFileByPath(context.Background(), path)
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
	allFiles, err := projectDB.GetAllFiles(context.Background())
	if err != nil {
		return nil, err
	}
	status.Total = len(allFiles)
	status.UpToDate = status.Total - len(changes.ToUpdate) - len(changes.ToDelete)

	return status, nil
}

// mergeSearchResults merges semantic and keyword results with weighted scoring
// Uses 70% semantic weight and 30% keyword weight
func (e *Engine) mergeSearchResults(semantic, keyword []db.SearchResult, topK int) []db.SearchResult {
	const semanticWeight = 0.7
	const keywordWeight = 0.3

	// Create a map to track results by chunk ID
	resultMap := make(map[string]*db.SearchResult)

	// Process semantic results
	for i := range semantic {
		chunkID := semantic[i].ChunkID
		score := (1.0 - semantic[i].Distance) * semanticWeight

		if existing, ok := resultMap[chunkID]; ok {
			existing.Distance = existing.Distance - score
		} else {
			result := semantic[i]
			result.Distance = 1.0 - score
			resultMap[chunkID] = &result
		}
	}

	// Process keyword results
	for i := range keyword {
		chunkID := keyword[i].ChunkID
		score := (1.0 - keyword[i].Distance) * keywordWeight

		if existing, ok := resultMap[chunkID]; ok {
			existing.Distance = existing.Distance - score
		} else {
			result := keyword[i]
			result.Distance = 1.0 - score
			resultMap[chunkID] = &result
		}
	}

	// Convert map to slice and sort by distance
	results := make([]db.SearchResult, 0, len(resultMap))
	for _, result := range resultMap {
		results = append(results, *result)
	}

	// Sort by distance (ascending - lower is better)
	for i := 0; i < len(results)-1; i++ {
		for j := i + 1; j < len(results); j++ {
			if results[j].Distance < results[i].Distance {
				results[i], results[j] = results[j], results[i]
			}
		}
	}

	// Return topK results
	if len(results) > topK {
		return results[:topK]
	}
	return results
}
