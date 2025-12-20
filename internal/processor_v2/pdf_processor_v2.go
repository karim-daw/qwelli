//go:build ignore

package main

import (
	"fmt"
	"os"

	"github.com/ledongthuc/pdf"
	"github.com/pdfcpu/pdfcpu/pkg/api"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/model"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Println("usage: go run test_extract.go <pdf_path>")
		os.Exit(1)
	}

	pdfPath := os.Args[1]
	fmt.Printf("Testing extraction: %s\n\n", pdfPath)

	// Text extraction
	fmt.Println("=== TEXT ===")
	f, r, err := pdf.Open(pdfPath)
	if err != nil {
		fmt.Printf("Error opening PDF: %v\n", err)
	} else {
		defer f.Close()
		for i := 1; i <= r.NumPage(); i++ {
			p := r.Page(i)
			if p.V.IsNull() {
				continue
			}
			text, err := p.GetPlainText(nil)
			if err != nil {
				fmt.Printf("Page %d: error - %v\n", i, err)
				continue
			}
			preview := text
			if len(preview) > 200 {
				preview = preview[:200] + "..."
			}
			fmt.Printf("Page %d: %d chars\n%s\n\n", i, len(text), preview)
		}
	}

	// Image extraction
	fmt.Println("=== IMAGES ===")
	tmpDir := "images"
	err = os.Mkdir(tmpDir, 0755)
	if err != nil {
		fmt.Printf("Error creating temp dir: %v\n", err)
		os.Exit(1)
	}

	conf := model.NewDefaultConfiguration()
	err = api.ExtractImagesFile(pdfPath, tmpDir, nil, conf)
	if err != nil {
		fmt.Printf("Error extracting images: %v\n", err)
	} else {
		entries, _ := os.ReadDir(tmpDir)
		if len(entries) == 0 {
			fmt.Println("No images found")
		}
		for _, e := range entries {
			info, _ := e.Info()
			fmt.Printf("%s (%d bytes)\n", e.Name(), info.Size())
		}
	}
}
