package embeddings

import (
	"context"
	"fmt"
	"log"

	"github.com/karim-daw/qwelli/internal/db"
	"github.com/karim-daw/qwelli/internal/engine/extraction"
)

// EmbeddingGenerator handles embedding generation for chunks
type EmbeddingGenerator struct {
	embedder         *Embedder
	imageValidator   *extraction.ImageValidator
	enableMultimodal bool
}

// NewEmbeddingGenerator creates a new embedding generator
func NewEmbeddingGenerator(embedder *Embedder, enableMultimodal bool) *EmbeddingGenerator {
	return &EmbeddingGenerator{
		embedder:         embedder,
		imageValidator:   extraction.NewImageValidator(),
		enableMultimodal: enableMultimodal,
	}
}

// GenerateEmbeddings generates embeddings for chunks
// Returns a map of chunk index to embedding vector
// progressCallback is called with (current, total) as batches are processed
func (g *EmbeddingGenerator) GenerateEmbeddings(ctx context.Context, chunks []db.Chunk, progressCallback func(current, total int)) (map[int][]float32, error) {
	if len(chunks) == 0 {
		return nil, fmt.Errorf("no chunks to embed")
	}

	// Check for cancellation
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}

	// Check if we have multimodal chunks (images)
	hasImages := false
	for _, chunk := range chunks {
		if chunk.ContentType == "image" {
			hasImages = true
			break
		}
	}

	if hasImages && g.enableMultimodal && g.embedder.IsMultimodal() {
		return g.generateMultimodalEmbeddings(ctx, chunks, progressCallback)
	}

	return g.generateTextEmbeddings(ctx, chunks, progressCallback)
}

// generateMultimodalEmbeddings generates embeddings for multimodal chunks (text + images)
func (g *EmbeddingGenerator) generateMultimodalEmbeddings(ctx context.Context, chunks []db.Chunk, progressCallback func(current, total int)) (map[int][]float32, error) {
	multimodalInputs := make([]MultimodalInput, 0, len(chunks))
	validChunkIndices := make([]int, 0, len(chunks))

	// Reset validator stats
	g.imageValidator.ResetStats()

	for i, chunk := range chunks {
		if chunk.ContentType == "image" {
			// Image chunk - validate and prepare
			imageBase64 := string(chunk.ImageData)
			imageData, valid := g.imageValidator.ValidateAndPrepare(imageBase64, i)
			if !valid {
				continue
			}

			multimodalInputs = append(multimodalInputs, MultimodalInput{
				Type:        "image",
				ImageBase64: imageData.Base64,
				ImageFormat: imageData.Format,
				ImagePixels: imageData.Pixels,
			})
			validChunkIndices = append(validChunkIndices, i)
		} else {
			// Text chunk - skip if empty
			if chunk.Content == "" {
				log.Printf("⚠️  Skipping text chunk %d: empty content", i)
				continue
			}

			multimodalInputs = append(multimodalInputs, MultimodalInput{
				Type: "text",
				Text: chunk.Content,
			})
			validChunkIndices = append(validChunkIndices, i)
		}
	}

	// Log validation stats
	if statsMsg := g.imageValidator.FormatStats(); statsMsg != "" {
		log.Print(statsMsg)
	}

	if len(multimodalInputs) == 0 {
		return nil, fmt.Errorf("no valid chunks to embed")
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

	// Generate embeddings with progress tracking
	embeddings, err := g.embedder.EmbedMultimodal(ctx, multimodalInputs, progressCallback)
	if err != nil {
		// Check if error is due to cancellation
		if ctx.Err() != nil {
			return nil, ctx.Err()
		}
		return nil, fmt.Errorf("failed to generate embeddings: %w", err)
	}

	// Map embeddings back to valid chunks
	if len(embeddings) != len(validChunkIndices) {
		return nil, fmt.Errorf("embedding count mismatch: got %d embeddings for %d valid chunks", len(embeddings), len(validChunkIndices))
	}

	// Create a map of chunk index to embedding
	embeddingMap := make(map[int][]float32)
	for j, idx := range validChunkIndices {
		if j < len(embeddings) {
			embeddingMap[idx] = embeddings[j]
		}
	}

	return embeddingMap, nil
}

// generateTextEmbeddings generates embeddings for text-only chunks
func (g *EmbeddingGenerator) generateTextEmbeddings(ctx context.Context, chunks []db.Chunk, progressCallback func(current, total int)) (map[int][]float32, error) {
	texts := make([]string, 0, len(chunks))
	validChunkIndices := make([]int, 0, len(chunks))

	for i, chunk := range chunks {
		if chunk.Content == "" {
			log.Printf("⚠️  Skipping text chunk %d: empty content", i)
			continue
		}
		texts = append(texts, chunk.Content)
		validChunkIndices = append(validChunkIndices, i)
	}

	if len(texts) == 0 {
		return nil, fmt.Errorf("no valid chunks to embed")
	}

	// Generate embeddings with progress tracking
	embeddings, err := g.embedder.EmbedBatch(ctx, texts, progressCallback)
	if err != nil {
		// Check if error is due to cancellation
		if ctx.Err() != nil {
			return nil, ctx.Err()
		}
		return nil, fmt.Errorf("failed to generate embeddings: %w", err)
	}

	// Map embeddings back to valid chunks
	if len(embeddings) != len(validChunkIndices) {
		return nil, fmt.Errorf("embedding count mismatch: got %d embeddings for %d valid chunks", len(embeddings), len(validChunkIndices))
	}

	// Create a map of chunk index to embedding
	embeddingMap := make(map[int][]float32)
	for j, idx := range validChunkIndices {
		if j < len(embeddings) {
			embeddingMap[idx] = embeddings[j]
		}
	}

	return embeddingMap, nil
}

// StoreChunksAndEmbeddings stores chunks and their embeddings in the database
func StoreChunksAndEmbeddings(projectDB *db.ProjectDB, chunks []db.Chunk, embeddingMap map[int][]float32) error {
	for i, chunk := range chunks {
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
	return nil
}
