package engine

import (
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"io/fs"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/karim-daw/qwelli/internal/db"
	"github.com/karim-daw/qwelli/internal/indexer"
	"github.com/karim-daw/qwelli/internal/processor"
)

type Engine struct {
	apiKey   string
	model    string
	endpoint string
}

func NewEngine(apiKey, model, endpoint string) *Engine {
	return &Engine{apiKey: apiKey, model: model, endpoint: endpoint}
}

type SearchResult struct {
	FilePath     string
	FileName     string
	Distance     float64
	Content      string
	TextMetadata map[string]interface{}
}

// ChangeSet represents files that need to be added, updated, or deleted
type ChangeSet struct {
	ToAdd    []string
	ToUpdate []string
	ToDelete []string
}

func (e *Engine) IndexFolder(folderPath, dbPath string, progressCallback func(current, total int, filename string)) error {
	return e.IndexFolderIncremental(folderPath, dbPath, false, progressCallback)
}

func (e *Engine) IndexFolderIncremental(folderPath, dbPath string, incremental bool, progressCallback func(current, total int, filename string)) error {
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

	projectDB, err := db.OpenProjectDB(dbPath, dimension)
	if err != nil {
		return err
	}
	defer projectDB.Close()

	// Store metadata
	projectDB.SetMetadata("dimension", fmt.Sprintf("%d", dimension))
	projectDB.SetMetadata("model", e.model)
	projectDB.SetMetadata("folder_path", folderPath)

	// Determine which files to process
	var filesToProcess []string
	var needsRebuild bool

	if incremental {
		// Detect changes
		changes, err := DetectChanges(projectDB, folderPath)
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
		allFiles, err := scanFolder(folderPath)
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
	var allContents []string

	for i, f := range filesToProcess {
		if progressCallback != nil {
			progressCallback(i+1, len(filesToProcess), f)
		}

		// Normalize path to absolute
		absPath, err := filepath.Abs(f)
		if err != nil {
			continue
		}

		info, err := os.Stat(absPath)
		if err != nil {
			log.Printf("⚠️  Failed to stat file %s: %v", absPath, err)
			continue
		}

		// Compute file hash
		fileHash, err := processor.ComputeSHA256(absPath)
		if err != nil {
			log.Printf("⚠️  Failed to compute hash for %s: %v", absPath, err)
			continue
		}

		// Create file record
		fileID := generateFileID(absPath)
		file := db.File{
			FileID:     fileID,
			Path:       absPath,
			FileType:   getFileType(absPath),
			FileHash:   fileHash,
			ModifiedAt: info.ModTime(),
			Size:       info.Size(),
			IndexedAt:  time.Now(),
		}

		if err := projectDB.InsertFile(file); err != nil {
			log.Printf("⚠️  Failed to insert file %s: %v", absPath, err)
			continue
		}

		// Process file and create chunks
		var chunks []db.Chunk
		var contents []string

		if strings.ToLower(filepath.Ext(absPath)) == ".pdf" {
			chunks, contents, err = processPDFFileNew(file)
			if err != nil {
				log.Printf("⚠️  Failed to process PDF %s: %v", filepath.Base(absPath), err)
				continue
			}
		} else {
			// Skip very large text files
			if info.Size() > 500*1024 {
				continue
			}

			// Read text file
			content, err := os.ReadFile(absPath)
			if err != nil {
				log.Printf("⚠️  Failed to read file %s: %v", absPath, err)
				continue
			}

			chunks, contents, err = processTextFileNew(file, string(content))
			if err != nil {
				log.Printf("⚠️  Failed to process text file %s: %v", absPath, err)
				continue
			}
		}

		allChunks = append(allChunks, chunks...)
		allContents = append(allContents, contents...)
	}

	// Generate embeddings for all chunks
	if len(allContents) > 0 {
		embeddings, err := embedder.EmbedBatch(allContents)
		if err != nil {
			return fmt.Errorf("failed to generate embeddings: %w", err)
		}

		// Insert chunks and embeddings
		for i, chunk := range allChunks {
			if err := projectDB.InsertChunk(chunk); err != nil {
				log.Printf("⚠️  Failed to insert chunk %s: %v", chunk.ChunkID, err)
				continue
			}

			if i < len(embeddings) {
				if err := projectDB.InsertEmbedding(db.Embedding{
					ChunkID: chunk.ChunkID,
					Vector:  embeddings[i],
				}); err != nil {
					log.Printf("⚠️  Failed to insert embedding for chunk %s: %v", chunk.ChunkID, err)
					continue
				}
			}
		}
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

	return nil
}

func (e *Engine) Search(query string, dbPath string, topK int) ([]SearchResult, error) {
	embedder, err := indexer.NewEmbedder(e.apiKey, e.model, e.endpoint)
	if err != nil {
		return nil, err
	}

	queryVec, err := embedder.Embed(query)
	if err != nil {
		return nil, err
	}

	projectDB, err := db.OpenProjectDB(dbPath, len(queryVec))
	if err != nil {
		return nil, err
	}
	defer projectDB.Close()

	results, err := projectDB.SearchANN(queryVec, topK)
	if err != nil {
		return nil, err
	}

	var out []SearchResult
	for _, r := range results {
		// New SearchResult already contains all needed fields from chunks table
		// Build metadata from available fields
		metadata := make(map[string]interface{})
		metadata["chunk_index"] = r.ChunkIndex
		metadata["total_chunks"] = r.TotalChunks
		if len(r.PageNumbers) > 0 {
			metadata["page_numbers"] = r.PageNumbers
		}

		out = append(out, SearchResult{
			FilePath:     r.FilePath,
			FileName:     filepath.Base(r.FilePath),
			Distance:     r.Distance,
			Content:      strings.TrimSpace(r.Content),
			TextMetadata: metadata,
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

// GetIndexStatus returns the status of an index showing pending changes
func (e *Engine) GetIndexStatus(dbPath, folderPath string) (*IndexStatus, error) {
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
	changes, err := DetectChanges(projectDB, folderPath)
	if err != nil {
		return nil, fmt.Errorf("failed to detect changes: %w", err)
	}

	status := &IndexStatus{
		ToAdd:    make([]FileStatus, 0),
		ToUpdate: make([]FileStatus, 0),
		ToDelete: make([]FileStatus, 0),
	}

	// Populate ToAdd
	for _, path := range changes.ToAdd {
		info, err := os.Stat(path)
		if err != nil {
			continue
		}
		status.ToAdd = append(status.ToAdd, FileStatus{
			Path:       path,
			FileType:   getFileType(path),
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
		status.ToUpdate = append(status.ToUpdate, FileStatus{
			Path:       path,
			FileType:   getFileType(path),
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
		status.ToDelete = append(status.ToDelete, FileStatus{
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

// Helper functions

func scanFolder(root string) ([]string, error) {
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
		if strings.HasPrefix(d.Name(), ".") || !isTextFile(path) {
			return nil
		}
		files = append(files, path)
		return nil
	})
	return files, err
}

func isTextFile(path string) bool {
	ext := strings.ToLower(filepath.Ext(path))
	textExts := map[string]bool{
		".txt": true, ".md": true, ".go": true, ".py": true, ".js": true, ".ts": true,
		".tsx": true, ".jsx": true, ".java": true, ".c": true, ".cpp": true, ".h": true,
		".rs": true, ".rb": true, ".php": true, ".cs": true, ".swift": true,
		".html": true, ".css": true, ".scss": true, ".yaml": true, ".yml": true,
		".toml": true, ".sh": true, ".proto": true, ".graphql": true,
		".pdf": true, // PDF support
	}
	return textExts[ext]
}

func generateDocID(path string) string {
	hash := md5.Sum([]byte(path))
	return hex.EncodeToString(hash[:])
}

func generateFileID(path string) string {
	hash := md5.Sum([]byte(path))
	return hex.EncodeToString(hash[:])
}

func generateChunkID(fileID string, chunkIndex int) string {
	source := fmt.Sprintf("%s:chunk:%d", fileID, chunkIndex)
	hash := md5.Sum([]byte(source))
	return hex.EncodeToString(hash[:])
}

func getFileType(path string) string {
	ext := strings.ToLower(filepath.Ext(path))
	if ext == "" {
		return "unknown"
	}
	return ext[1:]
}

// processPDFFileNew processes a PDF file and returns chunks using the new schema
func processPDFFileNew(file db.File) ([]db.Chunk, []string, error) {
	// Extract PDF text and metadata
	pdfProc := processor.NewPDFProcessor()
	pages, _, err := pdfProc.ExtractText(file.Path)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to extract PDF text: %w", err)
	}

	// Check if PDF has no text
	hasText := false
	for _, page := range pages {
		if strings.TrimSpace(page.Text) != "" {
			hasText = true
			break
		}
	}

	if !hasText {
		return nil, nil, fmt.Errorf("skipping image-only PDF")
	}

	// Chunk the PDF
	pdfChunker := processor.NewPDFChunker(processor.ChunkerConfig{
		ChunkSize:   300,
		OverlapSize: 10,
	})

	chunks, err := pdfChunker.ChunkPDFPages(pages, nil, file.Path)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to chunk PDF: %w", err)
	}

	// Convert to db.Chunk format
	var dbChunks []db.Chunk
	var contents []string

	for i, chunk := range chunks {
		// Extract page numbers from metadata if available
		var pageNumbers []int
		if chunk.Metadata != nil {
			if pages, ok := chunk.Metadata["page_numbers"].([]int); ok {
				pageNumbers = pages
			} else if pageNum, ok := chunk.Metadata["page_number"].(int); ok {
				pageNumbers = []int{pageNum}
			}
		}

		dbChunk := db.Chunk{
			ChunkID:     generateChunkID(file.FileID, i),
			FileID:      file.FileID,
			FilePath:    file.Path,     // Denormalized
			FileType:    file.FileType, // Denormalized
			ChunkIndex:  i,
			TotalChunks: len(chunks),
			Content:     chunk.Content,
			PageNumbers: pageNumbers,
		}

		dbChunks = append(dbChunks, dbChunk)
		contents = append(contents, chunk.Content)
	}

	return dbChunks, contents, nil
}

// processTextFileNew processes a text file and returns chunks using the new schema
func processTextFileNew(file db.File, content string) ([]db.Chunk, []string, error) {
	// Estimate tokens
	estimatedTokens := processor.EstimateTokens(content)

	var dbChunks []db.Chunk
	var contents []string

	if estimatedTokens > 1000 {
		// Chunk large text files
		chunker := processor.NewChunker(processor.ChunkerConfig{
			ChunkSize:   300,
			OverlapSize: 10,
		})

		chunks, err := chunker.ChunkByTokens(content, nil)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to chunk text: %w", err)
		}

		for i, chunk := range chunks {
			dbChunk := db.Chunk{
				ChunkID:     generateChunkID(file.FileID, i),
				FileID:      file.FileID,
				FilePath:    file.Path,     // Denormalized
				FileType:    file.FileType, // Denormalized
				ChunkIndex:  i,
				TotalChunks: len(chunks),
				Content:     chunk.Content,
				PageNumbers: []int{}, // Text files don't have pages
			}

			dbChunks = append(dbChunks, dbChunk)
			contents = append(contents, chunk.Content)
		}
	} else {
		// Small files: keep as single chunk
		dbChunk := db.Chunk{
			ChunkID:     generateChunkID(file.FileID, 0),
			FileID:      file.FileID,
			FilePath:    file.Path,     // Denormalized
			FileType:    file.FileType, // Denormalized
			ChunkIndex:  0,
			TotalChunks: 1,
			Content:     content,
			PageNumbers: []int{},
		}

		dbChunks = append(dbChunks, dbChunk)
		contents = append(contents, content)
	}

	return dbChunks, contents, nil
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
	fsFiles, err := scanFolder(folderPath)
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
