package db

import "time"

type Document struct {
	ID            string
	Path          string
	FileType      string
	ModifiedAt    time.Time
	Size          int64
	TextMetadata  any
	Content       string
	ContentType   string // 'text' or 'image'
	ImageMetaData any
}

type Embedding struct {
	DocID  string
	Vector []float32
}
