package server

import (
	"context"
	"fmt"
	"io/fs"
	"log"
	"net"
	"net/http"
	"time"
)

// SetupServer is a minimal HTTP server for first-run setup.
// It serves the web UI and setup API only (no indexes, search, etc).
type SetupServer struct {
	port         int
	listener     net.Listener
	httpServer   *http.Server
	restartAfter bool
}

// NewSetupServer creates a setup-mode server.
func NewSetupServer(port int) *SetupServer {
	return &SetupServer{port: port}
}

// Listen binds to the preferred port (or falls back to an available one).
// It stores the listener and updates the port. Call this before Start().
func (ss *SetupServer) Listen(portExplicit bool) (int, error) {
	ln, actualPort, err := ResolveListener(ss.port, portExplicit)
	if err != nil {
		return 0, err
	}
	ss.listener = ln
	ss.port = actualPort
	return actualPort, nil
}

// Start runs the setup server. Returns when the server shuts down.
func (ss *SetupServer) Start() error {
	mux := http.NewServeMux()

	mux.HandleFunc("/api/setup/status", handleSetupStatus)
	mux.HandleFunc("/api/setup", ss.handleSetupSave)
	mux.HandleFunc("/api/shutdown", ss.handleShutdown)

	distFS, err := fs.Sub(webFS, "web/dist")
	if err != nil {
		return fmt.Errorf("load web assets: %w", err)
	}
	mux.Handle("/", http.FileServer(http.FS(distFS)))

	addr := fmt.Sprintf(":%d", ss.port)
	log.Printf("🚀 Setup server starting on http://localhost%s", addr)

	ss.httpServer = &http.Server{
		Addr:    addr,
		Handler: setupCORS(mux),
	}
	if ss.listener != nil {
		return ss.httpServer.Serve(ss.listener)
	}
	return ss.httpServer.ListenAndServe()
}

// requestRestart is called after config is saved; shuts down so caller can start normal server
func (ss *SetupServer) requestRestart() {
	ss.restartAfter = true
	if ss.httpServer != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = ss.httpServer.Shutdown(ctx)
	}
}

// RestartRequested returns true if the server exited because config was saved (restart), not quit
func (ss *SetupServer) RestartRequested() bool {
	return ss.restartAfter
}

func (ss *SetupServer) handleShutdown(w http.ResponseWriter, r *http.Request) {
	if !requireMethod(w, r, http.MethodPost) {
		return
	}
	if ss.httpServer == nil {
		jsonError(w, http.StatusInternalServerError, "server not ready")
		return
	}
	jsonOK(w, map[string]string{"status": "shutting down"})
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = ss.httpServer.Shutdown(ctx)
	}()
}

func setupCORS(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}
		next.ServeHTTP(w, r)
	})
}
