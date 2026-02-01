package cli

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/karim-daw/qwelli/internal/config"
	"github.com/karim-daw/qwelli/internal/engine"
	"github.com/karim-daw/qwelli/internal/search"
	"github.com/karim-daw/qwelli/internal/voyage"
)

// runSearchShell is a wrapper for runSearch that handles shell-specific setup
func runSearchShell(query, indexPath string, topK int, textOnly, imagesOnly bool, strategy string, enableRerank bool, verbose bool) error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}

	// Get absolute path for the index folder
	absPath, err := filepath.Abs(indexPath)
	if err != nil {
		return fmt.Errorf("invalid index path: %w", err)
	}

	// With PostgreSQL, we use the folder path directly to filter results
	return searchResults(query, absPath, topK, textOnly, imagesOnly, strategy, cfg, enableRerank, verbose)
}

// searchResults performs the actual search and displays results (shared between CLI and shell)
func searchResults(query, folderPath string, topK int, textOnly, imagesOnly bool, strategy string, cfg *config.Config, enableRerank bool, verbose bool) error {
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
		EmbeddingEndpoint: cfg.VoyageEmbeddingEndpoint,
		RerankModel:       cfg.VoyageRerankModel,
		RerankEndpoint:    cfg.VoyageRerankEndpoint,
	})
	if err != nil {
		return fmt.Errorf("failed to create voyage client: %w", err)
	}

	eng := engine.NewEngine(voyageClient, true) // Enable multimodal

	// Use DATABASE_URL from config as the dbPath (compatibility)
	// TODO: Add folder path filtering to SearchWithStrategy or at DB level
	// For now, search returns results from ALL indexed folders
	results, err := eng.SearchWithStrategy(query, cfg.DatabaseURL, topK, contentType, strategy)
	if err != nil {
		return fmt.Errorf("search failed: %w", err)
	}

	if len(results) == 0 {
		fmt.Println("No results found.")
		return nil
	}

	// Apply reranking if enabled
	var wasReranked bool
	results, wasReranked = search.ApplyReranking(
		context.Background(),
		voyageClient,
		query,
		results,
		search.RerankOptions{
			Enabled: enableRerank,
			Verbose: verbose,
		},
	)

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

		// Show relevance score if reranked, otherwise show distance
		if wasReranked && result.Distance < 0 {
			relevance := -result.Distance
			fmt.Printf("  ⭐ Relevance: %.4f (reranked)\n", relevance)
		} else {
			fmt.Printf("  📏 Distance: %.4f\n", result.Distance)
		}

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

	// Show reranking summary if enabled and not in verbose mode
	if wasReranked && !verbose {
		fmt.Println("💡 Results were reranked for better relevance")
	}

	return nil
}

func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}
