package engine

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/karim-daw/qwelli/internal/db"
	"github.com/karim-daw/qwelli/internal/engine/fileprocessor"
)

// TestRealData_SequentialVsParallel benchmarks sequential vs parallel processing
// on the real testdata folder containing actual PDFs and images.
func TestRealData_SequentialVsParallel(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping real data performance test in short mode")
	}

	// Use the real testdata folder
	testdataPath := filepath.Join("..", "..", "testdata")
	absPath, err := filepath.Abs(testdataPath)
	if err != nil {
		t.Fatal(err)
	}

	// Check if testdata exists
	if _, err := os.Stat(absPath); os.IsNotExist(err) {
		t.Skip("testdata folder not found")
	}

	// Collect all supported files recursively
	files := collectRealDataFiles(t, absPath)
	if len(files) == 0 {
		t.Skip("no supported files found in testdata")
	}

	// Count file types
	var pdfCount, imageCount, textCount int
	for _, f := range files {
		ft := fileprocessor.GetFileTypeFromPath(f)
		switch {
		case fileprocessor.IsPDFFile(ft):
			pdfCount++
		case fileprocessor.IsImageFile(ft):
			imageCount++
		case fileprocessor.IsTextFile(ft):
			textCount++
		}
	}

	t.Logf("Real dataset from testdata/:")
	t.Logf("  Total files: %d", len(files))
	t.Logf("  PDFs: %d", pdfCount)
	t.Logf("  Images: %d", imageCount)
	t.Logf("  Text files: %d", textCount)
	t.Logf("  CPUs available: %d", runtime.NumCPU())
	t.Log("")

	// Sequential run
	eng := createBenchEngine(128)
	seqDB, err := db.OpenProjectDB(filepath.Join(t.TempDir(), "seq_real.db"), 128)
	if err != nil {
		t.Fatal(err)
	}
	defer seqDB.Close()

	ctx := context.Background()
	t.Log("Running sequential processing...")
	seqStart := time.Now()
	seqChunks, seqSkipped, _ := eng.processFilesParallel(ctx, seqDB, files, 1, nil)
	seqDuration := time.Since(seqStart)

	t.Logf("✓ Sequential: %v", seqDuration)
	t.Logf("    Chunks created: %d", len(seqChunks))
	t.Logf("    Files skipped: %d", seqSkipped)
	t.Log("")

	// Parallel runs with different worker counts
	workerCounts := []int{2, 4, runtime.NumCPU()}
	for _, workers := range workerCounts {
		parEng := createBenchEngine(128)
		parDB, err := db.OpenProjectDB(filepath.Join(t.TempDir(), fmt.Sprintf("par_real_%d.db", workers)), 128)
		if err != nil {
			t.Fatal(err)
		}

		t.Logf("Running parallel processing with %d workers...", workers)
		parStart := time.Now()
		parChunks, parSkipped, _ := parEng.processFilesParallel(ctx, parDB, files, workers, nil)
		parDuration := time.Since(parStart)

		speedup := float64(seqDuration) / float64(parDuration)
		t.Logf("✓ Parallel (%d workers): %v", workers, parDuration)
		t.Logf("    Chunks created: %d", len(parChunks))
		t.Logf("    Files skipped: %d", parSkipped)
		t.Logf("    Speedup: %.2fx", speedup)

		// Verify correctness
		if len(parChunks) != len(seqChunks) {
			t.Errorf("❌ Chunk count mismatch: sequential=%d, parallel(w=%d)=%d",
				len(seqChunks), workers, len(parChunks))
		} else {
			t.Logf("    ✓ Correctness verified")
		}
		t.Log("")

		parDB.Close()
	}

	// Print summary
	t.Log("========================================")
	t.Log("SUMMARY")
	t.Log("========================================")
	t.Logf("Sequential baseline: %v", seqDuration)
	for _, workers := range workerCounts {
		parEng := createBenchEngine(128)
		parDB, err := db.OpenProjectDB(filepath.Join(t.TempDir(), fmt.Sprintf("par_summary_%d.db", workers)), 128)
		if err != nil {
			continue
		}
		parStart := time.Now()
		parEng.processFilesParallel(ctx, parDB, files, workers, nil)
		parDuration := time.Since(parStart)
		speedup := float64(seqDuration) / float64(parDuration)
		t.Logf("Parallel (%2d workers): %v (%.2fx faster)", workers, parDuration, speedup)
		parDB.Close()
	}
}

// BenchmarkRealData_Sequential benchmarks sequential processing on real testdata.
func BenchmarkRealData_Sequential(b *testing.B) {
	testdataPath := filepath.Join("..", "..", "testdata")
	absPath, err := filepath.Abs(testdataPath)
	if err != nil {
		b.Fatal(err)
	}

	if _, err := os.Stat(absPath); os.IsNotExist(err) {
		b.Skip("testdata folder not found")
	}

	files := collectRealDataFiles(b, absPath)
	if len(files) == 0 {
		b.Skip("no supported files found in testdata")
	}

	eng := createBenchEngine(128)

	b.ResetTimer()
	b.ReportMetric(float64(len(files)), "files")

	for n := 0; n < b.N; n++ {
		dbPath := filepath.Join(b.TempDir(), "bench_real.db")
		projectDB, err := db.OpenProjectDB(dbPath, 128)
		if err != nil {
			b.Fatal(err)
		}

		ctx := context.Background()
		chunks, _, _ := eng.processFilesParallel(ctx, projectDB, files, 1, nil)

		b.ReportMetric(float64(len(chunks)), "chunks")
		projectDB.Close()
	}
}

// BenchmarkRealData_Parallel benchmarks parallel processing on real testdata.
func BenchmarkRealData_Parallel(b *testing.B) {
	testdataPath := filepath.Join("..", "..", "testdata")
	absPath, err := filepath.Abs(testdataPath)
	if err != nil {
		b.Fatal(err)
	}

	if _, err := os.Stat(absPath); os.IsNotExist(err) {
		b.Skip("testdata folder not found")
	}

	files := collectRealDataFiles(b, absPath)
	if len(files) == 0 {
		b.Skip("no supported files found in testdata")
	}

	workerCounts := []int{2, 4, runtime.NumCPU()}
	for _, workers := range workerCounts {
		b.Run(fmt.Sprintf("workers=%d", workers), func(b *testing.B) {
			eng := createBenchEngine(128)

			b.ResetTimer()
			b.ReportMetric(float64(len(files)), "files")

			for n := 0; n < b.N; n++ {
				dbPath := filepath.Join(b.TempDir(), "bench_real.db")
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

// collectRealDataFiles recursively scans a directory for supported files.
func collectRealDataFiles(tb testing.TB, dir string) []string {
	tb.Helper()
	var files []string
	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil // Skip errors
		}
		if info.IsDir() {
			return nil
		}
		ft := fileprocessor.GetFileTypeFromPath(path)
		if fileprocessor.IsSupported(ft) {
			files = append(files, path)
		}
		return nil
	})
	if err != nil {
		tb.Fatal(err)
	}
	return files
}
