package extraction

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"image"
	_ "image/jpeg"
	_ "image/png"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"sync"
)

// PageRenderer renders PDF pages to images using external tools.
// This is much faster than extracting individual embedded images for image-heavy PDFs.
type PageRenderer struct {
	toolPath   string // Path to pdftoppm or similar tool
	toolType   string // "pdftoppm", "mutool", or "none"
	maxWidth   int
	maxHeight  int
	dpi        int
	initOnce   sync.Once
	initResult bool
}

// NewPageRenderer creates a new page renderer with default settings.
// It auto-detects available PDF rendering tools on the system.
func NewPageRenderer() *PageRenderer {
	r := &PageRenderer{
		maxWidth:  1600, // Max width for rendered pages
		maxHeight: 2000, // Max height for rendered pages
		dpi:       150,  // Good balance of quality vs size
	}
	return r
}

// IsAvailable returns true if a PDF rendering tool is available on the system.
func (r *PageRenderer) IsAvailable() bool {
	r.initOnce.Do(r.detectTool)
	return r.initResult
}

// detectTool checks for available PDF rendering tools
func (r *PageRenderer) detectTool() {
	// Try pdftoppm first (poppler-utils) - most common
	if path, err := exec.LookPath("pdftoppm"); err == nil {
		r.toolPath = path
		r.toolType = "pdftoppm"
		r.initResult = true
		log.Printf("  Page renderer: using pdftoppm at %s", path)
		return
	}

	// Try mutool (MuPDF) as fallback
	if path, err := exec.LookPath("mutool"); err == nil {
		r.toolPath = path
		r.toolType = "mutool"
		r.initResult = true
		log.Printf("  Page renderer: using mutool at %s", path)
		return
	}

	// On Windows, check common installation paths
	if runtime.GOOS == "windows" {
		commonPaths := []string{
			`C:\Program Files\poppler\bin\pdftoppm.exe`,
			`C:\Program Files (x86)\poppler\bin\pdftoppm.exe`,
			`C:\poppler\bin\pdftoppm.exe`,
		}
		for _, p := range commonPaths {
			if _, err := os.Stat(p); err == nil {
				r.toolPath = p
				r.toolType = "pdftoppm"
				r.initResult = true
				log.Printf("  Page renderer: using pdftoppm at %s", p)
				return
			}
		}
	}

	log.Printf("  Page renderer: no PDF rendering tool found (install poppler-utils for faster image processing)")
	r.initResult = false
}

// RenderPage renders a single PDF page to an image.
// Returns the image as a PDFImage struct with base64-encoded data.
func (r *PageRenderer) RenderPage(pdfPath string, pageNumber int) (*PDFImage, error) {
	if !r.IsAvailable() {
		return nil, fmt.Errorf("no PDF rendering tool available")
	}

	tmpDir, err := os.MkdirTemp("", "qwelli-render-*")
	if err != nil {
		return nil, fmt.Errorf("failed to create temp directory: %w", err)
	}
	defer os.RemoveAll(tmpDir)

	outputPrefix := filepath.Join(tmpDir, "page")

	var cmd *exec.Cmd
	switch r.toolType {
	case "pdftoppm":
		// pdftoppm -png -f <page> -l <page> -r <dpi> input.pdf output_prefix
		cmd = exec.Command(r.toolPath,
			"-png",
			"-f", strconv.Itoa(pageNumber),
			"-l", strconv.Itoa(pageNumber),
			"-r", strconv.Itoa(r.dpi),
			pdfPath,
			outputPrefix,
		)
	case "mutool":
		// mutool draw -o output.png -r <dpi> input.pdf <page>
		outputPath := outputPrefix + ".png"
		cmd = exec.Command(r.toolPath, "draw",
			"-o", outputPath,
			"-r", strconv.Itoa(r.dpi),
			pdfPath,
			strconv.Itoa(pageNumber),
		)
	default:
		return nil, fmt.Errorf("unknown tool type: %s", r.toolType)
	}

	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("render failed: %w (stderr: %s)", err, stderr.String())
	}

	// Find the output file
	entries, err := os.ReadDir(tmpDir)
	if err != nil {
		return nil, fmt.Errorf("failed to read output directory: %w", err)
	}

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		if !strings.HasSuffix(strings.ToLower(name), ".png") &&
			!strings.HasSuffix(strings.ToLower(name), ".jpg") {
			continue
		}

		imagePath := filepath.Join(tmpDir, name)
		imageData, err := os.ReadFile(imagePath)
		if err != nil {
			return nil, fmt.Errorf("failed to read rendered image: %w", err)
		}

		// Decode to get dimensions
		img, format, err := image.Decode(bytes.NewReader(imageData))
		if err != nil {
			return nil, fmt.Errorf("failed to decode rendered image: %w", err)
		}

		bounds := img.Bounds()
		width := bounds.Dx()
		height := bounds.Dy()

		base64Data := base64.StdEncoding.EncodeToString(imageData)

		return &PDFImage{
			Data:       imageData,
			Format:     format,
			Width:      width,
			Height:     height,
			PageNumber: pageNumber,
			Base64:     base64Data,
		}, nil
	}

	return nil, fmt.Errorf("no output image found after rendering")
}

// RenderPagesParallel renders multiple pages in parallel.
// This is efficient for rendering many pages at once.
func (r *PageRenderer) RenderPagesParallel(pdfPath string, pageNumbers []int, numWorkers int) ([]PDFImage, error) {
	if !r.IsAvailable() {
		return nil, fmt.Errorf("no PDF rendering tool available")
	}

	if len(pageNumbers) == 0 {
		return []PDFImage{}, nil
	}

	if numWorkers <= 0 {
		numWorkers = int(float64(runtime.NumCPU()) * 0.9)
		if numWorkers < 1 {
			numWorkers = 1
		}
	}

	// For small page counts, render sequentially
	if len(pageNumbers) < 3 {
		var images []PDFImage
		for _, pageNum := range pageNumbers {
			img, err := r.RenderPage(pdfPath, pageNum)
			if err != nil {
				log.Printf("  Failed to render page %d: %v", pageNum, err)
				continue
			}
			images = append(images, *img)
		}
		return images, nil
	}

	// Parallel rendering
	var (
		images []PDFImage
		mu     sync.Mutex
	)

	jobs := make(chan int, numWorkers*2)
	var wg sync.WaitGroup

	for w := 0; w < numWorkers; w++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for pageNum := range jobs {
				img, err := r.RenderPage(pdfPath, pageNum)
				if err != nil {
					log.Printf("  Failed to render page %d: %v", pageNum, err)
					continue
				}
				mu.Lock()
				images = append(images, *img)
				mu.Unlock()
			}
		}()
	}

	for _, pageNum := range pageNumbers {
		jobs <- pageNum
	}
	close(jobs)

	wg.Wait()

	return images, nil
}
