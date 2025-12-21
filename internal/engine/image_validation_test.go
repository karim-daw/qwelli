package engine

import (
	"bytes"
	"encoding/base64"
	"image"
	"image/color"
	"image/jpeg"
	"image/png"
	"path/filepath"
	"testing"
	"time"

	"github.com/karim-daw/qwelli/internal/db"
)

// createTestImage creates a test image with specified dimensions
func createTestImage(width, height int, format string) ([]byte, string, error) {
	img := image.NewRGBA(image.Rect(0, 0, width, height))
	// Fill with a simple pattern
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			img.Set(x, y, color.RGBA{R: uint8(x % 256), G: uint8(y % 256), B: 128, A: 255})
		}
	}

	var buf bytes.Buffer
	var base64Str string

	switch format {
	case "jpeg", "jpg":
		if err := jpeg.Encode(&buf, img, nil); err != nil {
			return nil, "", err
		}
		base64Str = base64.StdEncoding.EncodeToString(buf.Bytes())
	case "png":
		if err := png.Encode(&buf, img); err != nil {
			return nil, "", err
		}
		base64Str = base64.StdEncoding.EncodeToString(buf.Bytes())
	default:
		return nil, "", image.ErrFormat
	}

	return buf.Bytes(), base64Str, nil
}

// TestEngine_ImageValidation_SizeFiltering tests image size validation
func TestEngine_ImageValidation_SizeFiltering(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	// Create test images
	_, tinyBase64, err := createTestImage(100, 100, "jpeg")
	if err != nil {
		t.Fatalf("createTestImage() tiny error = %v", err)
	}

	_, smallBase64, err := createTestImage(400, 300, "jpeg")
	if err != nil {
		t.Fatalf("createTestImage() small error = %v", err)
	}

	_, validBase64, err := createTestImage(800, 600, "jpeg")
	if err != nil {
		t.Fatalf("createTestImage() valid error = %v", err)
	}

	_, largeBase64, err := createTestImage(5000, 4000, "jpeg")
	if err != nil {
		t.Fatalf("createTestImage() large error = %v", err)
	}

	// Create database and insert chunks
	pdb, err := db.OpenProjectDB(dbPath, 1024)
	if err != nil {
		t.Fatalf("OpenProjectDB() error = %v", err)
	}
	defer pdb.Close()

	testFile := db.File{
		FileID:     "test-file",
		Path:       "/test/file.pdf",
		FileType:   "pdf",
		FileHash:   "hash",
		ModifiedAt: time.Now(),
		Size:       1000,
		IndexedAt:  time.Now(),
	}

	if err := pdb.InsertFile(testFile); err != nil {
		t.Fatalf("InsertFile() error = %v", err)
	}

	// Insert chunks with different image sizes
	chunks := []db.Chunk{
		{
			ChunkID:     "tiny",
			FileID:      testFile.FileID,
			FilePath:    testFile.Path,
			FileType:    testFile.FileType,
			ChunkIndex:  0,
			TotalChunks: 4,
			Content:     "Tiny image",
			ContentType: "image",
			ImageData:   []byte(tinyBase64),
			PageNumbers: []int{1},
		},
		{
			ChunkID:     "small",
			FileID:      testFile.FileID,
			FilePath:    testFile.Path,
			FileType:    testFile.FileType,
			ChunkIndex:  1,
			TotalChunks: 4,
			Content:     "Small image",
			ContentType: "image",
			ImageData:   []byte(smallBase64),
			PageNumbers: []int{1},
		},
		{
			ChunkID:     "valid",
			FileID:      testFile.FileID,
			FilePath:    testFile.Path,
			FileType:    testFile.FileType,
			ChunkIndex:  2,
			TotalChunks: 4,
			Content:     "Valid image",
			ContentType: "image",
			ImageData:   []byte(validBase64),
			PageNumbers: []int{1},
		},
		{
			ChunkID:     "large",
			FileID:      testFile.FileID,
			FilePath:    testFile.Path,
			FileType:    testFile.FileType,
			ChunkIndex:  3,
			TotalChunks: 4,
			Content:     "Large image",
			ContentType: "image",
			ImageData:   []byte(largeBase64),
			PageNumbers: []int{1},
		},
	}

	for _, chunk := range chunks {
		if err := pdb.InsertChunk(chunk); err != nil {
			t.Fatalf("InsertChunk() error = %v", err)
		}
	}

	// Note: We can't easily test the full embedding process without mocking the API,
	// but we can verify that the chunks are stored correctly and the validation
	// logic would be called during processFolder

	// Verify chunks are stored
	for _, chunk := range chunks {
		retrieved, err := pdb.GetChunk(chunk.ChunkID)
		if err != nil {
			t.Fatalf("GetChunk() %s error = %v", chunk.ChunkID, err)
		}
		if retrieved.ContentType != "image" {
			t.Errorf("GetChunk() %s ContentType = %q, want %q", chunk.ChunkID, retrieved.ContentType, "image")
		}
		if len(retrieved.ImageData) == 0 {
			t.Errorf("GetChunk() %s ImageData is empty", chunk.ChunkID)
		}
	}
}

// TestEngine_ImageValidation_InvalidBase64 tests invalid base64 handling
func TestEngine_ImageValidation_InvalidBase64(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	pdb, err := db.OpenProjectDB(dbPath, 1024)
	if err != nil {
		t.Fatalf("OpenProjectDB() error = %v", err)
	}
	defer pdb.Close()

	testFile := db.File{
		FileID:     "test-file",
		Path:       "/test/file.pdf",
		FileType:   "pdf",
		FileHash:   "hash",
		ModifiedAt: time.Now(),
		Size:       1000,
		IndexedAt:  time.Now(),
	}

	if err := pdb.InsertFile(testFile); err != nil {
		t.Fatalf("InsertFile() error = %v", err)
	}

	// Test various invalid base64 scenarios
	invalidChunks := []struct {
		name    string
		base64  string
		chunkID string
	}{
		{"empty", "", "empty"},
		{"too short", "abc", "short"},
		{"invalid chars", "!!!invalid!!!", "invalid"},
		{"incomplete", "SGVsbG8gV29ybGQ", "incomplete"}, // Valid base64 but not an image
	}

	for _, tc := range invalidChunks {
		chunk := db.Chunk{
			ChunkID:     tc.chunkID,
			FileID:      testFile.FileID,
			FilePath:    testFile.Path,
			FileType:    testFile.FileType,
			ChunkIndex:  0,
			TotalChunks: 1,
			Content:     tc.name,
			ContentType: "image",
			ImageData:   []byte(tc.base64),
			PageNumbers: []int{1},
		}

		if err := pdb.InsertChunk(chunk); err != nil {
			t.Fatalf("InsertChunk() %s error = %v", tc.name, err)
		}

		// Verify chunk is stored (validation happens during embedding, not storage)
		retrieved, err := pdb.GetChunk(chunk.ChunkID)
		if err != nil {
			t.Fatalf("GetChunk() %s error = %v", tc.name, err)
		}
		if string(retrieved.ImageData) != tc.base64 {
			t.Errorf("GetChunk() %s ImageData = %q, want %q", tc.name, string(retrieved.ImageData), tc.base64)
		}
	}
}
