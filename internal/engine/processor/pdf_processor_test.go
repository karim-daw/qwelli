package processor

import (
	"os"
	"path/filepath"
	"testing"
)

func TestNewPDFProcessor(t *testing.T) {
	processor, err := NewPDFProcessor(true)
	if err != nil {
		t.Fatal("NewPDFProcessor returned error:", err)
	}
	if processor == nil {
		t.Fatal("NewPDFProcessor returned nil")
	}
	if !processor.verbose {
		t.Error("Expected verbose to be true")
	}
}

func TestNewPDFProcessor_NonVerbose(t *testing.T) {
	processor, err := NewPDFProcessor(false)
	if err != nil {
		t.Fatal("NewPDFProcessor returned error:", err)
	}
	if processor.verbose {
		t.Error("Expected verbose to be false")
	}
}

// TestExtractText_WithSamplePDF tests PDF extraction with an actual PDF file
func TestExtractText_WithSamplePDF(t *testing.T) {
	// Get current working directory
	wd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get working directory: %v", err)
	}

	// Build paths relative to workspace root
	possiblePaths := []string{
		filepath.Join(wd, "..", "..", "..", "tests", "demo", "pdf_samples", "simple.pdf"),
		filepath.Join("..", "..", "..", "tests", "demo", "pdf_samples", "simple.pdf"),
	}

	var path string
	for _, p := range possiblePaths {
		absPath, err := filepath.Abs(p)
		if err != nil {
			continue
		}
		if _, err := os.Stat(absPath); err == nil {
			path = absPath
			break
		}
	}

	if path == "" {
		var triedAbs []string
		for _, p := range possiblePaths {
			if abs, err := filepath.Abs(p); err == nil {
				triedAbs = append(triedAbs, abs)
			}
		}
		t.Fatalf("Sample PDF not found. Working directory: %s. Tried absolute paths: %v", wd, triedAbs)
	}

	processor, err := NewPDFProcessor(true)
	if err != nil {
		t.Fatalf("Failed to create processor: %v", err)
	}

	result, err := processor.ExtractText(path)
	if err != nil {
		t.Fatalf("ExtractText failed: %v", err)
	}

	if result == nil {
		t.Fatal("Result is nil")
	}

	if result.Metadata == nil {
		t.Fatal("Metadata is nil")
	}

	if result.Metadata.PageCount == 0 {
		t.Error("PageCount is 0")
	}

	if len(result.Pages) != result.Metadata.PageCount {
		t.Errorf("Expected %d pages, got %d", result.Metadata.PageCount, len(result.Pages))
	}

	// Verify page numbers are sequential
	for i, page := range result.Pages {
		expectedPageNum := i + 1
		if page.PageNumber != expectedPageNum {
			t.Errorf("Page %d has incorrect page number: %d", i, page.PageNumber)
		}

		// Verify each page has some text (unless it's a blank page)
		if len(page.Text) == 0 {
			t.Logf("Warning: Page %d has no text", page.PageNumber)
		}
	}

	// Log results
	t.Logf("Successfully extracted %d pages from PDF", len(result.Pages))
	if result.Metadata.Title != "" {
		t.Logf("Title: %s", result.Metadata.Title)
	}
	if result.Metadata.Author != "" {
		t.Logf("Author: %s", result.Metadata.Author)
	}
	if !result.Metadata.CreationDate.IsZero() {
		t.Logf("Creation Date: %v", result.Metadata.CreationDate)
	}
}

// TestExtractMetadata_WithSamplePDF tests metadata extraction only
func TestExtractMetadata_WithSamplePDF(t *testing.T) {
	wd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get working directory: %v", err)
	}

	possiblePaths := []string{
		filepath.Join(wd, "..", "..", "..", "tests", "demo", "pdf_samples", "BillFile5086630.pdf"),
		filepath.Join(wd, "..", "..", "..", "tests", "demo", "pdf_samples", "simple.pdf"),
	}

	var path string
	for _, p := range possiblePaths {
		absPath, err := filepath.Abs(p)
		if err != nil {
			continue
		}
		if _, err := os.Stat(absPath); err == nil {
			path = absPath
			break
		}
	}

	if path == "" {
		t.Skip("Sample PDF not found, skipping metadata test")
	}

	processor, err := NewPDFProcessor(false)
	if err != nil {
		t.Fatalf("Failed to create processor: %v", err)
	}

	metadata, err := processor.ExtractMetadata(path)
	if err != nil {
		t.Fatalf("ExtractMetadata failed: %v", err)
	}

	if metadata == nil {
		t.Fatal("Metadata is nil")
	}

	if metadata.PageCount == 0 {
		t.Error("PageCount is 0")
	}

	t.Logf("Metadata extracted successfully")
	t.Logf("Page Count: %d", metadata.PageCount)
	if metadata.Title != "" {
		t.Logf("Title: %s", metadata.Title)
	}
}

// TestExtractText_NonExistentFile tests error handling for missing files
func TestExtractText_NonExistentFile(t *testing.T) {
	processor, err := NewPDFProcessor(false)
	if err != nil {
		t.Fatalf("Failed to create processor: %v", err)
	}

	_, err = processor.ExtractText("nonexistent.pdf")
	if err == nil {
		t.Error("Expected error for non-existent file, got nil")
	}

	// Verify error message is helpful
	if err != nil {
		t.Logf("Error message: %v", err)
	}
}

// TestExtractText_InvalidPDF tests error handling for invalid PDF files
func TestExtractText_InvalidPDF(t *testing.T) {
	// Create a temporary invalid PDF file
	tmpFile, err := os.CreateTemp("", "invalid*.pdf")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())

	// Write some invalid content
	if _, err := tmpFile.WriteString("This is not a valid PDF file"); err != nil {
		t.Fatalf("Failed to write to temp file: %v", err)
	}
	tmpFile.Close()

	processor, err := NewPDFProcessor(false)
	if err != nil {
		t.Fatalf("Failed to create processor: %v", err)
	}

	_, err = processor.ExtractText(tmpFile.Name())
	if err == nil {
		t.Error("Expected error for invalid PDF, got nil")
	}

	// Verify we get a meaningful error
	if err != nil {
		t.Logf("Error for invalid PDF: %v", err)
	}
}

// TestExtractMetadata_NonExistentFile tests metadata extraction error handling
func TestExtractMetadata_NonExistentFile(t *testing.T) {
	processor, err := NewPDFProcessor(false)
	if err != nil {
		t.Fatalf("Failed to create processor: %v", err)
	}

	_, err = processor.ExtractMetadata("nonexistent.pdf")
	if err == nil {
		t.Error("Expected error for non-existent file, got nil")
	}
}

// TestInterfaceImplementation verifies PDFProcessor implements IPDFProcessor
func TestInterfaceImplementation(t *testing.T) {
	processor, err := NewPDFProcessor(false)
	if err != nil {
		t.Fatalf("Failed to create processor: %v", err)
	}

	// This will fail at compile time if PDFProcessor doesn't implement IPDFProcessor
	var _ IPDFProcessor = processor

	t.Log("PDFProcessor correctly implements IPDFProcessor interface")
}

// TestParseDateFormats tests the ParseDate utility function
func TestParseDateFormats(t *testing.T) {
	testCases := []struct {
		input       string
		shouldError bool
		description string
	}{
		{"D:20231215143022-08'00'", false, "Full date with timezone"},
		{"D:20231215143022Z", false, "Full date with Z timezone"},
		{"D:20231215143022", false, "Full date without timezone"},
		{"D:20231215", false, "Date only"},
		{"invalid date", true, "Invalid format"},
		{"", true, "Empty string"},
	}

	for _, tc := range testCases {
		t.Run(tc.description, func(t *testing.T) {
			result, err := ParseDate(tc.input)

			if tc.shouldError {
				if err == nil {
					t.Errorf("Expected error for input '%s', got nil", tc.input)
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error for input '%s': %v", tc.input, err)
				}
				if result.IsZero() {
					t.Errorf("Expected valid time for input '%s', got zero time", tc.input)
				}
			}
		})
	}
}

// BenchmarkExtractText benchmarks full text extraction
func BenchmarkExtractText(b *testing.B) {
	wd, err := os.Getwd()
	if err != nil {
		b.Fatalf("Failed to get working directory: %v", err)
	}

	path := filepath.Join(wd, "..", "..", "..", "tests", "demo", "pdf_samples", "simple.pdf")
	absPath, err := filepath.Abs(path)
	if err != nil {
		b.Skip("Could not resolve path")
	}

	if _, err := os.Stat(absPath); err != nil {
		b.Skip("Sample PDF not found")
	}

	processor, err := NewPDFProcessor(false)
	if err != nil {
		b.Fatalf("Failed to create processor: %v", err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := processor.ExtractText(absPath)
		if err != nil {
			b.Fatalf("ExtractText failed: %v", err)
		}
	}
}

// BenchmarkExtractMetadata benchmarks metadata-only extraction
func BenchmarkExtractMetadata(b *testing.B) {
	wd, err := os.Getwd()
	if err != nil {
		b.Fatalf("Failed to get working directory: %v", err)
	}

	path := filepath.Join(wd, "..", "..", "..", "tests", "demo", "pdf_samples", "simple.pdf")
	absPath, err := filepath.Abs(path)
	if err != nil {
		b.Skip("Could not resolve path")
	}

	if _, err := os.Stat(absPath); err != nil {
		b.Skip("Sample PDF not found")
	}

	processor, err := NewPDFProcessor(false)
	if err != nil {
		b.Fatalf("Failed to create processor: %v", err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := processor.ExtractMetadata(absPath)
		if err != nil {
			b.Fatalf("ExtractMetadata failed: %v", err)
		}
	}
}
