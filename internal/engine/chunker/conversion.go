package chunker

import (
	"crypto/md5"
	"encoding/hex"
	"fmt"

	"github.com/karim-daw/qwelli/internal/db"
)

// ToDBChunk converts a chunker.Chunk to db.Chunk format
func ToDBChunk(chunk Chunk, file db.File, chunkIndex int, totalChunks int) db.Chunk {
	pageNumbers := []int{}
	if len(chunk.PageNumbers) > 0 {
		pageNumbers = chunk.PageNumbers
	}

	return db.Chunk{
		ChunkID:     GenerateChunkID(file.FileID, chunkIndex),
		FileID:      file.FileID,
		FilePath:    file.Path,     // Denormalized
		FileType:    file.FileType, // Denormalized
		ChunkIndex:  chunkIndex,
		TotalChunks: totalChunks,
		Content:     chunk.Content,
		PageNumbers: pageNumbers,
		ContentType: chunk.ContentType,
		ImageData:   chunk.ImageData,
	}
}

// ToDBChunks converts multiple chunks with proper indexing
func ToDBChunks(chunks []Chunk, file db.File) []db.Chunk {
	dbChunks := make([]db.Chunk, len(chunks))

	for i, chunk := range chunks {
		dbChunks[i] = ToDBChunk(chunk, file, i, len(chunks))
	}

	return dbChunks
}

// GenerateChunkID generates a unique chunk ID for a file and chunk index
func GenerateChunkID(fileID string, chunkIndex int) string {
	source := fmt.Sprintf("%s:chunk:%d", fileID, chunkIndex)
	hash := md5.Sum([]byte(source))
	return hex.EncodeToString(hash[:])
}
