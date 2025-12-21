package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/karim-daw/qwelli/internal/config"
	"github.com/karim-daw/qwelli/internal/engine"
	"github.com/spf13/cobra"
)

func NewSearchCmd() *cobra.Command {
	var indexPath string
	var topK int
	var textOnly bool
	var imagesOnly bool

	cmd := &cobra.Command{
		Use:   "search <query>",
		Short: "Search indexed files",
		Long:  "Perform semantic search across indexed files",
		Args:  cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			// Clean query: remove any flag-like strings that might have been included
			queryParts := []string{}
			for _, arg := range args {
				// Skip flag-like strings (they should have been parsed by Cobra already)
				if strings.HasPrefix(arg, "--") {
					continue
				}
				queryParts = append(queryParts, arg)
			}
			query := strings.Join(queryParts, " ")
			return runSearch(query, indexPath, topK, textOnly, imagesOnly)
		},
	}

	cmd.Flags().StringVarP(&indexPath, "index", "i", "", "Path to indexed folder (required)")
	cmd.Flags().IntVarP(&topK, "top", "t", 5, "Number of results to return")
	cmd.Flags().BoolVar(&textOnly, "text-only", false, "Search only text chunks")
	cmd.Flags().BoolVar(&imagesOnly, "images-only", false, "Search only image chunks")
	cmd.MarkFlagRequired("index")

	return cmd
}

func runSearch(query, indexPath string, topK int, textOnly, imagesOnly bool) error {
	absPath, err := filepath.Abs(indexPath)
	if err != nil {
		return fmt.Errorf("invalid index path: %w", err)
	}

	cfg, err := config.Load()
	if err != nil {
		return err
	}

	dbPath := filepath.Join(cfg.IndexDir, generateDBName(absPath))

	// Determine content type filter
	contentType := ""
	if textOnly && imagesOnly {
		return fmt.Errorf("cannot use both --text-only and --images-only")
	} else if textOnly {
		contentType = "text"
	} else if imagesOnly {
		contentType = "image"
	}

	fmt.Printf("🔍 Searching for: %s", query)
	if contentType != "" {
		fmt.Printf(" (filter: %s only)", contentType)
	}
	fmt.Printf("\n\n")

	// Use Voyage provider
	eng := engine.NewEngineWithProvider(cfg.APIKey, cfg.Model, cfg.Endpoint, "voyage", false)

	// check if the index exists
	if _, err := os.Stat(dbPath); os.IsNotExist(err) {
		return fmt.Errorf("index not found: %s", dbPath)
	}

	var results []engine.SearchResult
	if contentType != "" {
		results, err = eng.SearchWithFilter(query, dbPath, topK, contentType)
	} else {
		results, err = eng.Search(query, dbPath, topK)
	}
	if err != nil {
		return fmt.Errorf("search failed: %w", err)
	}

	if len(results) == 0 {
		fmt.Println("No results found.")
		return nil
	}

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
				// Note: ImageData would need to be passed through SearchResult
				// For now, just indicate it's an image
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
