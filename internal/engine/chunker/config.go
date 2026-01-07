package chunker

// DefaultConfig is the standard chunking configuration used throughout the application
var DefaultConfig = ChunkerConfig{
	ChunkSize:   300,
	OverlapSize: 10,
}
