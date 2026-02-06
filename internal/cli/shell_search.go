package cli

import (
	"fmt"
	"strings"

	"github.com/karim-daw/qwelli/internal/engine"
	"github.com/karim-daw/qwelli/internal/service"
)

// runSearchShell performs a search from the interactive shell.
func runSearchShell(query, indexPath string, topK int, textOnly, imagesOnly bool, strategy string) error {
	return executeSearch(query, indexPath, topK, textOnly, imagesOnly, strategy, false)
}

// displaySearchResults performs a search and prints results.
// Shared between CLI search and interactive shell.
func displaySearchResults(svc *service.Service, dbPath, query string, topK int, textOnly, imagesOnly bool, strategy string) error {
	contentType := ""
	if textOnly && imagesOnly {
		return fmt.Errorf("cannot use both --text-only and --images-only")
	} else if textOnly {
		contentType = "text"
	} else if imagesOnly {
		contentType = "image"
	}

	results, err := svc.SearchByDBPath(dbPath, query, topK, contentType, strategy)
	if err != nil {
		return fmt.Errorf("search failed: %w", err)
	}
	if len(results) == 0 {
		fmt.Println("No results found.")
		return nil
	}

	for i, r := range results {
		printResult(i+1, r)
	}
	return nil
}

func printResult(num int, r engine.SearchResult) {
	ct := "text"
	if v, ok := r.TextMetadata["content_type"].(string); ok && v != "" {
		ct = v
	}

	fmt.Printf("Result %d:\n", num)
	if ct == "image" {
		fmt.Println("  🖼️  Type: Image")
	} else {
		fmt.Println("  📄 Type: Text")
	}
	fmt.Printf("  📄 File: %s\n", r.FileName)

	// Page numbers
	if pn, ok := r.TextMetadata["page_numbers"]; ok {
		var pages []string
		switch v := pn.(type) {
		case []int:
			for _, p := range v {
				pages = append(pages, fmt.Sprintf("%d", p))
			}
		case []interface{}:
			for _, p := range v {
				pages = append(pages, fmt.Sprintf("%v", p))
			}
		}
		if len(pages) > 0 {
			fmt.Printf("  📖 Page(s): %s\n", strings.Join(pages, ", "))
		}
	}

	// Chunk info
	if idx, ok := asInt(r.TextMetadata["chunk_index"]); ok {
		if total, ok := asInt(r.TextMetadata["total_chunks"]); ok {
			fmt.Printf("  🧩 Chunk: %d of %d\n", idx+1, total)
		}
	}

	fmt.Printf("  📁 Path: %s\n", r.FilePath)
	fmt.Printf("  📏 Distance: %.4f\n", r.Distance)

	if ct == "image" {
		if has, ok := r.TextMetadata["has_image"].(bool); ok && has {
			fmt.Println("  🖼️  Image content (base64 data available)")
		}
	} else {
		fmt.Printf("  📝 Preview: %s\n", truncate(r.Content, 500))
	}
	fmt.Println()
}

func asInt(v interface{}) (int, bool) {
	switch n := v.(type) {
	case int:
		return n, true
	case float64:
		return int(n), true
	}
	return 0, false
}

func truncate(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max] + "..."
}
