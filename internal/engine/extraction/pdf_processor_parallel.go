package extraction

import (
	"fmt"
	"io"
	"runtime"
	"sync"

	"github.com/ledongthuc/pdf"
)

// ExtractTextFromReaderParallel extracts text from a PDF read from memory using parallel processing.
func (p *PDFProcessor) ExtractTextFromReaderParallel(r io.ReaderAt, size int64, filePath string, numWorkers int) ([]PDFPage, *PDFMetadata, error) {
	reader, err := pdf.NewReader(r, size)
	if err != nil {
		errMsg := err.Error()
		if contains(errMsg, "encrypted") || contains(errMsg, "password") {
			return nil, nil, fmt.Errorf("PDF is password protected: %s", filePath)
		}
		return nil, nil, fmt.Errorf("failed to open PDF reader: %w", err)
	}
	metadata := p.extractMetadata(reader, filePath)
	pageCount := reader.NumPage()
	if pageCount < 5 {
		return p.ExtractTextFromReader(r, size, filePath)
	}
	if numWorkers <= 0 {
		numWorkers = int(float64(runtime.NumCPU()) * 0.9)
		if numWorkers < 1 {
			numWorkers = 1
		}
	}
	pages := make([]PDFPage, pageCount)
	type pageJob struct {
		pageNum int
		reader  *pdf.Reader
	}
	jobs := make(chan pageJob, numWorkers*2)
	var wg sync.WaitGroup
	for w := 0; w < numWorkers; w++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for job := range jobs {
				page := job.reader.Page(job.pageNum)
				var text string
				if !page.V.IsNull() {
					if extractedText, err := p.extractPageText(page); err == nil {
						text = extractedText
					}
				}
				pages[job.pageNum-1] = PDFPage{PageNumber: job.pageNum, Text: text}
			}
		}()
	}
	for i := 1; i <= pageCount; i++ {
		jobs <- pageJob{pageNum: i, reader: reader}
	}
	close(jobs)
	wg.Wait()
	return pages, metadata, nil
}

// ExtractTextParallel extracts text from all pages of a PDF using parallel processing.
// This is beneficial for large PDFs with many pages.
func (p *PDFProcessor) ExtractTextParallel(filePath string, numWorkers int) ([]PDFPage, *PDFMetadata, error) {
	// Open the PDF file
	file, reader, err := pdf.Open(filePath)
	if err != nil {
		errMsg := err.Error()
		if contains(errMsg, "encrypted") || contains(errMsg, "password") {
			return nil, nil, fmt.Errorf("PDF is password protected: %s", filePath)
		}
		return nil, nil, fmt.Errorf("failed to open PDF: %w", err)
	}
	defer file.Close()

	// Extract metadata
	metadata := p.extractMetadata(reader, filePath)
	pageCount := reader.NumPage()

	// For small PDFs (< 5 pages), use sequential processing
	if pageCount < 5 {
		return p.ExtractText(filePath)
	}

	// Target ~90% CPU utilization by default
	if numWorkers <= 0 {
		numWorkers = int(float64(runtime.NumCPU()) * 0.9)
		if numWorkers < 1 {
			numWorkers = 1
		}
	}

	// Create result slice with correct size
	pages := make([]PDFPage, pageCount)

	// Worker pool for parallel page processing
	type pageJob struct {
		pageNum int
		reader  *pdf.Reader
	}

	jobs := make(chan pageJob, numWorkers*2)
	var wg sync.WaitGroup

	// Launch workers
	for w := 0; w < numWorkers; w++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for job := range jobs {
				page := job.reader.Page(job.pageNum)
				var text string

				if !page.V.IsNull() {
					extractedText, err := p.extractPageText(page)
					if err == nil {
						text = extractedText
					}
				}

				pages[job.pageNum-1] = PDFPage{
					PageNumber: job.pageNum,
					Text:       text,
				}
			}
		}()
	}

	// Feed jobs
	for i := 1; i <= pageCount; i++ {
		jobs <- pageJob{pageNum: i, reader: reader}
	}
	close(jobs)

	// Wait for completion
	wg.Wait()

	return pages, metadata, nil
}

// contains checks if string s contains substr (helper to avoid import cycles)
func contains(s, substr string) bool {
	if len(substr) == 0 {
		return true
	}
	if len(s) < len(substr) {
		return false
	}
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
