package engine

import (
	"context"
	"fmt"
	"log"
	"sync"
	"sync/atomic"
	"time"

	"github.com/karim-daw/qwelli/internal/db"
	"github.com/karim-daw/qwelli/internal/engine/embeddings"
)

// chunkBatch holds a batch of processed chunks ready for embedding.
type chunkBatch struct {
	chunks []db.Chunk
	files  []db.File
}

// embeddedBatch holds chunks with their computed embeddings ready for storage.
type embeddedBatch struct {
	chunks   []db.Chunk
	embedMap map[int][]float32 // index within this batch's chunks → vector
}

// pipelineStats tracks progress counters for the streaming pipeline.
type pipelineStats struct {
	filesProcessed  atomic.Int64
	chunksEmbedded  atomic.Int64
	chunksStored    atomic.Int64
	totalFiles      int
	totalChunks     atomic.Int64 // accumulated as chunks are produced
	skipped         atomic.Int64
	onedriveSkipped atomic.Int64
}

// indexFolderStreaming runs the streaming pipeline that overlaps file processing,
// embedding generation, and DB storage for maximum throughput.
func (e *Engine) indexFolderStreaming(
	ctx context.Context,
	projectDB *db.ProjectDB,
	files []string,
	maxConcurrentEmbeddings int,
	progressCb func(int, int, string),
	emitPhase func(string, string, int, int),
) (allChunks int, skipped int, onedriveSkipped int, err error) {

	if maxConcurrentEmbeddings <= 0 {
		maxConcurrentEmbeddings = 5
	}

	stats := &pipelineStats{totalFiles: len(files)}

	// Channel sizes: buffered to allow some decoupling between stages
	chunksCh := make(chan chunkBatch, maxConcurrentEmbeddings*2)
	batchesCh := make(chan chunkBatch, maxConcurrentEmbeddings*2)
	embeddedCh := make(chan embeddedBatch, maxConcurrentEmbeddings*2)
	errCh := make(chan error, 1) // first error wins

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	// Helper to send first error without blocking
	sendErr := func(e error) {
		select {
		case errCh <- e:
		default:
		}
		cancel()
	}

	var storeWg sync.WaitGroup
	var embedWg sync.WaitGroup

	// --- Stage 1: File processing workers → chunksCh ---
	go func() {
		defer close(chunksCh)
		e.processFilesToChannel(ctx, projectDB, files, chunksCh, stats, progressCb, emitPhase)
	}()

	// --- Stage 2: Batch collector goroutine: chunksCh → batchesCh ---
	go func() {
		defer close(batchesCh)
		e.collectBatches(ctx, chunksCh, batchesCh)
	}()

	// --- Stage 3: Embed workers (semaphore-bounded): batchesCh → embeddedCh ---
	embedWg.Add(1)
	go func() {
		defer embedWg.Done()

		// Ensure embeddedCh is always closed after all inner workers complete,
		// even if embedder creation fails. Without this, Stage 4 would deadlock
		// on `for eb := range embeddedCh` if this goroutine returns early.
		var innerWg sync.WaitGroup
		defer func() {
			innerWg.Wait()
			close(embeddedCh)
		}()

		embedder, embedErr := embeddings.NewEmbedder(e.voyageClient)
		if embedErr != nil {
			sendErr(embedErr)
			return
		}

		sem := make(chan struct{}, maxConcurrentEmbeddings)

		for batch := range batchesCh {
			// Acquire semaphore or bail on context cancellation
			acquired := false
			select {
			case <-ctx.Done():
			case sem <- struct{}{}:
				acquired = true
			}
			if !acquired {
				break
			}

			innerWg.Add(1)
			go func(b chunkBatch) {
				defer innerWg.Done()
				defer func() { <-sem }()

				// Create a per-goroutine generator to avoid shared mutable state
				// (imageValidator has non-thread-safe stats)
				gen := embeddings.NewEmbeddingGenerator(embedder, e.enableMultimodal)
				gen.SetCache(projectDB)

				embMap, embErr := gen.GenerateEmbeddings(ctx, b.chunks, nil)
				if embErr != nil {
					if ctx.Err() != nil {
						return
					}
					sendErr(fmt.Errorf("generate embeddings: %w", embErr))
					return
				}

				embedded := stats.chunksEmbedded.Add(int64(len(embMap)))
				total := stats.totalChunks.Load()
				if total > 0 {
					emitPhase("embedding", fmt.Sprintf("Generating embeddings: %d/%d", embedded, total), int(embedded), int(total))
				}

				select {
				case <-ctx.Done():
				case embeddedCh <- embeddedBatch{chunks: b.chunks, embedMap: embMap}:
				}
			}(batch)
		}

		// defer handles innerWg.Wait() + close(embeddedCh)
	}()

	// --- Stage 4: Store goroutine (single writer): embeddedCh → DB ---
	storeWg.Add(1)
	go func() {
		defer storeWg.Done()

		for eb := range embeddedCh {
			select {
			case <-ctx.Done():
				return
			default:
			}

			emitPhase("storing", fmt.Sprintf("Storing batch of %d chunks", len(eb.chunks)), int(stats.chunksStored.Load()), int(stats.totalChunks.Load()))

			if storeErr := embeddings.StoreChunksAndEmbeddings(projectDB, eb.chunks, eb.embedMap); storeErr != nil {
				sendErr(fmt.Errorf("store chunks: %w", storeErr))
				return
			}

			stored := stats.chunksStored.Add(int64(len(eb.chunks)))
			total := stats.totalChunks.Load()
			emitPhase("storing", fmt.Sprintf("Stored %d/%d chunks", stored, total), int(stored), int(total))
		}
	}()

	// Wait for all stages to complete
	embedWg.Wait()
	storeWg.Wait()

	// Check for errors
	select {
	case pipelineErr := <-errCh:
		if pipelineErr != nil {
			return 0, int(stats.skipped.Load()), int(stats.onedriveSkipped.Load()), pipelineErr
		}
	default:
	}

	if ctx.Err() != nil {
		return 0, int(stats.skipped.Load()), int(stats.onedriveSkipped.Load()), ctx.Err()
	}

	return int(stats.totalChunks.Load()), int(stats.skipped.Load()), int(stats.onedriveSkipped.Load()), nil
}

// collectBatches reads individual file chunk batches from chunksCh and groups them
// into larger embedding-sized batches for efficient API calls.
func (e *Engine) collectBatches(ctx context.Context, chunksCh <-chan chunkBatch, batchesCh chan<- chunkBatch) {
	const targetBatchSize = 950 // Match embedding API batch input limit

	var accumulated chunkBatch
	accumulated.chunks = make([]db.Chunk, 0, targetBatchSize)
	accumulated.files = make([]db.File, 0, 64)

	flush := func() {
		if len(accumulated.chunks) == 0 {
			return
		}
		select {
		case <-ctx.Done():
			return
		case batchesCh <- accumulated:
		}
		accumulated = chunkBatch{
			chunks: make([]db.Chunk, 0, targetBatchSize),
			files:  make([]db.File, 0, 64),
		}
	}

	for cb := range chunksCh {
		select {
		case <-ctx.Done():
			return
		default:
		}

		accumulated.chunks = append(accumulated.chunks, cb.chunks...)
		accumulated.files = append(accumulated.files, cb.files...)

		if len(accumulated.chunks) >= targetBatchSize {
			flush()
		}
	}

	// Flush remaining
	flush()
}

// processFilesToChannel processes files using the worker pool and sends results
// to the chunksCh channel instead of accumulating in memory.
func (e *Engine) processFilesToChannel(
	ctx context.Context,
	projectDB *db.ProjectDB,
	files []string,
	chunksCh chan<- chunkBatch,
	stats *pipelineStats,
	progressCb func(int, int, string),
	emitPhase func(string, string, int, int),
) {
	numWorkers := e.numWorkers
	if numWorkers <= 0 {
		numWorkers = 1
	}

	log.Printf("⚡ Streaming pipeline: %d file workers", numWorkers)
	emitPhase("processing", fmt.Sprintf("Processing %d files", len(files)), 0, len(files))

	jobs := make(chan int, numWorkers*2)
	var wg sync.WaitGroup

	// File insert buffer: collect files for batch append
	var (
		fileBuf []db.File
		fileMu  sync.Mutex
	)
	fileBuf = make([]db.File, 0, len(files))

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

				current := stats.filesProcessed.Add(1)
				if progressCb != nil {
					progressCb(int(current), len(files), files[idx])
				}
				emitPhase("processing", fmt.Sprintf("Processing files: %d/%d", current, len(files)), int(current), len(files))

				if result.skipped {
					stats.skipped.Add(1)
					if result.onedriveSkipped {
						stats.onedriveSkipped.Add(1)
					}
					continue
				}
				if result.err != nil {
					log.Printf("⚠️  process %s: %v", files[idx], result.err)
					continue
				}

				// Buffer the file for batch insert
				fileMu.Lock()
				fileBuf = append(fileBuf, result.file)
				fileMu.Unlock()

				if len(result.chunks) > 0 {
					stats.totalChunks.Add(int64(len(result.chunks)))

					select {
					case <-ctx.Done():
						return
					case chunksCh <- chunkBatch{
						chunks: result.chunks,
						files:  []db.File{result.file},
					}:
					}
				}
			}
		}()
	}

	// Feed jobs
	for i := range files {
		select {
		case <-ctx.Done():
			break
		case jobs <- i:
		}
	}
	close(jobs)
	wg.Wait()

	// Batch-append all files to DB after workers complete
	fileMu.Lock()
	toInsert := fileBuf
	fileMu.Unlock()

	if len(toInsert) > 0 {
		start := time.Now()
		if err := projectDB.AppendFiles(toInsert); err != nil {
			log.Printf("⚠️  Bulk insert files failed: %v", err)
			for _, f := range toInsert {
				if err := projectDB.InsertFile(f); err != nil {
					log.Printf("⚠️  insert file %s: %v", f.Path, err)
				}
			}
		}
		log.Printf("💾 Inserted %d files in %v", len(toInsert), time.Since(start).Round(time.Millisecond))
	}
}
