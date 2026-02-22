package cli

import (
	"errors"
	"fmt"
	"log"
	"net/http"
	"os/exec"
	"runtime"

	"github.com/karim-daw/qwelli/internal/config"
	"github.com/karim-daw/qwelli/internal/server"
	"github.com/karim-daw/qwelli/internal/service"
	"github.com/spf13/cobra"
)

func NewServeCmd() *cobra.Command {
	var port int

	cmd := &cobra.Command{
		Use:   "serve",
		Short: "Start web UI server",
		Long:  "Start the web interface for Qwelli on localhost",
		RunE: func(cmd *cobra.Command, args []string) error {
			portExplicit := cmd.Flags().Changed("port")
			return runServe(port, false, portExplicit)
		},
	}

	cmd.Flags().IntVarP(&port, "port", "p", 0, "Port to run the server on (0 = random available port)")

	return cmd
}

// RunServeDefault runs serve with default settings and auto-opens browser
func RunServeDefault() error {
	return runServe(0, true, false)
}

func runServe(port int, openBrowser bool, portExplicit bool) error {
	for {
		if !config.Exists() {
			// First-run: start setup server
			ss := server.NewSetupServer(port)
			actualPort, err := ss.Listen(portExplicit)
			if err != nil {
				return err
			}
			log.Printf("Starting on port %d", actualPort)

			if openBrowser {
				serverErr := make(chan error, 1)
				go func() {
					serverErr <- ss.Start()
				}()
				openBrowserURL(fmt.Sprintf("http://localhost:%d", actualPort))
				if err := <-serverErr; err != nil && !errors.Is(err, http.ErrServerClosed) {
					return err
				}
				if !ss.RestartRequested() {
					return nil
				}
				// Carry forward the actual port for the main server
				port = actualPort
				continue
			}
			return ss.Start()
		}

		svc, err := service.Load()
		if err != nil {
			return fmt.Errorf("failed to load config: %w\nRun 'qwelli init' first", err)
		}

		srv := server.NewServer(svc, port)
		actualPort, err := srv.Listen(portExplicit)
		if err != nil {
			return err
		}
		log.Printf("Starting on port %d", actualPort)

		if openBrowser {
			serverErr := make(chan error, 1)
			go func() {
				serverErr <- srv.Start()
			}()
			openBrowserURL(fmt.Sprintf("http://localhost:%d", actualPort))
			return <-serverErr
		}

		return srv.Start()
	}
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
