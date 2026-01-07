package chunker

import (
	"testing"

	"github.com/karim-daw/qwelli/internal/db"
)

func TestToDBChunk(t *testing.T) {
	file := db.File{
		FileID:   "test_file_1",
		Path:     "/path/to/test.txt",
		FileType: "txt",
	}

	chunk := Chunk{
		Content:     "Test content",
		ContentType: "text",
		PageNumbers: []int{},
		ChunkIndex:  0,
		TotalChunks: 1,
		Metadata:    map[string]interface{}{"key": "value"},
	}

	dbChunk := ToDBChunk(chunk, file, 0, 1)

	if dbChunk.ChunkID == "" {
		t.Error("Expected ChunkID to be set")
	}

	if dbChunk.FileID != file.FileID {
		t.Errorf("Expected FileID %s, got %s", file.FileID, dbChunk.FileID)
	}

	if dbChunk.FilePath != file.Path {
		t.Errorf("Expected FilePath %s, got %s", file.Path, dbChunk.FilePath)
	}

	if dbChunk.FileType != file.FileType {
		t.Errorf("Expected FileType %s, got %s", file.FileType, dbChunk.FileType)
	}

	if dbChunk.Content != chunk.Content {
		t.Errorf("Expected Content %s, got %s", chunk.Content, dbChunk.Content)
	}

	if dbChunk.ContentType != chunk.ContentType {
		t.Errorf("Expected ContentType %s, got %s", chunk.ContentType, dbChunk.ContentType)
	}
}

func TestToDBChunks(t *testing.T) {
	file := db.File{
		FileID:   "test_file_1",
		Path:     "/path/to/test.txt",
		FileType: "txt",
	}

	chunks := []Chunk{
		{
			Content:     "First chunk",
			ContentType: "text",
			PageNumbers: []int{},
			ChunkIndex:  0,
			TotalChunks: 2,
			Metadata:    make(map[string]interface{}),
		},
		{
			Content:     "Second chunk",
			ContentType: "text",
			PageNumbers: []int{},
			ChunkIndex:  1,
			TotalChunks: 2,
			Metadata:    make(map[string]interface{}),
		},
	}

	dbChunks := ToDBChunks(chunks, file)

	if len(dbChunks) != 2 {
		t.Fatalf("Expected 2 dbChunks, got %d", len(dbChunks))
	}

	for i, dbChunk := range dbChunks {
		if dbChunk.ChunkIndex != i {
			t.Errorf("Chunk %d: expected ChunkIndex %d, got %d", i, i, dbChunk.ChunkIndex)
		}

		if dbChunk.TotalChunks != 2 {
			t.Errorf("Chunk %d: expected TotalChunks 2, got %d", i, dbChunk.TotalChunks)
		}

		if dbChunk.Content != chunks[i].Content {
			t.Errorf("Chunk %d: expected content %s, got %s", i, chunks[i].Content, dbChunk.Content)
		}
	}
}

func TestToDBChunkWithPageNumbers(t *testing.T) {
	file := db.File{
		FileID:   "test_file_1",
		Path:     "/path/to/test.pdf",
		FileType: "pdf",
	}

	chunk := Chunk{
		Content:     "PDF content",
		ContentType: "text",
		PageNumbers: []int{1, 2},
		ChunkIndex:  0,
		TotalChunks: 1,
		Metadata:    make(map[string]interface{}),
	}

	dbChunk := ToDBChunk(chunk, file, 0, 1)

	if len(dbChunk.PageNumbers) != 2 {
		t.Errorf("Expected 2 page numbers, got %d", len(dbChunk.PageNumbers))
	}

	if dbChunk.PageNumbers[0] != 1 || dbChunk.PageNumbers[1] != 2 {
		t.Errorf("Expected page numbers [1, 2], got %v", dbChunk.PageNumbers)
	}
}

func TestGenerateChunkID(t *testing.T) {
	id1 := GenerateChunkID("file1", 0)
	id2 := GenerateChunkID("file1", 1)
	id3 := GenerateChunkID("file2", 0)

	if id1 == id2 {
		t.Error("Different chunk indices should generate different IDs")
	}

	if id1 == id3 {
		t.Error("Different file IDs should generate different IDs")
	}

	if id2 == id3 {
		t.Error("Different files and indices should generate different IDs")
	}

	// Test consistency - same input should produce same output
	id1_again := GenerateChunkID("file1", 0)
	if id1 != id1_again {
		t.Error("Same input should produce same ID consistently")
	}
}
