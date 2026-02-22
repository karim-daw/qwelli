package engine

import (
	"context"
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/karim-daw/qwelli/internal/db"
	"github.com/karim-daw/qwelli/internal/engine/fileprocessor"
)

// TestRealData_PDFParallelPages tests parallel page processing on real PDFs.
func TestRealData_PDFParallelPages(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping PDF parallel page test in short mode")
	}

	testdataPath := filepath.Join("..", "..", "testdata")
	absPath, err := filepath.Abs(testdataPath)
	if err != nil {
		t.Fatal(err)
	}

	if _, err := os.Stat(absPath); os.IsNotExist(err) {
		t.Skip("testdata folder not found")
	}

	files := collectRealDataFiles(t, absPath)
	if len(files) == 0 {
		t.Skip("no supported files found in testdata")
	}

	// Count PDFs
	var pdfFiles []string
	for _, f := range files {
		ft := fileprocessor.GetFileTypeFromPath(f)
		if fileprocessor.IsPDFFile(ft) {
			pdfFiles = append(pdfFiles, f)
		}
	}

	if len(pdfFiles) == 0 {
		t.Skip("no PDFs found in testdata")
	}

	t.Logf("Testing PDF parallel page processing:")
	t.Logf("  Total PDFs: %d", len(pdfFiles))
	t.Logf("  CPUs available: %d", runtime.NumCPU())
	t.Log("")

	ctx := context.Background()

	// Test 1: Sequential file processing, sequential PDF pages
	t.Log("=== Configuration 1: Sequential files + Sequential PDF pages ===")
	eng1 := createBenchEngine(128)
	eng1.SetParallelProcessing(false, 0, 0)
	eng1.SetParallelPDFProcessing(false, 0)

	db1, _ := db.OpenProjectDB(filepath.Join(t.TempDir(), "seq_seq.db"), 128)
	defer db1.Close()

	start1 := time.Now()
	chunks1, _, _ := eng1.processFilesParallel(ctx, db1, files, 1, nil)
	dur1 := time.Since(start1)
	t.Logf("✓ Sequential files + Sequential pages: %v (%d chunks)", dur1, len(chunks1))
	t.Log("")

	// Test 2: Parallel file processing, sequential PDF pages
	t.Log("=== Configuration 2: Parallel files + Sequential PDF pages ===")
	eng2 := createBenchEngine(128)
	eng2.SetParallelProcessing(true, 4, 0)
	eng2.SetParallelPDFProcessing(false, 0)

	db2, _ := db.OpenProjectDB(filepath.Join(t.TempDir(), "par_seq.db"), 128)
	defer db2.Close()

	start2 := time.Now()
	chunks2, _, _ := eng2.processFilesParallel(ctx, db2, files, 4, nil)
	dur2 := time.Since(start2)
	t.Logf("✓ Parallel files + Sequential pages: %v (%d chunks)", dur2, len(chunks2))
	t.Logf("  Speedup vs baseline: %.2fx", float64(dur1)/float64(dur2))
	t.Log("")

	// Test 3: Sequential file processing, parallel PDF pages
	t.Log("=== Configuration 3: Sequential files + Parallel PDF pages ===")
	eng3 := createBenchEngine(128)
	eng3.SetParallelProcessing(false, 0, 0)
	eng3.SetParallelPDFProcessing(true, 4)

	db3, _ := db.OpenProjectDB(filepath.Join(t.TempDir(), "seq_par.db"), 128)
	defer db3.Close()

	start3 := time.Now()
	chunks3, _, _ := eng3.processFilesParallel(ctx, db3, files, 1, nil)
	dur3 := time.Since(start3)
	t.Logf("✓ Sequential files + Parallel pages: %v (%d chunks)", dur3, len(chunks3))
	t.Logf("  Speedup vs baseline: %.2fx", float64(dur1)/float64(dur3))
	t.Log("")

	// Test 4: Parallel file processing, parallel PDF pages
	t.Log("=== Configuration 4: Parallel files + Parallel PDF pages ===")
	eng4 := createBenchEngine(128)
	eng4.SetParallelProcessing(true, 4, 0)
	eng4.SetParallelPDFProcessing(true, 4)

	db4, _ := db.OpenProjectDB(filepath.Join(t.TempDir(), "par_par.db"), 128)
	defer db4.Close()

	start4 := time.Now()
	chunks4, _, _ := eng4.processFilesParallel(ctx, db4, files, 4, nil)
	dur4 := time.Since(start4)
	t.Logf("✓ Parallel files + Parallel pages: %v (%d chunks)", dur4, len(chunks4))
	t.Logf("  Speedup vs baseline: %.2fx", float64(dur1)/float64(dur4))
	t.Log("")

	// Verify correctness
	t.Log("========================================")
	t.Log("VERIFICATION")
	t.Log("========================================")
	if len(chunks2) == len(chunks1) && len(chunks3) == len(chunks1) && len(chunks4) == len(chunks1) {
		t.Logf("✓ All configurations produced %d chunks (correct)", len(chunks1))
	} else {
		t.Errorf("❌ Chunk count mismatch: seq+seq=%d, par+seq=%d, seq+par=%d, par+par=%d",
			len(chunks1), len(chunks2), len(chunks3), len(chunks4))
	}
	t.Log("")

	// Summary
	t.Log("========================================")
	t.Log("PERFORMANCE SUMMARY")
	t.Log("========================================")
	t.Logf("Baseline (seq files + seq pages):  %v", dur1)
	t.Logf("Parallel files only:                %v (%.2fx)", dur2, float64(dur1)/float64(dur2))
	t.Logf("Parallel PDF pages only:            %v (%.2fx)", dur3, float64(dur1)/float64(dur3))
	t.Logf("Parallel files + Parallel pages:    %v (%.2fx) ← BEST", dur4, float64(dur1)/float64(dur4))
}

// BenchmarkPDFParallelPages benchmarks different parallelization strategies.
func BenchmarkPDFParallelPages(b *testing.B) {
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

	configs := []struct {
		name            string
		parallelFiles   bool
		parallelPages   bool
		fileWorkers     int
		pageWorkers     int
	}{
		{"SequentialBoth", false, false, 0, 0},
		{"ParallelFiles", true, false, 4, 0},
		{"ParallelPages", false, true, 0, 4},
		{"ParallelBoth", true, true, 4, 4},
	}

	for _, cfg := range configs {
		b.Run(cfg.name, func(b *testing.B) {
			eng := createBenchEngine(128)
			if cfg.parallelFiles {
				eng.SetParallelProcessing(true, cfg.fileWorkers, 0)
			}
			if cfg.parallelPages {
				eng.SetParallelPDFProcessing(true, cfg.pageWorkers)
			}

			b.ResetTimer()
			b.ReportMetric(float64(len(files)), "files")

			for n := 0; n < b.N; n++ {
				dbPath := filepath.Join(b.TempDir(), "bench.db")
				projectDB, err := db.OpenProjectDB(dbPath, 128)
				if err != nil {
					b.Fatal(err)
				}

				ctx := context.Background()
				var chunks []db.Chunk
				if cfg.parallelFiles {
					chunks, _, _ = eng.processFilesParallel(ctx, projectDB, files, cfg.fileWorkers, nil)
				} else {
					chunks, _, _ = eng.processFilesParallel(ctx, projectDB, files, 1, nil)
				}

				b.ReportMetric(float64(len(chunks)), "chunks")
				projectDB.Close()
			}
		})
	}
}
