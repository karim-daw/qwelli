package cli

import (
	"fmt"
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
	var rerank bool
	var noRerank bool
	var verbose bool

	cmd := &cobra.Command{
		Use:   "search <query>",
		Short: "Search indexed files",
		Long: `Perform search across indexed files.

Supports semantic, keyword (FTS), and hybrid strategies.
Results are automatically reranked for better relevance (disable with --no-rerank).

Examples:
  qwelli search "machine learning" --index ~/docs
  qwelli search "API docs" --strategy hybrid --rerank
  qwelli search "query" --no-rerank --verbose
`,
		Args: cobra.MinimumNArgs(1),
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

			// Determine rerank preference (flag takes precedence over config)
			var enableRerank *bool
			if cmd.Flags().Changed("rerank") {
				t := true
				enableRerank = &t
			} else if cmd.Flags().Changed("no-rerank") {
				f := false
				enableRerank = &f
			}

			return runSearch(query, indexPath, topK, textOnly, imagesOnly, strategy, enableRerank, verbose)
		},
	}

	cmd.Flags().StringVarP(&indexPath, "index", "i", "", "Path to indexed folder (required)")
	cmd.Flags().IntVarP(&topK, "top", "t", 5, "Number of results to return")
	cmd.Flags().BoolVar(&textOnly, "text-only", false, "Search only text chunks")
	cmd.Flags().BoolVar(&imagesOnly, "images-only", false, "Search only image chunks")
	cmd.Flags().StringVar(&strategy, "strategy", "semantic", "Search strategy: semantic, keyword, or hybrid")
	cmd.Flags().BoolVar(&rerank, "rerank", false, "Force enable reranking (overrides ENABLE_RERANKER)")
	cmd.Flags().BoolVar(&noRerank, "no-rerank", false, "Disable reranking for this search")
	cmd.Flags().BoolVarP(&verbose, "verbose", "v", false, "Show detailed reranking logs")
	cmd.MarkFlagRequired("index")

	return cmd
}

func runSearch(query, indexPath string, topK int, textOnly, imagesOnly bool, strategy string, enableRerank *bool, verbose bool) error {
	absPath, err := filepath.Abs(indexPath)
	if err != nil {
		return fmt.Errorf("invalid index path: %w", err)
	}

	cfg, err := config.Load()
	if err != nil {
		return err
	}

	// Determine final rerank setting (flag > config default)
	shouldRerank := cfg.EnableReranker
	if enableRerank != nil {
		shouldRerank = *enableRerank
	}

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
	fmt.Printf("\n")

	if verbose {
		fmt.Printf("🔄 Reranker: %s\n", map[bool]string{true: "enabled", false: "disabled"}[shouldRerank])
	}
	fmt.Printf("\n")

	// Use shared searchResults function with the folder path
	return searchResults(query, absPath, topK, textOnly, imagesOnly, strategy, cfg, shouldRerank, verbose)
}
