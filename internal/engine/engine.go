package engine

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/karim-daw/qwelli/internal/db"
	"github.com/karim-daw/qwelli/internal/engine/fileprocessor"
	"github.com/karim-daw/qwelli/internal/engine/indexer"
	"github.com/karim-daw/qwelli/internal/engine/processor"
	"github.com/karim-daw/qwelli/internal/engine/scanner"
	"github.com/karim-daw/qwelli/internal/engine/search"
)

type SearchResult struct {
	FilePath     string
	FileName     string
	Distance     float64
	Content      string
	TextMetadata map[string]interface{}
	ImageData    []byte // Base64 encoded image data for image results
}

type Engine struct {
	apiKey           string
	model            string
	endpoint         string
	enableMultimodal bool
}

func NewEngine(apiKey, model, endpoint string, enableMultimodal bool) *Engine {
	return &Engine{
		apiKey:           apiKey,
		model:            model,
		endpoint:         endpoint,
		enableMultimodal: enableMultimodal,
	}
}

func (e *Engine) IndexFolder(folderPath, dbPath string, incremental bool, progressCallback func(current, total int, filename string)) error {
	// Detect dimension
	embedder, err := indexer.NewEmbedder(e.apiKey, e.model, e.endpoint)
	if err != nil {
		return err
	}
	testEmbed, err := embedder.Embed("test")
	if err != nil {
		return err
	}
	dimension := len(testEmbed)

	// Check if model changed - if so, delete DB to start fresh
	if existingDim, err := db.GetDimensionFromDB(dbPath); err == nil {
		tempDB, _ := db.OpenProjectDB(dbPath, existingDim)
		if tempDB != nil {
			storedModel, _ := tempDB.GetMetadata("model")
			tempDB.Close()
			if storedModel != "" && storedModel != e.model {
				fmt.Printf("⚠️  Model changed from '%s' to '%s' - recreating database...\n", storedModel, e.model)
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
	projectDB.SetMetadata("model", e.model)
	projectDB.SetMetadata("folder_path", folderPath)

	// Determine which files to process
	var filesToProcess []string
	var needsRebuild bool

	// if incremental, detect changes
	if incremental {
		// Detect changes
		changes, err := scanner.DetectChanges(projectDB, folderPath)
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
		allFiles, err := scanner.ScanFolder(folderPath)
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
		fileHash, err := processor.ComputeSHA256(absPath)
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
		fileID := scanner.GenerateFileID(absPath)
		file := db.File{
			FileID:     fileID,
			Path:       absPath,
			FileType:   scanner.GetFileTypeFromPath(absPath),
			FileHash:   fileHash,
			ModifiedAt: info.ModTime(),
			Size:       info.Size(),
			IndexedAt:  time.Now(),
		}

		if err := projectDB.InsertFile(file); err != nil {
			log.Printf("⚠️  Failed to insert file %s: %v", absPath, err)
			continue
		}

		// Process file and create chunks using file processor registry
		processor := fileprocessor.GetProcessor(file.FileType)
		if processor == nil {
			log.Printf("⚠️  No processor found for file type %s: %s", file.FileType, absPath)
			continue
		}

		// Prepare processing options
		options := fileprocessor.ProcessOptions{
			EnableMultimodal: e.enableMultimodal && embedder.IsMultimodal(),
			ChunkSize:        300,
			OverlapSize:      10,
		}

		// For text files, read content first
		if file.FileType != "pdf" {
			// Skip very large text files
			if info.Size() > 500*1024 {
				continue
			}

			content, err := os.ReadFile(absPath)
			if err != nil {
				log.Printf("⚠️  Failed to read file %s: %v", absPath, err)
				continue
			}
			options.FileContent = string(content)
		}

		chunks, _, err := processor.Process(file, options)
		if err != nil {
			log.Printf("⚠️  Failed to process file %s: %v", filepath.Base(absPath), err)
			continue
		}

		allChunks = append(allChunks, chunks...)
	}

	// Generate embeddings for all chunks using EmbeddingGenerator
	if len(allChunks) > 0 {
		log.Printf("🔄 Generating embeddings for %d chunks...", len(allChunks))
		embeddingStart := time.Now()

		embeddingGen := NewEmbeddingGenerator(embedder, e.enableMultimodal)
		embeddingMap, err := embeddingGen.GenerateEmbeddings(allChunks)
		if err != nil {
			return fmt.Errorf("failed to generate embeddings: %w", err)
		}

		log.Printf("✅ Generated %d embeddings in %v", len(embeddingMap), time.Since(embeddingStart))

		// Store chunks and embeddings
		log.Printf("💾 Storing chunks and embeddings in database...")
		storeStart := time.Now()
		if err := StoreChunksAndEmbeddings(projectDB, allChunks, embeddingMap); err != nil {
			return fmt.Errorf("failed to store chunks and embeddings: %w", err)
		}
		log.Printf("✅ Stored in %v", time.Since(storeStart))
	}

	// Rebuild HNSW index if needed (after any embedding changes)
	if (needsRebuild || len(allChunks) > 0) && len(allChunks) > 0 {
		log.Printf("🔨 Rebuilding HNSW index...")
		indexStart := time.Now()
		if err := projectDB.RebuildHNSWIndex(); err != nil {
			return fmt.Errorf("failed to rebuild HNSW index: %w", err)
		}
		log.Printf("✅ HNSW index rebuilt in %v", time.Since(indexStart))
	}

	// Print summary
	successfulFiles := len(filesToProcess) - skippedFiles
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

	switch strategyName {
	case "semantic":
		strategy = search.NewSemanticSearchStrategyWithConfig(e.apiKey, e.model, e.endpoint)
	case "keyword":
		strategy = search.NewKeywordSearchStrategy()
	case "hybrid":
		semantic := search.NewSemanticSearchStrategyWithConfig(e.apiKey, e.model, e.endpoint)
		keyword := search.NewKeywordSearchStrategy()
		strategy = search.NewHybridSearchStrategy(semantic, keyword)
	default:
		// Fall back to semantic
		strategy = search.NewSemanticSearchStrategyWithConfig(e.apiKey, e.model, e.endpoint)
	}

	// Open database - we need dimension, so try to get it from DB first
	dim, err := db.GetDimensionFromDB(dbPath)
	if err != nil {
		// If dimension not found, we need to create embedder to get dimension
		embedder, err2 := indexer.NewEmbedder(e.apiKey, e.model, e.endpoint)
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
			ImageData:    r.ImageData, // Include image data for previews
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
func (e *Engine) GetIndexStatus(dbPath, folderPath string) (*scanner.IndexStatus, error) {
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
	changes, err := scanner.DetectChanges(projectDB, folderPath)
	if err != nil {
		return nil, fmt.Errorf("failed to detect changes: %w", err)
	}

	status := &scanner.IndexStatus{
		ToAdd:    make([]scanner.FileStatus, 0),
		ToUpdate: make([]scanner.FileStatus, 0),
		ToDelete: make([]scanner.FileStatus, 0),
	}

	// Populate ToAdd
	for _, path := range changes.ToAdd {
		info, err := os.Stat(path)
		if err != nil {
			continue
		}
		status.ToAdd = append(status.ToAdd, scanner.FileStatus{
			Path:       path,
			FileType:   scanner.GetFileTypeFromPath(path),
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
		status.ToUpdate = append(status.ToUpdate, scanner.FileStatus{
			Path:       path,
			FileType:   scanner.GetFileTypeFromPath(path),
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
		status.ToDelete = append(status.ToDelete, scanner.FileStatus{
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
