package extraction

import (
	"runtime"
	"sync"
)

// ExtractImagesByPagesParallel extracts images from multiple pages in parallel.
// This is more efficient for large PDFs with many pages containing images.
func (e *ImageExtractor) ExtractImagesByPagesParallel(pdfPath string, pageNumbers []int, numWorkers int) ([]PDFImage, error) {
	if len(pageNumbers) == 0 {
		return []PDFImage{}, nil
	}

	// For small page counts, use sequential extraction
	if len(pageNumbers) < 5 {
		var allImages []PDFImage
		for _, pageNum := range pageNumbers {
			images, err := e.ExtractImagesByPage(pdfPath, pageNum)
			if err != nil {
				// Skip this page on error (graceful degradation)
				continue
			}
			allImages = append(allImages, images...)
		}
		return allImages, nil
	}

	// Target ~90% CPU utilization by default
	if numWorkers <= 0 {
		numWorkers = int(float64(runtime.NumCPU()) * 0.9)
		if numWorkers < 1 {
			numWorkers = 1
		}
	}

	// Result collection with mutex
	var (
		allImages []PDFImage
		mu        sync.Mutex
	)

	// Worker pool
	jobs := make(chan int, numWorkers*2)
	var wg sync.WaitGroup

	// Launch workers
	for w := 0; w < numWorkers; w++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for pageNum := range jobs {
				images, err := e.ExtractImagesByPage(pdfPath, pageNum)
				if err != nil {
					// Skip this page on error
					continue
				}
				if len(images) > 0 {
					mu.Lock()
					allImages = append(allImages, images...)
					mu.Unlock()
				}
			}
		}()
	}

	// Feed jobs
	for _, pageNum := range pageNumbers {
		jobs <- pageNum
	}
	close(jobs)

	// Wait for completion
	wg.Wait()

	return allImages, nil
}
