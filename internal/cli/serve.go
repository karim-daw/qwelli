package cli

import (
	"fmt"
	"os/exec"
	"runtime"
	"time"

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
			return runServe(port, false)
		},
	}

	cmd.Flags().IntVarP(&port, "port", "p", 8080, "Port to run the server on")

	return cmd
}

// RunServeDefault runs serve with default settings and auto-opens browser
func RunServeDefault() error {
	return runServe(8080, true)
}

func runServe(port int, openBrowser bool) error {
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("failed to load config: %w\nRun 'qwelli init' first", err)
	}

	srv := server.NewServer(cfg, port)

	// If we should open browser, start server in goroutine and open browser
	if openBrowser {
		// Start server in goroutine
		serverErr := make(chan error, 1)
		go func() {
			serverErr <- srv.Start()
		}()

		// Wait a bit for server to start, then open browser
		time.Sleep(1 * time.Second)
		openBrowserURL(fmt.Sprintf("http://localhost:%d", port))

		// Wait for server to finish (or error)
		return <-serverErr
	}

	return srv.Start()
}

// openBrowserURL opens the specified URL in the default browser
func openBrowserURL(url string) {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "windows":
		cmd = exec.Command("cmd", "/c", "start", url)
	case "darwin":
		cmd = exec.Command("open", url)
	case "linux":
		cmd = exec.Command("xdg-open", url)
	default:
		return // Unsupported OS
	}

	// Start the command (don't wait for it to finish)
	cmd.Start()
}
