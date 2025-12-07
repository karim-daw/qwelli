package db

import "time"

type Document struct {
	ID         string
	Path       string
	FileType   string
	ModifiedAt time.Time
	Size       int64
	Metadata   any
	Content    string
}

type Embedding struct {
	DocID   string
	ModelID int
	Vector  []float32
}

type Model struct {
	ID        int
	Name      string
	Dimension int
	CreatedAt time.Time
}
