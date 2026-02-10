package db

import (
	"fmt"
	"math"
	"path/filepath"
	"testing"
)

// makeBenchChunks creates chunks and embeddings for benchmarking.
func makeBenchChunks(n int, dimension int, seed int64) ([]Chunk, map[int][]float32) {
	chunks := make([]Chunk, n)
	embMap := make(map[int][]float32, n)
	for i := 0; i < n; i++ {
		chunks[i] = Chunk{
			ChunkID:     fmt.Sprintf("chunk-bench-%d-%d", seed, i),
			FileID:      "file-001",
			FilePath:    "/test/path/doc.txt",
			FileType:    "txt",
			ChunkIndex:  i,
			TotalChunks: n,
			Content:     fmt.Sprintf("This is chunk content %d with enough text for embedding.", i),
			ContentType: "text",
		}
		// Deterministic fake embedding
		vec := make([]float32, dimension)
		for j := range vec {
			vec[j] = float32(math.Sin(float64(i*dimension+j))) * 0.5
		}
		embMap[i] = vec
	}
	return chunks, embMap
}

// BenchmarkStoreChunks_RowByRow benchmarks the legacy row-by-row InsertChunk + InsertEmbedding.
func BenchmarkStoreChunks_RowByRow(b *testing.B) {
	sizes := []int{100, 500, 1000}
	for _, n := range sizes {
		b.Run(fmt.Sprintf("chunks=%d", n), func(b *testing.B) {
			tmpDir := b.TempDir()
			dbPath := filepath.Join(tmpDir, "bench.db")
			pdb, err := OpenProjectDB(dbPath, 128)
			if err != nil {
				b.Fatal(err)
			}
			defer pdb.Close()

			chunks, embMap := makeBenchChunks(n, 128, 0)

			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				for j, c := range chunks {
					_ = pdb.InsertChunk(c)
					if vec, ok := embMap[j]; ok {
						_ = pdb.InsertEmbedding(Embedding{ChunkID: c.ChunkID, Vector: vec})
					}
				}
				// Reset for next iteration: delete all
				pdb.Conn().Exec("DELETE FROM embeddings")
				pdb.Conn().Exec("DELETE FROM chunks")
			}
		})
	}
}

// BenchmarkStoreChunks_Appender benchmarks the bulk AppendChunksAndEmbeddings.
func BenchmarkStoreChunks_Appender(b *testing.B) {
	sizes := []int{100, 500, 1000}
	for _, n := range sizes {
		b.Run(fmt.Sprintf("chunks=%d", n), func(b *testing.B) {
			tmpDir := b.TempDir()
			dbPath := filepath.Join(tmpDir, "bench.db")
			pdb, err := OpenProjectDB(dbPath, 128)
			if err != nil {
				b.Fatal(err)
			}
			defer pdb.Close()

			chunks, embMap := makeBenchChunks(n, 128, 0)

			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				_ = pdb.AppendChunksAndEmbeddings(chunks, embMap)
				// Reset for next iteration
				pdb.Conn().Exec("DELETE FROM embeddings")
				pdb.Conn().Exec("DELETE FROM chunks")
			}
		})
	}
}
