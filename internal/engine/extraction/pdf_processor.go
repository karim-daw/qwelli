package extraction

import (
	"fmt"
	"io"
	"path/filepath"
	"strings"
	"time"

	"github.com/ledongthuc/pdf"
)

// PDFMetadata contains metadata extracted from a PDF
type PDFMetadata struct {
	Title        string
	Creator      string
	CreationDate time.Time
	ModDate      time.Time
	PageCount    int
}

// PDFPage represents a single page from a PDF
type PDFPage struct {
	PageNumber int
	Text       string
}

// PDFProcessor handles PDF text extraction
type PDFProcessor struct{}

// NewPDFProcessor creates a new PDF processor
func NewPDFProcessor() *PDFProcessor {
	return &PDFProcessor{}
}

// ExtractTextFromReader extracts text from a PDF read from memory (io.ReaderAt).
// filePath is used for metadata fallback (e.g. title from filename).
func (p *PDFProcessor) ExtractTextFromReader(r io.ReaderAt, size int64, filePath string) ([]PDFPage, *PDFMetadata, error) {
	reader, err := pdf.NewReader(r, size)
	if err != nil {
		errMsg := err.Error()
		if strings.Contains(errMsg, "encrypted") || strings.Contains(errMsg, "password") {
			return nil, nil, fmt.Errorf("PDF is password protected: %s", filePath)
		}
		return nil, nil, fmt.Errorf("failed to open PDF reader: %w", err)
	}
	metadata := p.extractMetadata(reader, filePath)
	var pages []PDFPage
	pageCount := reader.NumPage()
	for i := 1; i <= pageCount; i++ {
		page := reader.Page(i)
		if page.V.IsNull() {
			pages = append(pages, PDFPage{PageNumber: i, Text: ""})
			continue
		}
		text, err := p.extractPageText(page)
		if err != nil {
			text = ""
		}
		pages = append(pages, PDFPage{PageNumber: i, Text: text})
	}
	return pages, metadata, nil
}

// ExtractText extracts text from all pages of a PDF file
func (p *PDFProcessor) ExtractText(filePath string) ([]PDFPage, *PDFMetadata, error) {
	// Open the PDF file
	file, reader, err := pdf.Open(filePath)
	if err != nil {
		// Check for common error types
		errMsg := err.Error()
		if strings.Contains(errMsg, "encrypted") || strings.Contains(errMsg, "password") {
			return nil, nil, fmt.Errorf("PDF is password protected: %s", filePath)
		}
		return nil, nil, fmt.Errorf("failed to open PDF: %w", err)
	}
	defer file.Close()

	// Extract metadata
	metadata := p.extractMetadata(reader, filePath)

	// Extract text from each page
	var pages []PDFPage
	pageCount := reader.NumPage()

	for i := 1; i <= pageCount; i++ {
		page := reader.Page(i)
		if page.V.IsNull() {
			// Empty or invalid page
			pages = append(pages, PDFPage{
				PageNumber: i,
				Text:       "",
			})
			continue
		}

		// Extract text from the page
		text, err := p.extractPageText(page)
		if err != nil {
			// Log error but continue with empty text
			text = ""
		}

		pages = append(pages, PDFPage{
			PageNumber: i,
			Text:       text,
		})
	}

	return pages, metadata, nil
}

// extractPageText extracts and cleans text from a single page
func (p *PDFProcessor) extractPageText(page pdf.Page) (string, error) {
	// Get plain text from the page
	text, err := page.GetPlainText(nil)
	if err != nil {
		return "", fmt.Errorf("failed to extract text: %w", err)
	}

	// Clean and normalize the text
	text = p.cleanText(text)

	return text, nil
}

// cleanText normalizes and cleans extracted text
func (p *PDFProcessor) cleanText(text string) string {
	// Replace multiple spaces with single space
	text = strings.Join(strings.Fields(text), " ")

	// Trim whitespace
	text = strings.TrimSpace(text)

	return text
}

// extractMetadata extracts metadata from the PDF
func (p *PDFProcessor) extractMetadata(reader *pdf.Reader, filePath string) *PDFMetadata {
	metadata := &PDFMetadata{
		PageCount: reader.NumPage(),
	}

	// Try to get title from PDF metadata
	if title := reader.Trailer().Key("Info").Key("Title").String(); title != "" {
		metadata.Title = strings.Trim(title, "()")
	}

	// Fallback to filename if no title in PDF
	if metadata.Title == "" {
		metadata.Title = filepath.Base(filePath)
		// Remove .pdf extension
		if strings.HasSuffix(strings.ToLower(metadata.Title), ".pdf") {
			metadata.Title = metadata.Title[:len(metadata.Title)-4]
		}
	}

	// Extract creator
	if creator := reader.Trailer().Key("Info").Key("Creator").String(); creator != "" {
		metadata.Creator = strings.Trim(creator, "()")
	}

	// Extract creation date
	if creationDate := reader.Trailer().Key("Info").Key("CreationDate").String(); creationDate != "" {
		metadata.CreationDate = p.parsePDFDate(creationDate)
	}

	// Extract modification date
	if modDate := reader.Trailer().Key("Info").Key("ModDate").String(); modDate != "" {
		metadata.ModDate = p.parsePDFDate(modDate)
	}

	return metadata
}

// parsePDFDate parses a PDF date string
// PDF dates are in format: D:YYYYMMDDHHmmSSOHH'mm'
func (p *PDFProcessor) parsePDFDate(dateStr string) time.Time {
	// Remove parentheses if present
	dateStr = strings.Trim(dateStr, "()")

	// Remove D: prefix if present
	dateStr = strings.TrimPrefix(dateStr, "D:")

	// Try to parse common PDF date formats
	formats := []string{
		"20060102150405-07'00'",
		"20060102150405",
		"20060102",
		time.RFC3339,
	}

	for _, format := range formats {
		if t, err := time.Parse(format, dateStr); err == nil {
			return t
		}
	}

	// Return zero time if parsing fails
	return time.Time{}
}
