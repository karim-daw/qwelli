package cli

import (
	"fmt"

	"github.com/karim-daw/qwelli/internal/config"
	"github.com/karim-daw/qwelli/internal/server"
	"github.com/spf13/cobra"
)

func NewServeCmd() *cobra.Command {
	var port int

	cmd := &cobra.Command{
		Use:   "serve",
		Short: "Start web UI server",
		Long:  "Start the web interface for Qwelli on localhost",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runServe(port)
		},
	}

	cmd.Flags().IntVarP(&port, "port", "p", 8080, "Port to run the server on")

	return cmd
}

func runServe(port int) error {
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("failed to load config: %w\nRun 'qwelli init' first", err)
	}

	srv := server.NewServer(cfg, port)
	return srv.Start()
}
