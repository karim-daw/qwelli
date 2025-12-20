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

	cmd := &cobra.Command{
		Use:   "search <query>",
		Short: "Search indexed files",
		Long:  "Perform semantic search across indexed files",
		Args:  cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			query := strings.Join(args, " ")
			return runSearch(query, indexPath, topK)
		},
	}

	cmd.Flags().StringVarP(&indexPath, "index", "i", "", "Path to indexed folder (required)")
	cmd.Flags().IntVarP(&topK, "top", "t", 5, "Number of results to return")
	cmd.MarkFlagRequired("index")

	return cmd
}

func runSearch(query, indexPath string, topK int) error {
	absPath, err := filepath.Abs(indexPath)
	if err != nil {
		return fmt.Errorf("invalid index path: %w", err)
	}

	cfg, err := config.Load()
	if err != nil {
		return err
	}

	dbPath := filepath.Join(cfg.IndexDir, generateDBName(absPath))

	fmt.Printf("🔍 Searching for: %s\n\n", query)

	eng := engine.NewEngine(cfg.APIKey, cfg.Model, cfg.Endpoint)

	// check if the index exists
	if _, err := os.Stat(dbPath); os.IsNotExist(err) {
		return fmt.Errorf("index not found: %s", dbPath)
	}

	results, err := eng.Search(query, dbPath, topK)
	if err != nil {
		return fmt.Errorf("search failed: %w", err)
	}

	if len(results) == 0 {
		fmt.Println("No results found.")
		return nil
	}

	for i, result := range results {
		fmt.Printf("Result %d:\n", i+1)
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
		fmt.Printf("  📝 Preview: %s\n", truncate(result.Content, 500))
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
