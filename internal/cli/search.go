package cli

import (
	"fmt"
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
	// Resolve to absolute path
	absPath, err := filepath.Abs(indexPath)
	if err != nil {
		return fmt.Errorf("invalid index path: %w", err)
	}

	// Load config
	cfg, err := config.Load()
	if err != nil {
		return err
	}

	// Find database file
	dbName := generateDBName(absPath)
	dbPath := filepath.Join(cfg.IndexDir, dbName)

	// Check if database exists
	if _, err := filepath.Abs(dbPath); err != nil {
		return fmt.Errorf("index not found for %s. Run 'qwelli index %s' first", absPath, absPath)
	}

	fmt.Printf("🔍 Searching for: \"%s\"\n\n", query)

	// Create engine
	eng := engine.NewEngine(cfg.APIKey, cfg.Model, cfg.Endpoint)

	// Perform search
	results, err := eng.Search(query, dbPath, topK)
	if err != nil {
		return fmt.Errorf("search failed: %w", err)
	}

	if len(results) == 0 {
		fmt.Println("No results found.")
		return nil
	}

	// Display results
	for i, result := range results {
		fmt.Printf("Result %d:\n", i+1)
		fmt.Printf("  📄 File: %s\n", result.FileName)
		fmt.Printf("  📁 Path: %s\n", result.FilePath)
		fmt.Printf("  📏 Distance: %.4f\n", result.Distance)

		// Show content preview
		preview := truncate(result.Content, 150)
		fmt.Printf("  📝 Preview: %s\n", preview)
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
