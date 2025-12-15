# PDF Test Samples

This directory contains sample PDF files for testing the PDF extraction functionality.

## How to Add Test PDFs

You can add test PDFs in several ways:

1. **Use an existing PDF**: Copy any PDF file here and rename it to `simple.pdf`
2. **Create a PDF**: Use any tool to create a simple PDF with some text
3. **Download a sample**: Get a sample PDF from the internet

## Required Test Files

- `simple.pdf` - A simple 1-2 page PDF with basic text for testing

## Manual Testing

To manually test PDF extraction, you can run:

```bash
go test ./internal/processor -v -run TestExtractText_WithSamplePDF
```

This test will only run if a `simple.pdf` file exists in this directory.
