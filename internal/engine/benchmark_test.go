package engine

import (
	"context"
	"crypto/sha256"
	"encoding/binary"
	"fmt"
	"image"
	"image/color"
	"image/jpeg"
	"image/png"
	"math"
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/karim-daw/qwelli/internal/db"
	"github.com/karim-daw/qwelli/internal/engine/extraction"
	"github.com/karim-daw/qwelli/internal/engine/fileprocessor"
	"github.com/karim-daw/qwelli/internal/voyage"
)

// =============================================================================
// Local Mock Voyage Client (avoids testutil import cycle)
// =============================================================================

// benchMockClient implements voyage.ClientInterface for benchmarks.
type benchMockClient struct {
	dimension int
}

func (m *benchMockClient) EmbeddingModel() string { return "bench-mock" }
func (m *benchMockClient) RerankModel() string     { return "bench-mock-rerank" }

func (m *benchMockClient) Embed(text string) ([]float32, error) {
	return m.deterministicEmbedding(text), nil
}

func (m *benchMockClient) EmbedBatch(ctx context.Context, texts []string, cb func(int, int)) ([][]float32, error) {
	out := make([][]float32, len(texts))
	for i, t := range texts {
		out[i] = m.deterministicEmbedding(t)
		if cb != nil {
			cb(i+1, len(texts))
		}
	}
	return out, nil
}

func (m *benchMockClient) EmbedMultimodal(ctx context.Context, inputs []voyage.MultimodalInput, cb func(int, int)) ([][]float32, error) {
	out := make([][]float32, len(inputs))
	for i, inp := range inputs {
		key := inp.Text
		if inp.Type == "image" {
			key = inp.ImageBase64[:min(64, len(inp.ImageBase64))]
		}
		out[i] = m.deterministicEmbedding(key)
		if cb != nil {
			cb(i+1, len(inputs))
		}
	}
	return out, nil
}

func (m *benchMockClient) Rerank(_ context.Context, _ string, docs []string) ([]voyage.RerankResult, error) {
	results := make([]voyage.RerankResult, len(docs))
	for i, d := range docs {
		results[i] = voyage.RerankResult{Index: i, RelevanceScore: 0.5, Content: d}
	}
	return results, nil
}

func (m *benchMockClient) RerankWithMetadata(_ context.Context, _ string, inputs []voyage.RerankInput) ([]voyage.RerankResult, error) {
	results := make([]voyage.RerankResult, len(inputs))
	for i, inp := range inputs {
		results[i] = voyage.RerankResult{Index: i, RelevanceScore: 0.5, Content: inp.Content}
	}
	return results, nil
}

func (m *benchMockClient) deterministicEmbedding(text string) []float32 {
	h := sha256.Sum256([]byte(text))
	emb := make([]float32, m.dimension)
	for i := range emb {
		off := (i * 4) % len(h)
		var b [4]byte
		for j := 0; j < 4; j++ {
			b[j] = h[(off+j)%len(h)]
		}
		v := binary.BigEndian.Uint32(b[:])
		emb[i] = (float32(v)/float32(math.MaxUint32))*2.0 - 1.0
	}
	var ss float32
	for _, v := range emb {
		ss += v * v
	}
	mag := float32(math.Sqrt(float64(ss)))
	if mag > 0 {
		for i := range emb {
			emb[i] /= mag
		}
	}
	return emb
}

// Compile-time check
var _ voyage.ClientInterface = (*benchMockClient)(nil)

// createBenchEngine creates an Engine with a local mock client.
func createBenchEngine(dimension int) *Engine {
	mock := &benchMockClient{dimension: dimension}
	return NewEngine(mock, true)
}

// =============================================================================
// Test Data Generators
// =============================================================================

// generateTestImage creates a test image file with the given dimensions.
func generateTestImage(t testing.TB, dir, name string, width, height int, format string) string {
	t.Helper()
	path := filepath.Join(dir, name)
	img := image.NewRGBA(image.Rect(0, 0, width, height))
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			// Create a gradient pattern so the image isn't trivially compressible
			img.Set(x, y, color.RGBA{
				R: uint8((x * 255) / width),
				G: uint8((y * 255) / height),
				B: uint8(((x + y) * 128) / (width + height)),
				A: 255,
			})
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
		if err := jpeg.Encode(f, img, &jpeg.Options{Quality: 85}); err != nil {
			t.Fatal(err)
		}
	default:
		t.Fatalf("unsupported format: %s", format)
	}
	return path
}

// generateTestTextFile creates a text file with repeating content to simulate
// a realistic document.
func generateTestTextFile(t testing.TB, dir, name string, numSentences int) string {
	t.Helper()
	path := filepath.Join(dir, name)
	content := ""
	for i := 0; i < numSentences; i++ {
		content += fmt.Sprintf("This is sentence number %d in the document. It contains meaningful content about software engineering and search indexing. ", i+1)
		if (i+1)%10 == 0 {
			content += "\n\n"
		}
	}
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
	return path
}

// BenchmarkScenario defines a test dataset configuration.
type BenchmarkScenario struct {
	Name      string
	NumImages int
	NumTexts  int
	ImgWidth  int
	ImgHeight int
	ImgFormat string
	Sentences int // per text file
}

// setupBenchmarkDataset creates a temporary folder with test files
// matching the scenario specification.
func setupBenchmarkDataset(b *testing.B, scenario BenchmarkScenario) string {
	b.Helper()
	dir := b.TempDir()
	dataDir := filepath.Join(dir, "data")
	if err := os.MkdirAll(dataDir, 0755); err != nil {
		b.Fatal(err)
	}

	for i := 0; i < scenario.NumImages; i++ {
		name := fmt.Sprintf("img_%04d.%s", i, scenario.ImgFormat)
		generateTestImage(b, dataDir, name, scenario.ImgWidth, scenario.ImgHeight, scenario.ImgFormat)
	}

	for i := 0; i < scenario.NumTexts; i++ {
		name := fmt.Sprintf("doc_%04d.txt", i)
		generateTestTextFile(b, dataDir, name, scenario.Sentences)
	}

	return dataDir
}

// =============================================================================
// Micro-Benchmarks: Individual Operations
// =============================================================================

// BenchmarkFileHashing_Sequential benchmarks sequential SHA256 hashing.
func BenchmarkFileHashing_Sequential(b *testing.B) {
	dir := b.TempDir()
	var files []string
	for i := 0; i < 50; i++ {
		files = append(files, generateTestImage(b, dir, fmt.Sprintf("img_%d.png", i), 800, 600, "png"))
	}
	for i := 0; i < 50; i++ {
		files = append(files, generateTestTextFile(b, dir, fmt.Sprintf("doc_%d.txt", i), 200))
	}

	b.ResetTimer()
	for n := 0; n < b.N; n++ {
		for _, f := range files {
			if _, err := extraction.ComputeSHA256(f); err != nil {
				b.Fatal(err)
			}
		}
	}
}

// BenchmarkFileHashing_FromBytes benchmarks SHA256 from pre-read bytes (single-read optimization).
func BenchmarkFileHashing_FromBytes(b *testing.B) {
	dir := b.TempDir()
	var fileData [][]byte
	for i := 0; i < 50; i++ {
		path := generateTestImage(b, dir, fmt.Sprintf("img_%d.png", i), 800, 600, "png")
		data, _ := os.ReadFile(path)
		fileData = append(fileData, data)
	}
	for i := 0; i < 50; i++ {
		path := generateTestTextFile(b, dir, fmt.Sprintf("doc_%d.txt", i), 200)
		data, _ := os.ReadFile(path)
		fileData = append(fileData, data)
	}

	b.ResetTimer()
	for n := 0; n < b.N; n++ {
		for _, data := range fileData {
			_ = extraction.ComputeSHA256FromBytes(data)
		}
	}
}

// BenchmarkFileHashing_Parallel benchmarks parallel SHA256 hashing
// using a worker pool with semaphore pattern.
func BenchmarkFileHashing_Parallel(b *testing.B) {
	dir := b.TempDir()
	var files []string
	for i := 0; i < 50; i++ {
		files = append(files, generateTestImage(b, dir, fmt.Sprintf("img_%d.png", i), 800, 600, "png"))
	}
	for i := 0; i < 50; i++ {
		files = append(files, generateTestTextFile(b, dir, fmt.Sprintf("doc_%d.txt", i), 200))
	}

	numWorkers := runtime.NumCPU()

	b.ResetTimer()
	for n := 0; n < b.N; n++ {
		sem := make(chan struct{}, numWorkers)
		errs := make(chan error, len(files))

		for _, f := range files {
			sem <- struct{}{}
			go func(path string) {
				defer func() { <-sem }()
				if _, err := extraction.ComputeSHA256(path); err != nil {
					errs <- err
				}
			}(f)
		}
		// Drain semaphore
		for i := 0; i < numWorkers; i++ {
			sem <- struct{}{}
		}
		close(errs)
		for err := range errs {
			b.Fatal(err)
		}
	}
}

// BenchmarkImageDecoding_Sequential benchmarks sequential image decoding.
func BenchmarkImageDecoding_Sequential(b *testing.B) {
	dir := b.TempDir()
	var files []string
	for i := 0; i < 30; i++ {
		files = append(files, generateTestImage(b, dir, fmt.Sprintf("img_%d.png", i), 1024, 768, "png"))
	}
	for i := 0; i < 30; i++ {
		files = append(files, generateTestImage(b, dir, fmt.Sprintf("photo_%d.jpg", i), 1920, 1080, "jpeg"))
	}

	service := fileprocessor.NewFileProcessingService(fileprocessor.DefaultProcessingConfig())

	b.ResetTimer()
	for n := 0; n < b.N; n++ {
		for i, f := range files {
			ext := filepath.Ext(f)[1:]
			file := db.File{
				FileID:   fmt.Sprintf("bench-img-%d", i),
				Path:     f,
				FileType: ext,
			}
			if _, _, err := service.ProcessImage(file); err != nil {
				b.Fatal(err)
			}
		}
	}
}

// BenchmarkImageDecoding_Parallel benchmarks parallel image decoding.
func BenchmarkImageDecoding_Parallel(b *testing.B) {
	dir := b.TempDir()
	var files []string
	for i := 0; i < 30; i++ {
		files = append(files, generateTestImage(b, dir, fmt.Sprintf("img_%d.png", i), 1024, 768, "png"))
	}
	for i := 0; i < 30; i++ {
		files = append(files, generateTestImage(b, dir, fmt.Sprintf("photo_%d.jpg", i), 1920, 1080, "jpeg"))
	}

	numWorkers := runtime.NumCPU()

	b.ResetTimer()
	for n := 0; n < b.N; n++ {
		sem := make(chan struct{}, numWorkers)
		errs := make(chan error, len(files))

		for i, f := range files {
			sem <- struct{}{}
			go func(idx int, path string) {
				defer func() { <-sem }()
				service := fileprocessor.NewFileProcessingService(fileprocessor.DefaultProcessingConfig())
				ext := filepath.Ext(path)[1:]
				file := db.File{
					FileID:   fmt.Sprintf("bench-img-%d", idx),
					Path:     path,
					FileType: ext,
				}
				if _, _, err := service.ProcessImage(file); err != nil {
					errs <- err
				}
			}(i, f)
		}
		for i := 0; i < numWorkers; i++ {
			sem <- struct{}{}
		}
		close(errs)
		for err := range errs {
			b.Fatal(err)
		}
	}
}

// BenchmarkTextChunking_Sequential benchmarks sequential text file processing.
func BenchmarkTextChunking_Sequential(b *testing.B) {
	dir := b.TempDir()
	type testFile struct {
		path    string
		content string
	}
	var files []testFile
	for i := 0; i < 100; i++ {
		path := generateTestTextFile(b, dir, fmt.Sprintf("doc_%d.txt", i), 300)
		data, _ := os.ReadFile(path)
		files = append(files, testFile{path: path, content: string(data)})
	}

	service := fileprocessor.NewFileProcessingService(fileprocessor.DefaultProcessingConfig())

	b.ResetTimer()
	for n := 0; n < b.N; n++ {
		for i, f := range files {
			file := db.File{
				FileID:   fmt.Sprintf("bench-txt-%d", i),
				Path:     f.path,
				FileType: "txt",
			}
			if _, _, err := service.ProcessText(file, f.content); err != nil {
				b.Fatal(err)
			}
		}
	}
}

// =============================================================================
// End-to-End Benchmarks: Full Pipeline (Sequential vs Parallel)
// =============================================================================

var scenarios = []BenchmarkScenario{
	{
		Name:      "Small",
		NumImages: 5,
		NumTexts:  10,
		ImgWidth:  800,
		ImgHeight: 600,
		ImgFormat: "jpeg",
		Sentences: 100,
	},
	{
		Name:      "Medium",
		NumImages: 20,
		NumTexts:  30,
		ImgWidth:  1024,
		ImgHeight: 768,
		ImgFormat: "jpeg",
		Sentences: 200,
	},
	{
		Name:      "Large",
		NumImages: 50,
		NumTexts:  50,
		ImgWidth:  1920,
		ImgHeight: 1080,
		ImgFormat: "jpeg",
		Sentences: 300,
	},
}

// BenchmarkProcessFiles_Sequential benchmarks the current sequential processFiles.
func BenchmarkProcessFiles_Sequential(b *testing.B) {
	for _, sc := range scenarios {
		b.Run(sc.Name, func(b *testing.B) {
			dataDir := setupBenchmarkDataset(b, sc)
			eng := createBenchEngine(128)
			files := collectFiles(b, dataDir)

			b.ResetTimer()
			b.ReportMetric(float64(len(files)), "files")

			for n := 0; n < b.N; n++ {
				dbPath := filepath.Join(b.TempDir(), "bench.db")
				projectDB, err := db.OpenProjectDB(dbPath, 128)
				if err != nil {
					b.Fatal(err)
				}

				ctx := context.Background()
				chunks, _, _ := eng.processFiles(ctx, projectDB, files, nil)

				b.ReportMetric(float64(len(chunks)), "chunks")
				projectDB.Close()
			}
		})
	}
}

// BenchmarkProcessFiles_Parallel benchmarks the parallel processFilesParallel.
func BenchmarkProcessFiles_Parallel(b *testing.B) {
	workerCounts := []int{2, 4, 8}
	maxWorkers := runtime.NumCPU()

	for _, sc := range scenarios {
		for _, workers := range workerCounts {
			if workers > maxWorkers {
				continue
			}
			name := fmt.Sprintf("%s/workers=%d", sc.Name, workers)
			b.Run(name, func(b *testing.B) {
				dataDir := setupBenchmarkDataset(b, sc)
				eng := createBenchEngine(128)
				files := collectFiles(b, dataDir)

				b.ResetTimer()
				b.ReportMetric(float64(len(files)), "files")

				for n := 0; n < b.N; n++ {
					dbPath := filepath.Join(b.TempDir(), "bench.db")
					projectDB, err := db.OpenProjectDB(dbPath, 128)
					if err != nil {
						b.Fatal(err)
					}

					ctx := context.Background()
					chunks, _, _ := eng.processFilesParallel(ctx, projectDB, files, workers, nil)

					b.ReportMetric(float64(len(chunks)), "chunks")
					projectDB.Close()
				}
			})
		}
	}
}

// =============================================================================
// Comparative Test: Side-by-Side Timing with Human-Readable Output
// =============================================================================

// TestProcessFiles_SequentialVsParallel runs a direct comparison test
// that prints human-readable timing results.
func TestProcessFiles_SequentialVsParallel(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping performance comparison in short mode")
	}

	sc := BenchmarkScenario{
		Name:      "Comparison",
		NumImages: 20,
		NumTexts:  30,
		ImgWidth:  1024,
		ImgHeight: 768,
		ImgFormat: "jpeg",
		Sentences: 200,
	}

	// Setup
	dir := t.TempDir()
	dataDir := filepath.Join(dir, "data")
	if err := os.MkdirAll(dataDir, 0755); err != nil {
		t.Fatal(err)
	}
	for i := 0; i < sc.NumImages; i++ {
		generateTestImage(t, dataDir, fmt.Sprintf("img_%04d.jpg", i), sc.ImgWidth, sc.ImgHeight, "jpeg")
	}
	for i := 0; i < sc.NumTexts; i++ {
		generateTestTextFile(t, dataDir, fmt.Sprintf("doc_%04d.txt", i), sc.Sentences)
	}

	files := collectFilesT(t, dataDir)
	t.Logf("Dataset: %d files (%d images, %d text)", len(files), sc.NumImages, sc.NumTexts)
	t.Logf("CPUs available: %d", runtime.NumCPU())

	// Sequential run
	eng := createBenchEngine(128)
	seqDB, err := db.OpenProjectDB(filepath.Join(t.TempDir(), "seq.db"), 128)
	if err != nil {
		t.Fatal(err)
	}
	defer seqDB.Close()

	ctx := context.Background()
	seqStart := time.Now()
	seqChunks, seqSkipped, _ := eng.processFiles(ctx, seqDB, files, nil)
	seqDuration := time.Since(seqStart)

	t.Logf("Sequential: %v (chunks=%d, skipped=%d)", seqDuration, len(seqChunks), seqSkipped)

	// Parallel runs with different worker counts
	workerCounts := []int{2, 4, runtime.NumCPU()}
	for _, workers := range workerCounts {
		parEng := createBenchEngine(128)
		parDB, err := db.OpenProjectDB(filepath.Join(t.TempDir(), fmt.Sprintf("par_%d.db", workers)), 128)
		if err != nil {
			t.Fatal(err)
		}

		parStart := time.Now()
		parChunks, parSkipped, _ := parEng.processFilesParallel(ctx, parDB, files, workers, nil)
		parDuration := time.Since(parStart)

		speedup := float64(seqDuration) / float64(parDuration)
		t.Logf("Parallel (workers=%d): %v (chunks=%d, skipped=%d) speedup=%.2fx",
			workers, parDuration, len(parChunks), parSkipped, speedup)

		// Verify correctness: parallel should produce same number of chunks
		if len(parChunks) != len(seqChunks) {
			t.Errorf("Chunk count mismatch: sequential=%d, parallel(w=%d)=%d",
				len(seqChunks), workers, len(parChunks))
		}

		parDB.Close()
	}
}

// =============================================================================
// Helpers
// =============================================================================

// collectFiles scans a directory and returns all supported file paths.
func collectFiles(b *testing.B, dir string) []string {
	b.Helper()
	var files []string
	filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return nil
		}
		ft := fileprocessor.GetFileTypeFromPath(path)
		if fileprocessor.IsSupported(ft) {
			files = append(files, path)
		}
		return nil
	})
	return files
}

// collectFilesT is collectFiles for *testing.T instead of *testing.B.
func collectFilesT(t *testing.T, dir string) []string {
	t.Helper()
	var files []string
	filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return nil
		}
		ft := fileprocessor.GetFileTypeFromPath(path)
		if fileprocessor.IsSupported(ft) {
			files = append(files, path)
		}
		return nil
	})
	return files
}
