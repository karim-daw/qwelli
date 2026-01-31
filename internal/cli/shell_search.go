package cli

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/karim-daw/qwelli/internal/config"
	"github.com/karim-daw/qwelli/internal/engine"
	"github.com/karim-daw/qwelli/internal/voyage"
)

// runSearchShell is a wrapper for runSearch that handles shell-specific setup
func runSearchShell(query, indexPath string, topK int, textOnly, imagesOnly bool, strategy string) error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}

	dbPath := filepath.Join(cfg.DatabaseURL, generateDBName(indexPath))
	return searchResults(query, dbPath, topK, textOnly, imagesOnly, strategy, cfg)
}

// searchResults performs the actual search and displays results (shared between CLI and shell)
func searchResults(query, dbPath string, topK int, textOnly, imagesOnly bool, strategy string, cfg *config.Config) error {
	// Determine content type filter
	contentType := ""
	if textOnly && imagesOnly {
		return fmt.Errorf("cannot use both --text-only and --images-only")
	} else if textOnly {
		contentType = "text"
	} else if imagesOnly {
		contentType = "image"
	}

	// Create voyage client and engine
	voyageClient, err := voyage.NewClient(voyage.ClientConfig{
		APIKey:            cfg.VoyageAPIKey,
		EmbeddingModel:    cfg.VoyageModel,
		EmbeddingEndpoint: cfg.VoyageModel,
	})
	if err != nil {
		return fmt.Errorf("failed to create voyage client: %w", err)
	}

	eng := engine.NewEngine(voyageClient, true)

	// Use SearchWithStrategy to support different search methods
	results, err := eng.SearchWithStrategy(query, dbPath, topK, contentType, strategy)
	if err != nil {
		return fmt.Errorf("search failed: %w", err)
	}

	if len(results) == 0 {
		fmt.Println("No results found.")
		return nil
	}

	// Display results (same format as search.go)
	for i, result := range results {
		fmt.Printf("Result %d:\n", i+1)

		// Display content type
		contentType := "text"
		if ct, ok := result.TextMetadata["content_type"].(string); ok && ct != "" {
			contentType = ct
		}
		if contentType == "image" {
			fmt.Printf("  🖼️  Type: Image\n")
		} else {
			fmt.Printf("  📄 Type: Text\n")
		}

		fmt.Printf("  📄 File: %s\n", result.FileName)

		// Display page numbers if available (PDFs)
		if pageNumbers, ok := result.TextMetadata["page_numbers"]; ok {
			var pages []string
			switch v := pageNumbers.(type) {
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

		// Display chunk info if available
		if chunkIdx, ok := result.TextMetadata["chunk_index"]; ok {
			var idx int
			switch v := chunkIdx.(type) {
			case int:
				idx = v
			case float64:
				idx = int(v)
			}
			if totalChunks, ok := result.TextMetadata["total_chunks"]; ok {
				var total int
				switch v := totalChunks.(type) {
				case int:
					total = v
				case float64:
					total = int(v)
				}
				fmt.Printf("  🧩 Chunk: %d of %d\n", idx+1, total)
			}
		}

		fmt.Printf("  📁 Path: %s\n", result.FilePath)
		fmt.Printf("  📏 Distance: %.4f\n", result.Distance)

		// For images, try to save preview
		if contentType == "image" {
			if hasImage, ok := result.TextMetadata["has_image"].(bool); ok && hasImage {
				fmt.Printf("  🖼️  Image content (base64 data available)\n")
			}
		} else {
			fmt.Printf("  📝 Preview: %s\n", truncate(result.Content, 500))
		}
		fmt.Println()
	}

	return nil
}

func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}
