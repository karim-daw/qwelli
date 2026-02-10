package fileprocessor

import (
	"encoding/base64"
	"image"
	"image/color"
	"image/gif"
	"image/jpeg"
	"image/png"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/karim-daw/qwelli/internal/db"
)

// createTestImage creates a test image file of the given format and dimensions.
func createTestImage(t *testing.T, dir, name string, width, height int, format string) string {
	t.Helper()
	path := filepath.Join(dir, name)
	img := image.NewRGBA(image.Rect(0, 0, width, height))
	// Fill with a solid color so the image is valid
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			img.Set(x, y, color.RGBA{R: 100, G: 150, B: 200, A: 255})
		}
	}

	f, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()

	switch format {
	case "png":
		if err := png.Encode(f, img); err != nil {
			t.Fatal(err)
		}
	case "jpeg", "jpg":
		if err := jpeg.Encode(f, img, &jpeg.Options{Quality: 80}); err != nil {
			t.Fatal(err)
		}
	case "gif":
		if err := gif.Encode(f, img, nil); err != nil {
			t.Fatal(err)
		}
	default:
		t.Fatalf("unsupported test image format: %s", format)
	}
	return path
}

func TestProcessImage_PNG(t *testing.T) {
	tmpDir := t.TempDir()
	imgPath := createTestImage(t, tmpDir, "test.png", 800, 600, "png")

	file := db.File{
		FileID:   "test-png-id",
		Path:     imgPath,
		FileType: "png",
	}

	service := NewFileProcessingService(DefaultProcessingConfig())
	chunks, contents, err := service.ProcessImage(file)
	if err != nil {
		t.Fatalf("ProcessImage failed: %v", err)
	}

	if len(chunks) != 1 {
		t.Fatalf("Expected 1 chunk, got %d", len(chunks))
	}
	if len(contents) != 1 {
		t.Fatalf("Expected 1 content string, got %d", len(contents))
	}

	chunk := chunks[0]
	if chunk.ContentType != "image" {
		t.Errorf("Expected ContentType 'image', got %q", chunk.ContentType)
	}
	if chunk.ChunkIndex != 0 {
		t.Errorf("Expected ChunkIndex 0, got %d", chunk.ChunkIndex)
	}
	if chunk.TotalChunks != 1 {
		t.Errorf("Expected TotalChunks 1, got %d", chunk.TotalChunks)
	}
	if len(chunk.PageNumbers) != 0 {
		t.Errorf("Expected no page numbers for standalone image, got %v", chunk.PageNumbers)
	}
	if !strings.Contains(chunk.Content, "800x600") {
		t.Errorf("Expected content to contain dimensions, got %q", chunk.Content)
	}

	// Verify base64 is valid
	if _, err := base64.StdEncoding.DecodeString(string(chunk.ImageData)); err != nil {
		t.Errorf("ImageData is not valid base64: %v", err)
	}

	// Verify content string is base64
	if _, err := base64.StdEncoding.DecodeString(contents[0]); err != nil {
		t.Errorf("Content string is not valid base64: %v", err)
	}
}

func TestProcessImage_JPEG(t *testing.T) {
	tmpDir := t.TempDir()
	imgPath := createTestImage(t, tmpDir, "test.jpg", 1024, 768, "jpeg")

	file := db.File{
		FileID:   "test-jpeg-id",
		Path:     imgPath,
		FileType: "jpg",
	}

	service := NewFileProcessingService(DefaultProcessingConfig())
	chunks, contents, err := service.ProcessImage(file)
	if err != nil {
		t.Fatalf("ProcessImage failed: %v", err)
	}

	if len(chunks) != 1 {
		t.Fatalf("Expected 1 chunk, got %d", len(chunks))
	}
	if len(contents) != 1 {
		t.Fatalf("Expected 1 content string, got %d", len(contents))
	}

	chunk := chunks[0]
	if chunk.ContentType != "image" {
		t.Errorf("Expected ContentType 'image', got %q", chunk.ContentType)
	}
	if !strings.Contains(chunk.Content, "1024x768") {
		t.Errorf("Expected content to contain dimensions, got %q", chunk.Content)
	}
}

func TestProcessImage_GIF(t *testing.T) {
	tmpDir := t.TempDir()
	imgPath := createTestImage(t, tmpDir, "test.gif", 640, 480, "gif")

	file := db.File{
		FileID:   "test-gif-id",
		Path:     imgPath,
		FileType: "gif",
	}

	service := NewFileProcessingService(DefaultProcessingConfig())
	chunks, contents, err := service.ProcessImage(file)
	if err != nil {
		t.Fatalf("ProcessImage failed: %v", err)
	}

	if len(chunks) != 1 {
		t.Fatalf("Expected 1 chunk, got %d", len(chunks))
	}
	if len(contents) != 1 {
		t.Fatalf("Expected 1 content string, got %d", len(contents))
	}

	chunk := chunks[0]
	if chunk.ContentType != "image" {
		t.Errorf("Expected ContentType 'image', got %q", chunk.ContentType)
	}
	if !strings.Contains(chunk.Content, "640x480") {
		t.Errorf("Expected content to contain dimensions, got %q", chunk.Content)
	}
}

func TestProcessImage_TextOnlyMode(t *testing.T) {
	tmpDir := t.TempDir()
	imgPath := createTestImage(t, tmpDir, "test.png", 100, 100, "png")

	file := db.File{
		FileID:   "test-skip-id",
		Path:     imgPath,
		FileType: "png",
	}

	service := NewFileProcessingService(DefaultProcessingConfig())
	service.SetContentTypeMode(ContentTypeText)

	_, _, err := service.ProcessImage(file)
	if err == nil {
		t.Error("Expected error in text-only mode, got nil")
	}
	if err != nil && !strings.Contains(err.Error(), "text-only mode") {
		t.Errorf("Expected text-only mode error, got: %v", err)
	}
}

func TestProcessImage_InvalidImage(t *testing.T) {
	tmpDir := t.TempDir()
	invalidPath := filepath.Join(tmpDir, "invalid.png")
	if err := os.WriteFile(invalidPath, []byte("not an image"), 0644); err != nil {
		t.Fatal(err)
	}

	file := db.File{
		FileID:   "test-invalid-id",
		Path:     invalidPath,
		FileType: "png",
	}

	service := NewFileProcessingService(DefaultProcessingConfig())
	_, _, err := service.ProcessImage(file)
	if err == nil {
		t.Error("Expected error for invalid image, got nil")
	}
}

func TestProcessImage_NonexistentFile(t *testing.T) {
	file := db.File{
		FileID:   "test-missing-id",
		Path:     "/nonexistent/path/image.png",
		FileType: "png",
	}

	service := NewFileProcessingService(DefaultProcessingConfig())
	_, _, err := service.ProcessImage(file)
	if err == nil {
		t.Error("Expected error for nonexistent file, got nil")
	}
}

func TestProcessImage_FileMetadata(t *testing.T) {
	tmpDir := t.TempDir()
	imgPath := createTestImage(t, tmpDir, "photo.jpg", 1920, 1080, "jpeg")

	file := db.File{
		FileID:   "test-meta-id",
		Path:     imgPath,
		FileType: "jpg",
	}

	service := NewFileProcessingService(DefaultProcessingConfig())
	chunks, _, err := service.ProcessImage(file)
	if err != nil {
		t.Fatalf("ProcessImage failed: %v", err)
	}

	chunk := chunks[0]
	// Verify file metadata is set correctly on db.Chunk
	if chunk.FileID != "test-meta-id" {
		t.Errorf("Expected FileID 'test-meta-id', got %q", chunk.FileID)
	}
	if chunk.FilePath != imgPath {
		t.Errorf("Expected FilePath %q, got %q", imgPath, chunk.FilePath)
	}
	if chunk.FileType != "jpg" {
		t.Errorf("Expected FileType 'jpg', got %q", chunk.FileType)
	}
}

func TestProcessImage_ViaProcessRouter(t *testing.T) {
	tmpDir := t.TempDir()
	imgPath := createTestImage(t, tmpDir, "routed.png", 500, 500, "png")

	file := db.File{
		FileID:   "test-route-id",
		Path:     imgPath,
		FileType: "png",
	}

	service := NewFileProcessingService(DefaultProcessingConfig())
	chunks, contents, err := service.Process(file, DefaultProcessOptions())
	if err != nil {
		t.Fatalf("Process (image routing) failed: %v", err)
	}

	if len(chunks) != 1 {
		t.Fatalf("Expected 1 chunk via Process routing, got %d", len(chunks))
	}
	if len(contents) != 1 {
		t.Fatalf("Expected 1 content via Process routing, got %d", len(contents))
	}
	if chunks[0].ContentType != "image" {
		t.Errorf("Expected ContentType 'image', got %q", chunks[0].ContentType)
	}
}

func TestCanProcess_IncludesImages(t *testing.T) {
	service := NewFileProcessingService(DefaultProcessingConfig())

	imageTypes := []string{"jpg", "jpeg", "png", "gif"}
	for _, ext := range imageTypes {
		if !service.CanProcess(ext) {
			t.Errorf("CanProcess(%q) = false, want true", ext)
		}
	}
}
