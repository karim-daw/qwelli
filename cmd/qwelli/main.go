package main

import (
	"fmt"
	"os"

	"github.com/karim-daw/qwelli/internal/cli"
	"github.com/spf13/cobra"
)

var version = "0.1.0"

func main() {
	// Double-click / no-args: default to serve
	if len(os.Args) == 1 {
		if err := cli.RunServeDefault(); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		return
	}

	rootCmd := &cobra.Command{
		Use:     "qwelli",
		Short:   "Qwelli — Semantic search for your local files",
		Version: version,
	}

	rootCmd.AddCommand(
		cli.NewShellCmd(),
		cli.NewServeCmd(),
		cli.NewInitCmd(),
		cli.NewIndexCmd(),
		cli.NewSearchCmd(),
		cli.NewListCmd(),
		cli.NewStatusCmd(),
		cli.NewDeleteCmd(),
		cli.NewChatCmd(),
	)

	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
