package main

import (
	"fmt"
	"os"

	"github.com/karim-daw/qwelli/internal/cli"
	"github.com/spf13/cobra"
)

var version = "0.1.0"

func main() {
	// If no arguments provided (double-click scenario), default to serve
	if len(os.Args) == 1 {
		if err := cli.RunServeDefault(); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		return
	}

	rootCmd := &cobra.Command{
		Use:   "qwelli",
		Short: "Qwelli - Semantic search for your local files",
		Long: `Qwelli indexes your local folders and provides semantic search capabilities.

Using vector embeddings, Qwelli understands the meaning of your files
and helps you find what you're looking for with natural language queries.`,
		Version: version,
	}

	// Add commands
	rootCmd.AddCommand(cli.NewShellCmd())
	rootCmd.AddCommand(cli.NewServeCmd())
	rootCmd.AddCommand(cli.NewInitCmd())
	rootCmd.AddCommand(cli.NewIndexCmd())
	rootCmd.AddCommand(cli.NewSearchCmd())
	rootCmd.AddCommand(cli.NewListCmd())
	rootCmd.AddCommand(cli.NewStatusCmd())
	rootCmd.AddCommand(cli.NewDeleteCmd())

	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
