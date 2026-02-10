package server

import (
	"context"
	"embed"
	"fmt"
	"io/fs"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/karim-daw/qwelli/internal/service"
)

//go:embed web/dist
var webFS embed.FS

type Server struct {
	service *service.Service
	port    int

	// HTTP server for graceful shutdown
	httpServer *http.Server

	// SSE progress broadcasting
	progressClients map[string][]chan string
	progressMutex   sync.RWMutex

	// Search cache
	searchCache map[string]*searchCacheEntry
	cacheMutex  sync.RWMutex
	cacheTTL    time.Duration

	// Background indexing cancellation
	cancelFuncs map[string]context.CancelFunc
	cancelMutex sync.RWMutex
}

// NewServer creates a server with an injected service (and port).
// The caller is responsible for loading config and creating the service.
func NewServer(svc *service.Service, port int) *Server {
	s := &Server{
		service:         svc,
		port:            port,
		progressClients: make(map[string][]chan string),
		searchCache:     make(map[string]*searchCacheEntry),
		cacheTTL:        5 * time.Minute,
		cancelFuncs:     make(map[string]context.CancelFunc),
	}
	go s.cleanupExpiredCache()
	return s
}

func (s *Server) Start() error {
	mux := http.NewServeMux()

	// API routes
	mux.HandleFunc("/api/indexes", s.handleListIndexes)
	mux.HandleFunc("/api/search", s.handleSearch)
	mux.HandleFunc("/api/file", s.handleFileAccess)
	mux.HandleFunc("/api/index/create", s.handleCreateIndex)
	mux.HandleFunc("/api/index/status", s.handleIndexStatus)
	mux.HandleFunc("/api/index/update", s.handleUpdateIndex)
	mux.HandleFunc("/api/index/progress", s.handleIndexProgress)
	mux.HandleFunc("/api/index/cancel", s.handleCancelIndex)
	mux.HandleFunc("/api/index/delete", s.handleDeleteIndex)
	mux.HandleFunc("/api/open-folder", s.handleOpenFolder)
	mux.HandleFunc("/api/open-file-location", s.handleOpenFileLocation)
	mux.HandleFunc("/api/terminal/stream", s.handleTerminalStream)
	mux.HandleFunc("/api/shutdown", s.handleShutdown)
	mux.HandleFunc("/api/setup/status", handleSetupStatus)

	// Static web assets
	distFS, err := fs.Sub(webFS, "web/dist")
	if err != nil {
		return fmt.Errorf("load web assets: %w", err)
	}
	mux.Handle("/", http.FileServer(http.FS(distFS)))

	cfg := s.service.Config()
	addr := fmt.Sprintf(":%d", s.port)
	
	// Broadcast server startup messages to terminal
	BroadcastTerminalOutput(fmt.Sprintf(`{"type":"log","level":"info","message":"🚀 Server starting on http://localhost%s"}`, addr))
	BroadcastTerminalOutput(fmt.Sprintf(`{"type":"log","level":"info","message":"📂 Index directory: %s"}`, cfg.IndexDir))
	BroadcastTerminalOutput(fmt.Sprintf(`{"type":"log","level":"info","message":"🤖 Model: %s"}`, cfg.Model))
	if cfg.EnableReranker {
		BroadcastTerminalOutput(`{"type":"log","level":"info","message":"🔄 Reranker: enabled"}`)
	}
	
	log.Printf("🚀 Server starting on http://localhost%s", addr)
	log.Printf("📂 Index directory: %s", cfg.IndexDir)
	log.Printf("🤖 Model: %s", cfg.Model)
	if cfg.EnableReranker {
		log.Printf("🔄 Reranker: enabled")
	}

	s.httpServer = &http.Server{
		Addr:    addr,
		Handler: s.corsMiddleware(mux),
	}
	return s.httpServer.ListenAndServe()
}

func (s *Server) corsMiddleware(next http.Handler) http.Handler {
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
