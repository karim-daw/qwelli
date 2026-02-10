package engine

import (
	"context"
	"log"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/karim-daw/qwelli/internal/db"
	"github.com/karim-daw/qwelli/internal/engine/differ"
	"github.com/karim-daw/qwelli/internal/engine/extraction"
	"github.com/karim-daw/qwelli/internal/engine/fileprocessor"
)

// fileProcessResult holds the result of processing a single file.
type fileProcessResult struct {
	file            db.File
	chunks          []db.Chunk
	err             error
	skipped         bool
	onedriveSkipped bool
}

// processFilesParallel processes files concurrently using a worker pool.
// It uses a semaphore pattern with a bounded number of goroutines to prevent
// resource exhaustion while maximizing throughput for CPU-bound operations
// like PDF extraction, image decoding, and SHA256 hashing.
func (e *Engine) processFilesParallel(ctx context.Context, projectDB *db.ProjectDB, files []string, numWorkers int, progressCb func(int, int, string)) ([]db.Chunk, int, int) {

	if numWorkers <= 0 {
		// Target ~90% CPU utilization by default
		numWorkers = int(float64(runtime.NumCPU()) * 0.9)
		if numWorkers < 1 {
			numWorkers = 1
		}
	}

	log.Printf("⚡ Parallel file processing: %d workers (CPUs: %d)", numWorkers, runtime.NumCPU())

	// Result collection: accumulate files and chunks for batch append (Appender is not thread-safe)
	var (
		allFiles       []db.File
		allChunks      []db.Chunk
		skipped        int
		onedriveSkipped int
		mu             sync.Mutex // Protects allFiles, allChunks, skipped, onedriveSkipped
	)

	// Atomic counter for progress reporting
	var processed atomic.Int64

	// Job channel acts as a work queue - bounded to prevent memory bloat
	jobs := make(chan int, numWorkers*2)

	// WaitGroup tracks worker completion
	var wg sync.WaitGroup

	// Launch worker pool
	for w := 0; w < numWorkers; w++ {
		wg.Add(1)
		go func() {
			defer wg.Done()

			for idx := range jobs {
				select {
				case <-ctx.Done():
					return
				default:
				}

				result := e.processOneFile(files[idx])

				// Atomically update progress
				current := processed.Add(1)
				if progressCb != nil {
					progressCb(int(current), len(files), files[idx])
				}

				// Lock only for shared state mutation
				mu.Lock()
				if result.skipped {
					skipped++
					if result.onedriveSkipped {
						onedriveSkipped++
						if onedriveSkipped <= 3 {
							log.Printf("⚠️  Skipping OneDrive placeholder: %s", filepath.Base(files[idx]))
						}
					}
				} else if result.err != nil {
					log.Printf("⚠️  process %s: %v", filepath.Base(files[idx]), result.err)
				} else {
					allFiles = append(allFiles, result.file)
					if len(result.chunks) > 0 {
						allChunks = append(allChunks, result.chunks...)
					}
				}
				mu.Unlock()
			}
		}()
	}

	// Produce jobs - feed file indices into the job channel
	for i := range files {
		select {
		case <-ctx.Done():
			break
		case jobs <- i:
		}
	}
	close(jobs)

	// Wait for all workers to finish
	wg.Wait()

	// Batch-append files via Appender (not thread-safe, so done after workers complete)
	if len(allFiles) > 0 {
		if err := projectDB.AppendFiles(allFiles); err != nil {
			log.Printf("⚠️  Bulk insert files failed: %v", err)
			// Fallback to row-by-row
			for _, f := range allFiles {
				if err := projectDB.InsertFile(f); err != nil {
					log.Printf("⚠️  insert file %s: %v", f.Path, err)
				}
			}
		}
	}

	return allChunks, skipped, onedriveSkipped
}

const maxSingleReadSize = 50 * 1024 * 1024 // 50MB - above this use streaming hash

// processOneFile handles the CPU-bound work for a single file:
// path resolution, stat, hashing, and content processing.
// Reads each file once from disk when under maxSingleReadSize to avoid redundant I/O.
func (e *Engine) processOneFile(f string) fileProcessResult {
	absPath, err := filepath.Abs(f)
	if err != nil {
		return fileProcessResult{skipped: true}
	}

	info, err := os.Stat(absPath)
	if err != nil {
		log.Printf("⚠️  stat %s: %v", absPath, err)
		return fileProcessResult{skipped: true}
	}

	file := db.File{
		FileID:     differ.GenerateFileID(absPath),
		Path:       absPath,
		FileType:   fileprocessor.GetFileTypeFromPath(absPath),
		ModifiedAt: info.ModTime(),
		Size:       info.Size(),
		IndexedAt:  time.Now(),
	}

	if !e.fileProcessingService.CanProcess(file.FileType) {
		return fileProcessResult{file: file, skipped: false}
	}

	// Single read path for files under 50MB: read once, hash from bytes, process from bytes
	if info.Size() <= maxSingleReadSize && info.Size() >= 0 {
		data, readErr := os.ReadFile(absPath)
		if readErr != nil {
			isOnedrive := strings.Contains(readErr.Error(), "OneDrive placeholder") || strings.Contains(readErr.Error(), "input/output error")
			if !isOnedrive {
				log.Printf("⚠️  read %s: %v", absPath, readErr)
			}
			return fileProcessResult{skipped: true, onedriveSkipped: isOnedrive}
		}
		file.FileHash = extraction.ComputeSHA256FromBytes(data)

		var chunks []db.Chunk
		if file.FileType == "pdf" {
			chunks, _, err = e.fileProcessingService.ProcessPDFFromBytes(file, data)
		} else if fileprocessor.IsImageFile(file.FileType) {
			chunks, _, err = e.fileProcessingService.ProcessImageFromBytes(file, data)
		} else {
			if info.Size() > 500*1024 {
				return fileProcessResult{file: file, skipped: false}
			}
			chunks, _, err = e.fileProcessingService.ProcessText(file, string(data))
		}
		if err != nil {
			return fileProcessResult{file: file, err: err}
		}
		return fileProcessResult{file: file, chunks: chunks}
	}

	// Large files (>50MB): use streaming hash to avoid loading into memory
	fileHash, err := extraction.ComputeSHA256(absPath)
	if err != nil {
		isOnedrive := strings.Contains(err.Error(), "OneDrive placeholder") || strings.Contains(err.Error(), "input/output error")
		if !isOnedrive {
			log.Printf("⚠️  hash %s: %v", absPath, err)
		}
		return fileProcessResult{skipped: true, onedriveSkipped: isOnedrive}
	}
	file.FileHash = fileHash

	// Re-read for processing (large files - no single-read optimization)
	var chunks []db.Chunk
	if file.FileType == "pdf" {
		chunks, _, err = e.fileProcessingService.ProcessPDF(file)
	} else if fileprocessor.IsImageFile(file.FileType) {
		chunks, _, err = e.fileProcessingService.ProcessImage(file)
	} else {
		if info.Size() > 500*1024 {
			return fileProcessResult{file: file, skipped: false}
		}
		content, readErr := os.ReadFile(absPath)
		if readErr != nil {
			log.Printf("⚠️  read %s: %v", absPath, readErr)
			return fileProcessResult{file: file, skipped: false}
		}
		chunks, _, err = e.fileProcessingService.ProcessText(file, string(content))
	}
	if err != nil {
		return fileProcessResult{file: file, err: err}
	}
	return fileProcessResult{file: file, chunks: chunks}
}
