package processor

import (
	"fmt"
	"os"
	"time"

	"github.com/unidoc/unipdf/v4/extractor"
	"github.com/unidoc/unipdf/v4/model"
)

// PDFMetadata contains metadata extracted from a PDF
type PDFMetadata struct {
	Title        string
	Author       string
	Subject      string
	Creator      string
	Producer     string
	CreationDate time.Time
	ModDate      time.Time
	PageCount    int
}

// PDFPage represents a single page from a PDF
type PDFPage struct {
	PageNumber int
	Text       string
}

// PDFResult contains the complete extraction result
type PDFResult struct {
	Pages    []PDFPage
	Metadata *PDFMetadata
}

// IPDFProcessor defines the interface for PDF processing
type IPDFProcessor interface {
	ExtractText(filePath string) (*PDFResult, error)
	ExtractMetadata(filePath string) (*PDFMetadata, error)
}

// PDFProcessor implements IPDFProcessor using UniPDF
type PDFProcessor struct {
	verbose bool
}

// Compile-time check that PDFProcessor implements IPDFProcessor
var _ IPDFProcessor = (*PDFProcessor)(nil)

// NewPDFProcessor creates a new PDF processor
func NewPDFProcessor(verbose bool) (*PDFProcessor, error) {
	// Check if license initialization failed
	if licenseInitErr != nil {
		return nil, licenseInitErr
	}

	return &PDFProcessor{
		verbose: verbose,
	}, nil
}

// ExtractText extracts text and metadata from all pages of a PDF file
func (p *PDFProcessor) ExtractText(filePath string) (*PDFResult, error) {
	reader, err := p.openPDF(filePath)
	if err != nil {
		return nil, err
	}

	numPages, err := reader.GetNumPages()
	if err != nil {
		return nil, fmt.Errorf("failed to get page count: %w", err)
	}

	if p.verbose {
		fmt.Printf("Extracting text from %d pages in: %s\n", numPages, filePath)
	}

	pages, err := p.extractPages(reader, numPages)
	if err != nil {
		return nil, err
	}

	metadata, err := p.extractMetadataFromReader(reader, numPages)
	if err != nil {
		return nil, err
	}

	return &PDFResult{
		Pages:    pages,
		Metadata: metadata,
	}, nil
}

// ExtractMetadata extracts only metadata without processing pages
func (p *PDFProcessor) ExtractMetadata(filePath string) (*PDFMetadata, error) {
	reader, err := p.openPDF(filePath)
	if err != nil {
		return nil, err
	}

	numPages, err := reader.GetNumPages()
	if err != nil {
		return nil, fmt.Errorf("failed to get page count: %w", err)
	}

	return p.extractMetadataFromReader(reader, numPages)
}

// openPDF opens a PDF file and returns a reader
func (p *PDFProcessor) openPDF(filePath string) (*model.PdfReader, error) {
	f, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open PDF file: %w", err)
	}

	reader, err := model.NewPdfReader(f)
	if err != nil {
		f.Close()
		return nil, fmt.Errorf("failed to create PDF reader: %w", err)
	}

	return reader, nil
}

// extractPages extracts text from all pages
func (p *PDFProcessor) extractPages(reader *model.PdfReader, numPages int) ([]PDFPage, error) {
	pages := make([]PDFPage, 0, numPages)

	for i := 0; i < numPages; i++ {
		pageNum := i + 1
		page, err := reader.GetPage(pageNum)
		if err != nil {
			return nil, fmt.Errorf("failed to get page %d: %w", pageNum, err)
		}

		ex, err := extractor.New(page)
		if err != nil {
			return nil, fmt.Errorf("failed to create extractor for page %d: %w", pageNum, err)
		}

		text, err := ex.ExtractText()
		if err != nil {
			return nil, fmt.Errorf("failed to extract text from page %d: %w", pageNum, err)
		}

		if p.verbose {
			fmt.Printf("Page %d: extracted %d characters\n", pageNum, len(text))
		}

		pages = append(pages, PDFPage{
			PageNumber: pageNum,
			Text:       text,
		})
	}

	return pages, nil
}

// extractMetadataFromReader extracts metadata from a PDF reader
func (p *PDFProcessor) extractMetadataFromReader(reader *model.PdfReader, pageCount int) (*PDFMetadata, error) {
	pdfInfo, err := reader.GetPdfInfo()
	if err != nil {
		return &PDFMetadata{PageCount: pageCount}, nil // Return basic metadata on error
	}

	metadata := &PDFMetadata{
		PageCount: pageCount,
	}

	if pdfInfo.Title != nil {
		metadata.Title = pdfInfo.Title.Decoded()
	}
	if pdfInfo.Author != nil {
		metadata.Author = pdfInfo.Author.Decoded()
	}
	if pdfInfo.Subject != nil {
		metadata.Subject = pdfInfo.Subject.Decoded()
	}
	if pdfInfo.Creator != nil {
		metadata.Creator = pdfInfo.Creator.Decoded()
	}
	if pdfInfo.Producer != nil {
		metadata.Producer = pdfInfo.Producer.Decoded()
	}
	if pdfInfo.CreationDate != nil {
		metadata.CreationDate = pdfInfo.CreationDate.ToGoTime()
	}
	if pdfInfo.ModifiedDate != nil {
		metadata.ModDate = pdfInfo.ModifiedDate.ToGoTime()
	}

	return metadata, nil
}

// ParseDate converts PDF date string to time.Time
func ParseDate(dateStr string) (time.Time, error) {
	// PDF date format: D:YYYYMMDDHHmmSSOHH'mm
	layouts := []string{
		"D:20060102150405-07'00'",
		"D:20060102150405Z",
		"D:20060102150405",
		"D:20060102",
	}

	for _, layout := range layouts {
		if t, err := time.Parse(layout, dateStr); err == nil {
			return t, nil
		}
	}

	return time.Time{}, fmt.Errorf("unable to parse date: %s", dateStr)
}
