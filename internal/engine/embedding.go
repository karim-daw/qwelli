package engine

import (
	"fmt"
	"log"

	"github.com/karim-daw/qwelli/internal/db"
	"github.com/karim-daw/qwelli/internal/engine/indexer"
	"github.com/karim-daw/qwelli/internal/engine/processor"
)

// EmbeddingGenerator handles embedding generation for chunks
type EmbeddingGenerator struct {
	embedder         *indexer.Embedder
	imageValidator   *processor.ImageValidator
	enableMultimodal bool
}

// NewEmbeddingGenerator creates a new embedding generator
func NewEmbeddingGenerator(embedder *indexer.Embedder, enableMultimodal bool) *EmbeddingGenerator {
	return &EmbeddingGenerator{
		embedder:         embedder,
		imageValidator:   processor.NewImageValidator(),
		enableMultimodal: enableMultimodal,
	}
}

// GenerateEmbeddings generates embeddings for chunks
// Returns a map of chunk index to embedding vector
func (g *EmbeddingGenerator) GenerateEmbeddings(chunks []db.Chunk) (map[int][]float32, error) {
	if len(chunks) == 0 {
		return nil, fmt.Errorf("no chunks to embed")
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
		return g.generateMultimodalEmbeddings(chunks)
	}

	return g.generateTextEmbeddings(chunks)
}

// generateMultimodalEmbeddings generates embeddings for multimodal chunks (text + images)
func (g *EmbeddingGenerator) generateMultimodalEmbeddings(chunks []db.Chunk) (map[int][]float32, error) {
	multimodalInputs := make([]indexer.MultimodalInput, 0, len(chunks))
	validChunkIndices := make([]int, 0, len(chunks))

	// Reset validator stats
	g.imageValidator.ResetStats()

	for i, chunk := range chunks {
		if chunk.ContentType == "image" {
			// Image chunk - validate and prepare
			imageBase64 := string(chunk.ImageData)
			input, valid := g.imageValidator.ValidateAndPrepare(imageBase64, i)
			if !valid {
				continue
			}

			multimodalInputs = append(multimodalInputs, *input)
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

	// Log validation stats
	if statsMsg := g.imageValidator.FormatStats(); statsMsg != "" {
		log.Print(statsMsg)
	}

	if len(multimodalInputs) == 0 {
		return nil, fmt.Errorf("no valid chunks to embed")
	}

	// Log batch composition only if images are present
	imageCount := 0
	for _, inp := range multimodalInputs {
		if inp.Type == "image" {
			imageCount++
		}
	}
	if imageCount > 0 {
		log.Printf("  Processing %d text and %d image inputs", len(multimodalInputs)-imageCount, imageCount)
	}

	// Generate embeddings
	embeddings, err := g.embedder.EmbedMultimodal(multimodalInputs)
	if err != nil {
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
func (g *EmbeddingGenerator) generateTextEmbeddings(chunks []db.Chunk) (map[int][]float32, error) {
	texts := make([]string, 0, len(chunks))
	validChunkIndices := make([]int, 0, len(chunks))

	for i, chunk := range chunks {
		if chunk.Content == "" {
			continue
		}
		texts = append(texts, chunk.Content)
		validChunkIndices = append(validChunkIndices, i)
	}

	if len(texts) == 0 {
		return nil, fmt.Errorf("no valid chunks to embed")
	}

	// Generate embeddings
	embeddings, err := g.embedder.EmbedBatch(texts)
	if err != nil {
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
