package engine

import (
	"bytes"
	"crypto/md5"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"image"
	_ "image/jpeg"
	_ "image/png"
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
	apiKey           string
	model            string
	endpoint         string
	providerType     string
	enableMultimodal bool
}

func NewEngine(apiKey, model, endpoint string) *Engine {
	return &Engine{
		apiKey:           apiKey,
		model:            model,
		endpoint:         endpoint,
		providerType:     "voyage",
		enableMultimodal: true, // Default to true for Voyage
	}
}

func NewEngineWithProvider(apiKey, model, endpoint, providerType string, enableMultimodal bool) *Engine {
	return &Engine{
		apiKey:           apiKey,
		model:            model,
		endpoint:         endpoint,
		providerType:     providerType,
		enableMultimodal: enableMultimodal,
	}
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
	embedder, err := indexer.NewEmbedder(e.providerType, e.apiKey, e.model, e.endpoint)
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
		changes, err := detectChanges(projectDB, folderPath)
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

		if strings.ToLower(filepath.Ext(absPath)) == ".pdf" {
			if e.enableMultimodal && embedder.IsMultimodal() {
				chunks, _, err = processPDFFileMultimodal(file)
			} else {
				chunks, _, err = processPDFFileNew(file)
			}
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

			chunks, _, err = processTextFileNew(file, string(content))
			if err != nil {
				log.Printf("⚠️  Failed to process text file %s: %v", absPath, err)
				continue
			}
		}

		allChunks = append(allChunks, chunks...)
	}

	// Generate embeddings for all chunks
	if len(allChunks) > 0 {
		var embeddings [][]float32
		var err error

		// Check if we have multimodal chunks (images)
		hasImages := false
		for _, chunk := range allChunks {
			if chunk.ContentType == "image" {
				hasImages = true
				break
			}
		}

		if hasImages && embedder.IsMultimodal() {
			// Use multimodal embedding
			multimodalInputs := make([]indexer.MultimodalInput, 0, len(allChunks))
			validChunkIndices := make([]int, 0, len(allChunks))

			// Statistics for image validation
			imageSkipStats := struct {
				emptyBase64      int
				invalidBase64    int
				emptyDecoded     int
				tooSmallBytes    int
				tooLargeBytes    int
				invalidFormat    int
				decodeFailed     int
				tooSmallDims     int
				tooLargeDims     int
				badAspectRatio   int
				exceedsVoyageMax int
				reencodeFailed   int
				accepted         int
			}{}

			for i, chunk := range allChunks {
				if chunk.ContentType == "image" {
					// Image chunk - ImageData contains base64 string as bytes
					imageBase64 := string(chunk.ImageData)

					// Validate base64 - skip if empty or invalid
					if imageBase64 == "" || len(imageBase64) < 10 {
						imageSkipStats.emptyBase64++
						continue
					}

					// Remove any whitespace/newlines that might have been introduced
					imageBase64 = strings.TrimSpace(imageBase64)
					imageBase64 = strings.ReplaceAll(imageBase64, "\n", "")
					imageBase64 = strings.ReplaceAll(imageBase64, "\r", "")

					// Remove data URI prefix if present (e.g., "data:image/png;base64,")
					if strings.HasPrefix(imageBase64, "data:") {
						// Find the comma after the base64 declaration
						commaIdx := strings.Index(imageBase64, ",")
						if commaIdx > 0 && commaIdx < len(imageBase64)-1 {
							imageBase64 = imageBase64[commaIdx+1:]
						}
					}

					// Validate base64 by attempting to decode it
					decoded, err := base64.StdEncoding.DecodeString(imageBase64)
					if err != nil {
						imageSkipStats.invalidBase64++
						continue
					}

					// Additional validation: decoded data should be reasonable size
					if len(decoded) == 0 {
						imageSkipStats.emptyDecoded++
						continue
					}

					// Skip images that are too small (likely corrupted or placeholder)
					// Minimum reasonable image size is ~100 bytes decoded (about 133 bytes base64)
					minImageSize := 100
					if len(decoded) < minImageSize {
						imageSkipStats.tooSmallBytes++
						continue
					}

					// Voyage API has size limits - check if image is too large
					// Typical limit is around 20MB for base64, which is ~15MB decoded
					// Let's be conservative and limit to 10MB decoded (about 13.3MB base64)
					maxImageSize := 10 * 1024 * 1024 // 10MB
					if len(decoded) > maxImageSize {
						imageSkipStats.tooLargeBytes++
						continue
					}

					// Check if it's a valid image format (basic check - should start with image magic bytes)
					// JPEG: FF D8 FF, PNG: 89 50 4E 47, GIF: 47 49 46 38
					if len(decoded) < 4 {
						imageSkipStats.tooSmallBytes++
						continue
					}

					isValidImage := (decoded[0] == 0xFF && decoded[1] == 0xD8 && decoded[2] == 0xFF) || // JPEG
						(decoded[0] == 0x89 && decoded[1] == 0x50 && decoded[2] == 0x4E && decoded[3] == 0x47) || // PNG
						(decoded[0] == 0x47 && decoded[1] == 0x49 && decoded[2] == 0x46 && decoded[3] == 0x38) // GIF

					if !isValidImage {
						imageSkipStats.invalidFormat++
						continue
					}

					// Additional check: try to decode and verify image can be parsed
					// This catches issues that basic validation might miss
					img, format, err := image.Decode(bytes.NewReader(decoded))
					if err != nil {
						imageSkipStats.decodeFailed++
						continue
					}

					// Get image dimensions
					bounds := img.Bounds()
					width := bounds.Dx()
					height := bounds.Dy()
					pixelCount := width * height

					// Skip images that are too small
					// Minimum: ~1/3 of A4 page size
					// A4 at 150 DPI: ~1240x1754 pixels
					// 1/3 of A4: ~700x500 pixels = ~350,000 pixels
					// We'll use 500x350 pixels (175,000 pixels) as minimum to ensure meaningful content
					minWidth := 500
					minHeight := 350
					minPixels := 175_000 // ~1/3 of A4 page at reasonable resolution
					if width < minWidth || height < minHeight || pixelCount < minPixels {
						imageSkipStats.tooSmallDims++
						continue
					}

					// Skip images that are too large (max dimensions and pixel count)
					// Maximum: 4000x3000 pixels (12M pixels) - reasonable upper limit for search
					// Voyage API limit is 16M pixels, but we'll be more conservative
					maxWidth := 4000
					maxHeight := 3000
					maxPixels := 12_000_000 // 12 million pixels
					if width > maxWidth || height > maxHeight || pixelCount > maxPixels {
						imageSkipStats.tooLargeDims++
						continue
					}

					// Check aspect ratio (Voyage API has limits on aspect ratio)
					// Aspect ratio = width / height
					// Common limits are typically between 0.1 and 10.0
					aspectRatio := float64(width) / float64(height)
					minAspectRatio := 0.1  // Very tall images (1:10)
					maxAspectRatio := 10.0 // Very wide images (10:1)
					if aspectRatio < minAspectRatio || aspectRatio > maxAspectRatio {
						imageSkipStats.badAspectRatio++
						continue
					}

					// Note: We already check max dimensions above (12M pixels)
					// This check is redundant but kept for Voyage API compliance (16M pixels is their absolute limit)
					voyageMaxPixels := 16_000_000
					if pixelCount > voyageMaxPixels {
						imageSkipStats.exceedsVoyageMax++
						continue
					}

					// Check file size (Voyage limit is 20MB, we already check 10MB above, but double-check)
					maxImageSizeBytes := 20 * 1024 * 1024 // 20MB
					if len(decoded) > maxImageSizeBytes {
						imageSkipStats.tooLargeBytes++
						continue
					}

					// Re-encode base64 to ensure it's clean and properly formatted
					// This helps catch any encoding issues
					cleanBase64 := base64.StdEncoding.EncodeToString(decoded)

					// Verify the re-encoded base64 can be decoded back
					testDecoded, err := base64.StdEncoding.DecodeString(cleanBase64)
					if err != nil || len(testDecoded) != len(decoded) {
						imageSkipStats.reencodeFailed++
						continue
					}

					// Image passed all validation
					imageSkipStats.accepted++

					// Normalize format name (image.Decode returns "jpeg" but we need lowercase)
					formatLower := strings.ToLower(format)

					multimodalInputs = append(multimodalInputs, indexer.MultimodalInput{
						Type:        "image",
						ImageBase64: cleanBase64, // Use re-encoded clean base64 (without data URI prefix)
						ImageFormat: formatLower, // Format: "jpeg", "png", "gif", etc.
						ImagePixels: pixelCount,  // Pixel count for token calculation (560 pixels = 1 token)
					})
					validChunkIndices = append(validChunkIndices, i)
				} else {
					// Text chunk - skip if empty
					if chunk.Content == "" {
						log.Printf("⚠️  Skipping text chunk %d: empty content", i)
						continue
					}

					multimodalInputs = append(multimodalInputs, indexer.MultimodalInput{
						Type: "text",
						Text: chunk.Content,
					})
					validChunkIndices = append(validChunkIndices, i)
				}
			}

			// Log summary of image validation
			totalSkipped := imageSkipStats.emptyBase64 + imageSkipStats.invalidBase64 + imageSkipStats.emptyDecoded +
				imageSkipStats.tooSmallBytes + imageSkipStats.tooLargeBytes + imageSkipStats.invalidFormat +
				imageSkipStats.decodeFailed + imageSkipStats.tooSmallDims + imageSkipStats.tooLargeDims +
				imageSkipStats.badAspectRatio + imageSkipStats.exceedsVoyageMax + imageSkipStats.reencodeFailed
			if totalSkipped > 0 || imageSkipStats.accepted > 0 {
				skipReasons := []string{}
				if imageSkipStats.emptyBase64 > 0 {
					skipReasons = append(skipReasons, fmt.Sprintf("%d empty", imageSkipStats.emptyBase64))
				}
				if imageSkipStats.invalidBase64 > 0 {
					skipReasons = append(skipReasons, fmt.Sprintf("%d invalid base64", imageSkipStats.invalidBase64))
				}
				if imageSkipStats.tooSmallDims > 0 {
					skipReasons = append(skipReasons, fmt.Sprintf("%d too small", imageSkipStats.tooSmallDims))
				}
				if imageSkipStats.tooLargeDims > 0 {
					skipReasons = append(skipReasons, fmt.Sprintf("%d too large", imageSkipStats.tooLargeDims))
				}
				if imageSkipStats.badAspectRatio > 0 {
					skipReasons = append(skipReasons, fmt.Sprintf("%d bad aspect ratio", imageSkipStats.badAspectRatio))
				}
				if imageSkipStats.decodeFailed > 0 {
					skipReasons = append(skipReasons, fmt.Sprintf("%d decode failed", imageSkipStats.decodeFailed))
				}
				if imageSkipStats.invalidFormat > 0 {
					skipReasons = append(skipReasons, fmt.Sprintf("%d invalid format", imageSkipStats.invalidFormat))
				}
				if imageSkipStats.tooLargeBytes > 0 {
					skipReasons = append(skipReasons, fmt.Sprintf("%d too large (bytes)", imageSkipStats.tooLargeBytes))
				}
				if len(skipReasons) > 0 {
					log.Printf("📊 Image validation: %d accepted, %d skipped (%s)", imageSkipStats.accepted, totalSkipped, strings.Join(skipReasons, ", "))
				} else {
					log.Printf("📊 Image validation: %d accepted", imageSkipStats.accepted)
				}
			}

			if len(multimodalInputs) == 0 {
				return fmt.Errorf("no valid chunks to embed")
			}

			// Log batch composition for debugging
			imageCount := 0
			textCount := 0
			for _, inp := range multimodalInputs {
				if inp.Type == "image" {
					imageCount++
				} else {
					textCount++
				}
			}
			log.Printf("  Batch contains %d text inputs and %d image inputs", textCount, imageCount)

			embeddings, err = embedder.EmbedMultimodal(multimodalInputs)
			if err != nil {
				return fmt.Errorf("failed to generate embeddings: %w", err)
			}

			// Map embeddings back to valid chunks
			if len(embeddings) != len(validChunkIndices) {
				return fmt.Errorf("embedding count mismatch: got %d embeddings for %d valid chunks", len(embeddings), len(validChunkIndices))
			}

			// Create a map of chunk index to embedding
			embeddingMap := make(map[int][]float32)
			for j, idx := range validChunkIndices {
				if j < len(embeddings) {
					embeddingMap[idx] = embeddings[j]
				}
			}

			// Insert chunks and embeddings
			for i, chunk := range allChunks {
				if err := projectDB.InsertChunk(chunk); err != nil {
					log.Printf("⚠️  Failed to insert chunk %s: %v", chunk.ChunkID, err)
					continue
				}

				// Only insert embedding if this chunk was successfully embedded
				if emb, ok := embeddingMap[i]; ok {
					if err := projectDB.InsertEmbedding(db.Embedding{
						ChunkID: chunk.ChunkID,
						Vector:  emb,
					}); err != nil {
						log.Printf("⚠️  Failed to insert embedding for chunk %s: %v", chunk.ChunkID, err)
						continue
					}
				}
			}
		} else {
			// Use text-only embedding
			texts := make([]string, 0, len(allChunks))
			validChunkIndices := make([]int, 0, len(allChunks))

			for i, chunk := range allChunks {
				if chunk.Content == "" {
					log.Printf("⚠️  Skipping text chunk %d: empty content", i)
					continue
				}
				texts = append(texts, chunk.Content)
				validChunkIndices = append(validChunkIndices, i)
			}

			if len(texts) == 0 {
				return fmt.Errorf("no valid chunks to embed")
			}

			embeddings, err = embedder.EmbedBatch(texts)
			if err != nil {
				return fmt.Errorf("failed to generate embeddings: %w", err)
			}

			// Map embeddings back to valid chunks
			if len(embeddings) != len(validChunkIndices) {
				return fmt.Errorf("embedding count mismatch: got %d embeddings for %d valid chunks", len(embeddings), len(validChunkIndices))
			}

			// Create a map of chunk index to embedding
			embeddingMap := make(map[int][]float32)
			for j, idx := range validChunkIndices {
				if j < len(embeddings) {
					embeddingMap[idx] = embeddings[j]
				}
			}

			// Insert chunks and embeddings
			for i, chunk := range allChunks {
				if err := projectDB.InsertChunk(chunk); err != nil {
					log.Printf("⚠️  Failed to insert chunk %s: %v", chunk.ChunkID, err)
					continue
				}

				// Only insert embedding if this chunk was successfully embedded
				if emb, ok := embeddingMap[i]; ok {
					if err := projectDB.InsertEmbedding(db.Embedding{
						ChunkID: chunk.ChunkID,
						Vector:  emb,
					}); err != nil {
						log.Printf("⚠️  Failed to insert embedding for chunk %s: %v", chunk.ChunkID, err)
						continue
					}
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
	return e.SearchWithFilter(query, dbPath, topK, "")
}

func (e *Engine) SearchWithFilter(query string, dbPath string, topK int, contentType string) ([]SearchResult, error) {
	embedder, err := indexer.NewEmbedder(e.providerType, e.apiKey, e.model, e.endpoint)
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

	var results []db.SearchResult
	if contentType != "" {
		results, err = projectDB.SearchANNWithFilter(queryVec, topK, contentType)
	} else {
		results, err = projectDB.SearchANN(queryVec, topK)
	}
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
	changes, err := detectChanges(projectDB, folderPath)
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

// processPDFFileMultimodal processes a PDF file with multimodal support (text + images)
func processPDFFileMultimodal(file db.File) ([]db.Chunk, []string, error) {
	// Extract PDF text and metadata
	pdfProc := processor.NewPDFProcessor()
	pages, metadata, err := pdfProc.ExtractText(file.Path)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to extract PDF text: %w", err)
	}

	// Extract images
	imageExtractor := processor.NewImageExtractor(1024, 1024)
	images, err := imageExtractor.ExtractImages(file.Path)
	if err != nil {
		log.Printf("⚠️  Failed to extract images from %s: %v (continuing with text only)", filepath.Base(file.Path), err)
		images = []processor.PDFImage{}
	}

	// Create multimodal chunker
	pdfChunker := processor.NewPDFChunker(processor.ChunkerConfig{
		ChunkSize:   300,
		OverlapSize: 10,
	})
	multimodalChunker := processor.NewMultimodalChunker(pdfChunker, imageExtractor)

	// Chunk PDF with multimodal support
	multimodalChunks, err := multimodalChunker.ChunkPDF(pages, images, metadata, file.Path)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to chunk PDF: %w", err)
	}

	// Convert to db.Chunk format
	var dbChunks []db.Chunk
	var contents []string

	for i, mc := range multimodalChunks {
		pageNumbers := []int{mc.PageNumber}

		dbChunk := db.Chunk{
			ChunkID:     generateChunkID(file.FileID, i),
			FileID:      file.FileID,
			FilePath:    file.Path,
			FileType:    file.FileType,
			ChunkIndex:  i,
			TotalChunks: len(multimodalChunks),
			Content:     mc.Content,
			PageNumbers: pageNumbers,
			ContentType: mc.ContentType,
			ImageData:   mc.ImageData, // Already base64 string as bytes
		}

		dbChunks = append(dbChunks, dbChunk)

		// For embedding, use text content for text chunks
		// Images will be handled separately in the embedding generation
		contents = append(contents, mc.Content)
	}

	return dbChunks, contents, nil
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
func detectChanges(projectDB *db.ProjectDB, folderPath string) (*ChangeSet, error) {
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
