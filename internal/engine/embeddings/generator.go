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

func NewEmbeddingGenerator(embedder *Embedder, enableMultimodal bool) *EmbeddingGenerator {
	return &EmbeddingGenerator{
		embedder:         embedder,
		imageValidator:   extraction.NewImageValidator(),
		enableMultimodal: enableMultimodal,
	}
}

// GenerateEmbeddings generates embeddings for chunks, returning chunkIndex -> vector
func (g *EmbeddingGenerator) GenerateEmbeddings(ctx context.Context, chunks []db.Chunk, progressCb func(int, int)) (map[int][]float32, error) {
	if len(chunks) == 0 {
		return nil, fmt.Errorf("no chunks to embed")
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	// Check if we need multimodal mode
	hasImages := false
	for _, c := range chunks {
		if c.ContentType == "image" {
			hasImages = true
			break
		}
	}
	useMultimodal := hasImages && g.enableMultimodal && g.embedder.IsMultimodal()

	// Build inputs and track valid indices
	var (
		validIndices []int
		textInputs   []string
		mmInputs     []MultimodalInput
	)

	if useMultimodal {
		g.imageValidator.ResetStats()
	}

	for i, chunk := range chunks {
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
		return nil, fmt.Errorf("no valid chunks to embed")
	}

	// Generate embeddings
	var (
		embeddings [][]float32
		err        error
	)
	if useMultimodal {
		embeddings, err = g.embedder.EmbedMultimodal(ctx, mmInputs, progressCb)
	} else {
		embeddings, err = g.embedder.EmbedBatch(ctx, textInputs, progressCb)
	}
	if err != nil {
		if ctx.Err() != nil {
			return nil, ctx.Err()
		}
		return nil, fmt.Errorf("generate embeddings: %w", err)
	}

	if len(embeddings) != len(validIndices) {
		return nil, fmt.Errorf("embedding count mismatch: got %d for %d chunks", len(embeddings), len(validIndices))
	}

	// Map back to chunk indices
	result := make(map[int][]float32, len(validIndices))
	for j, idx := range validIndices {
		result[idx] = embeddings[j]
	}
	return result, nil
}

// StoreChunksAndEmbeddings stores chunks and their embeddings in the database
func StoreChunksAndEmbeddings(projectDB *db.ProjectDB, chunks []db.Chunk, embeddingMap map[int][]float32) error {
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
	return nil
}
