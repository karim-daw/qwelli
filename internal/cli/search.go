package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/karim-daw/qwelli/internal/service"
	"github.com/spf13/cobra"
)

func NewSearchCmd() *cobra.Command {
	var indexPath, strategy string
	var topK int
	var textOnly, imagesOnly bool

	cmd := &cobra.Command{
		Use:   "search <query>",
		Short: "Search indexed files",
		Long:  "Supports semantic, keyword, and hybrid strategies.",
		Args:  cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			var parts []string
			for _, a := range args {
				if !strings.HasPrefix(a, "--") {
					parts = append(parts, a)
				}
			}
			return runSearch(strings.Join(parts, " "), indexPath, topK, textOnly, imagesOnly, strategy)
		},
	}
	cmd.Flags().StringVarP(&indexPath, "index", "i", "", "Path to indexed folder (required)")
	cmd.Flags().IntVarP(&topK, "top", "t", 5, "Number of results")
	cmd.Flags().BoolVar(&textOnly, "text-only", false, "Text chunks only")
	cmd.Flags().BoolVar(&imagesOnly, "images-only", false, "Image chunks only")
	cmd.Flags().StringVar(&strategy, "strategy", "semantic", "semantic, keyword, or hybrid")
	cmd.MarkFlagRequired("index")
	return cmd
}

func runSearch(query, indexPath string, topK int, textOnly, imagesOnly bool, strategy string) error {
	return executeSearch(query, indexPath, topK, textOnly, imagesOnly, strategy, true)
}

func executeSearch(query, indexPath string, topK int, textOnly, imagesOnly bool, strategy string, printHeader bool) error {
	absPath, err := filepath.Abs(indexPath)
	if err != nil {
		return fmt.Errorf("invalid path: %w", err)
	}

	svc, err := service.Load()
	if err != nil {
		return err
	}

	dbPath, err := svc.GenerateDBPath(absPath)
	if err != nil {
		return err
	}
	if _, err := os.Stat(dbPath); os.IsNotExist(err) {
		return fmt.Errorf("index not found: %s", dbPath)
	}

	if printHeader {
		fmt.Printf("🔍 Searching: %s", query)
		if strategy != "semantic" {
			fmt.Printf(" [%s]", strategy)
		}
		fmt.Println()
	}

	return displaySearchResults(svc, dbPath, query, topK, textOnly, imagesOnly, strategy)
}
