package db

import "time"

type File struct {
	FileID     string
	Path       string
	FileType   string
	FileHash   string // SHA256 of content
	ModifiedAt time.Time
	Size       int64
	IndexedAt  time.Time
}

type Chunk struct {
	ChunkID string
	FileID  string

	// DENORMALIZED: Hot path fields for fast search results
	FilePath string // Duplicated from File.Path
	FileType string // Duplicated from File.FileType

	ChunkIndex  int
	TotalChunks int
	Content     string
	StartToken  *int
	EndToken    *int
	PageNumbers []int
}

type Embedding struct {
	ChunkID string
	Vector  []float32
}

// SearchResult returned from queries (no JOIN needed for basic display)
type SearchResult struct {
	ChunkID     string
	FilePath    string // From chunks.file_path (denormalized)
	FileType    string // From chunks.file_type (denormalized)
	Content     string
	ChunkIndex  int
	TotalChunks int
	PageNumbers []int
	Distance    float64
}
