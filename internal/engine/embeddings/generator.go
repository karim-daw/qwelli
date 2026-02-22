package embeddings

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"log"

	"github.com/karim-daw/qwelli/internal/db"
	"github.com/karim-daw/qwelli/internal/engine/extraction"
)

// EmbeddingCache is the interface for looking up and storing cached embeddings.
type EmbeddingCache interface {
	LookupEmbeddingCache(hashes []string) (map[string][]float32, error)
	StoreEmbeddingCache(entries []db.CacheEntry) error
}

// EmbeddingGenerator handles embedding generation for chunks
type EmbeddingGenerator struct {
	embedder         *Embedder
	imageValidator   *extraction.ImageValidator
	enableMultimodal bool
	cache            EmbeddingCache
}

func NewEmbeddingGenerator(embedder *Embedder, enableMultimodal bool) *EmbeddingGenerator {
	return &EmbeddingGenerator{
		embedder:         embedder,
		imageValidator:   extraction.NewImageValidator(),
		enableMultimodal: enableMultimodal,
	}
}

// SetCache enables the embedding cache for this generator.
func (g *EmbeddingGenerator) SetCache(cache EmbeddingCache) {
	g.cache = cache
}

// contentHash computes SHA256 of the chunk's embeddable content.
func contentHash(chunk db.Chunk) string {
	h := sha256.New()
	if chunk.ContentType == "image" && len(chunk.ImageData) > 0 {
		h.Write(chunk.ImageData)
	} else {
		h.Write([]byte(chunk.Content))
	}
	return hex.EncodeToString(h.Sum(nil))
}

// GenerateEmbeddings generates embeddings for chunks, returning chunkIndex -> vector.
// When a cache is set, it checks the cache first and only sends cache-misses to the API.
func (g *EmbeddingGenerator) GenerateEmbeddings(ctx context.Context, chunks []db.Chunk, progressCb func(int, int)) (map[int][]float32, error) {
	if len(chunks) == 0 {
		return nil, fmt.Errorf("no chunks to embed")
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	result := make(map[int][]float32, len(chunks))

	// --- Cache lookup phase ---
	var (
		chunkHashes []string                 // hash for each chunk index
		hashToIdx   = make(map[string][]int) // hash → chunk indices with that hash
	)

	if g.cache != nil {
		chunkHashes = make([]string, len(chunks))
		uniqueHashes := make([]string, 0, len(chunks))

		for i, chunk := range chunks {
			h := contentHash(chunk)
			chunkHashes[i] = h
			if _, seen := hashToIdx[h]; !seen {
				uniqueHashes = append(uniqueHashes, h)
			}
			hashToIdx[h] = append(hashToIdx[h], i)
		}

		cached, err := g.cache.LookupEmbeddingCache(uniqueHashes)
		if err != nil {
			log.Printf("⚠️  Embedding cache lookup failed: %v", err)
			// Continue without cache
		} else if len(cached) > 0 {
			hits := 0
			for hash, vec := range cached {
				for _, idx := range hashToIdx[hash] {
					result[idx] = vec
					hits++
				}
			}
			log.Printf("  Embedding cache: %d hits, %d misses", hits, len(chunks)-hits)

			// If all chunks are cached, we're done
			if len(result) == len(chunks) {
				if progressCb != nil {
					progressCb(len(chunks), len(chunks))
				}
				return result, nil
			}
		}
	}

	// --- Build inputs for uncached chunks ---
	hasImages := false
	for _, c := range chunks {
		if c.ContentType == "image" {
			hasImages = true
			break
		}
	}
	useMultimodal := hasImages && g.enableMultimodal && g.embedder.IsMultimodal()

	var (
		validIndices []int
		textInputs   []string
		mmInputs     []MultimodalInput
	)

	if useMultimodal {
		g.imageValidator.ResetStats()
	}

	for i, chunk := range chunks {
		// Skip already-cached chunks
		if _, cached := result[i]; cached {
			continue
		}

		if useMultimodal && chunk.ContentType == "image" {
			imgData, valid := g.imageValidator.ValidateAndPrepare(string(chunk.ImageData), i)
			if !valid {
				continue
			}
			mmInputs = append(mmInputs, MultimodalInput{
				Type: "image", ImageBase64: imgData.Base64,
				ImageFormat: imgData.Format, ImagePixels: imgData.Pixels,
			})
			validIndices = append(validIndices, i)
		} else {
			if chunk.Content == "" {
				log.Printf("⚠️  Skipping empty chunk %d", i)
				continue
			}
			if useMultimodal {
				mmInputs = append(mmInputs, MultimodalInput{Type: "text", Text: chunk.Content})
			} else {
				textInputs = append(textInputs, chunk.Content)
			}
			validIndices = append(validIndices, i)
		}
	}

	if useMultimodal {
		if msg := g.imageValidator.FormatStats(); msg != "" {
			log.Print(msg)
		}
	}

	if len(validIndices) == 0 {
		// All chunks were cached or invalid
		if len(result) > 0 {
			return result, nil
		}
		return nil, fmt.Errorf("no valid chunks to embed")
	}

	// --- Generate embeddings via API ---
	var (
		apiEmbeddings [][]float32
		err           error
	)
	if useMultimodal {
		apiEmbeddings, err = g.embedder.EmbedMultimodal(ctx, mmInputs, progressCb)
	} else {
		apiEmbeddings, err = g.embedder.EmbedBatch(ctx, textInputs, progressCb)
	}
	if err != nil {
		if ctx.Err() != nil {
			return nil, ctx.Err()
		}
		return nil, fmt.Errorf("generate embeddings: %w", err)
	}

	if len(apiEmbeddings) != len(validIndices) {
		return nil, fmt.Errorf("embedding count mismatch: got %d for %d chunks", len(apiEmbeddings), len(validIndices))
	}

	// Map API results back to chunk indices
	var newCacheEntries []db.CacheEntry
	for j, idx := range validIndices {
		result[idx] = apiEmbeddings[j]

		// Store in cache
		if g.cache != nil && len(chunkHashes) > idx {
			newCacheEntries = append(newCacheEntries, db.CacheEntry{
				ContentHash: chunkHashes[idx],
				Vector:      apiEmbeddings[j],
			})
		}
	}

	// --- Store new embeddings in cache ---
	if g.cache != nil && len(newCacheEntries) > 0 {
		if err := g.cache.StoreEmbeddingCache(newCacheEntries); err != nil {
			log.Printf("⚠️  Failed to store embedding cache: %v", err)
			// Non-fatal: continue without caching
		}
	}

	return result, nil
}

// StoreChunksAndEmbeddings stores chunks and their embeddings in the database using bulk Appender API.
func StoreChunksAndEmbeddings(projectDB *db.ProjectDB, chunks []db.Chunk, embeddingMap map[int][]float32) error {
	if len(chunks) == 0 {
		return nil
	}
	if err := projectDB.AppendChunksAndEmbeddings(chunks, embeddingMap); err != nil {
		log.Printf("⚠️  Bulk insert failed, falling back to row-by-row: %v", err)
		// Fallback to legacy path
		for i, chunk := range chunks {
			if err := projectDB.InsertChunk(chunk); err != nil {
				log.Printf("⚠️  Failed to insert chunk %s: %v", chunk.ChunkID, err)
				continue
			}
			if emb, ok := embeddingMap[i]; ok {
				if err := projectDB.InsertEmbedding(db.Embedding{ChunkID: chunk.ChunkID, Vector: emb}); err != nil {
					log.Printf("⚠️  Failed to insert embedding for %s: %v", chunk.ChunkID, err)
				}
			}
		}
	}
	return nil
}
