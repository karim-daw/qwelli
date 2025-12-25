package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/karim-daw/qwelli/internal/config"
	"github.com/spf13/cobra"
)

func NewSearchCmd() *cobra.Command {
	var indexPath string
	var topK int
	var textOnly bool
	var imagesOnly bool
	var strategy string

	cmd := &cobra.Command{
		Use:   "search <query>",
		Short: "Search indexed files",
		Long:  "Perform search across indexed files. Supports semantic, keyword (FTS), and hybrid strategies.",
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
			return runSearch(query, indexPath, topK, textOnly, imagesOnly, strategy)
		},
	}

	cmd.Flags().StringVarP(&indexPath, "index", "i", "", "Path to indexed folder (required)")
	cmd.Flags().IntVarP(&topK, "top", "t", 5, "Number of results to return")
	cmd.Flags().BoolVar(&textOnly, "text-only", false, "Search only text chunks")
	cmd.Flags().BoolVar(&imagesOnly, "images-only", false, "Search only image chunks")
	cmd.Flags().StringVar(&strategy, "strategy", "semantic", "Search strategy: semantic, keyword, or hybrid")
	cmd.MarkFlagRequired("index")

	return cmd
}

func runSearch(query, indexPath string, topK int, textOnly, imagesOnly bool, strategy string) error {
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
	if strategy != "semantic" {
		fmt.Printf(" [strategy: %s]", strategy)
	}
	fmt.Printf("\n\n")

	// check if the index exists
	if _, err := os.Stat(dbPath); os.IsNotExist(err) {
		return fmt.Errorf("index not found: %s", dbPath)
	}

	// Use shared searchResults function
	return searchResults(query, dbPath, topK, textOnly, imagesOnly, strategy, cfg)
}
