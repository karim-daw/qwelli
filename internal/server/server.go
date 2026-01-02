package server

import (
	"context"
	"embed"
	"encoding/json"
	"fmt"
	"io/fs"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/karim-daw/qwelli/internal/config"
	"github.com/karim-daw/qwelli/internal/db"
	"github.com/karim-daw/qwelli/internal/engine"
	"github.com/karim-daw/qwelli/internal/engine/fileprocessor"
)

//go:embed web/dist
var webFS embed.FS

// SearchCacheEntry represents a cached search result
type SearchCacheEntry struct {
	Results   []SearchResultItem
	Timestamp time.Time
}

type Server struct {
	config          *config.Config
	engine          *engine.Engine
	port            int
	progressClients map[string][]chan string // indexPath -> clients listening for progress
	progressMutex   sync.RWMutex
	searchCache     map[string]*SearchCacheEntry // cacheKey -> cached results
	cacheMutex      sync.RWMutex
	cacheTTL        time.Duration                 // Time-to-live for cache entries
	cancelFuncs     map[string]context.CancelFunc // indexPath -> cancel function for active indexing
	cancelMutex     sync.RWMutex
}

func NewServer(cfg *config.Config, port int) *Server {
	eng := engine.NewEngine(cfg.APIKey, cfg.Model, cfg.Endpoint, cfg.EnableMultimodal)
	s := &Server{
		config:          cfg,
		engine:          eng,
		port:            port,
		progressClients: make(map[string][]chan string),
		searchCache:     make(map[string]*SearchCacheEntry),
		cacheTTL:        5 * time.Minute, // Cache results for 5 minutes
		cancelFuncs:     make(map[string]context.CancelFunc),
	}

	// Start cache cleanup goroutine
	go s.cleanupExpiredCache()

	return s
}

func (s *Server) Start() error {
	mux := http.NewServeMux()

	// API endpoints
	mux.HandleFunc("/api/indexes", s.handleListIndexes)
	mux.HandleFunc("/api/search", s.handleSearch)
	mux.HandleFunc("/api/file", s.handleFileAccess)
	mux.HandleFunc("/api/index/create", s.handleCreateIndex)
	mux.HandleFunc("/api/index/status", s.handleIndexStatus)
	mux.HandleFunc("/api/index/update", s.handleUpdateIndex)
	mux.HandleFunc("/api/index/progress", s.handleIndexProgress)
	mux.HandleFunc("/api/index/cancel", s.handleCancelIndex)
	mux.HandleFunc("/api/open-folder", s.handleOpenFolder)
	mux.HandleFunc("/api/open-file-location", s.handleOpenFileLocation)
	mux.HandleFunc("/api/index/delete", s.handleDeleteIndex)

	// Serve static files from embedded filesystem
	distFS, err := fs.Sub(webFS, "web/dist")
	if err != nil {
		return fmt.Errorf("failed to load web assets: %w", err)
	}

	fileServer := http.FileServer(http.FS(distFS))
	mux.Handle("/", fileServer)

	addr := fmt.Sprintf(":%d", s.port)
	log.Printf("🚀 Server starting on http://localhost%s\n", addr)
	log.Printf("📂 Index directory: %s\n", s.config.IndexDir)
	log.Printf("🤖 Model: %s\n", s.config.Model)

	return http.ListenAndServe(addr, s.corsMiddleware(mux))
}

// corsMiddleware adds CORS headers for development
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

// cleanupExpiredCache periodically removes expired cache entries
func (s *Server) cleanupExpiredCache() {
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		s.cacheMutex.Lock()
		now := time.Now()
		for key, entry := range s.searchCache {
			if now.Sub(entry.Timestamp) > s.cacheTTL {
				delete(s.searchCache, key)
			}
		}
		s.cacheMutex.Unlock()
	}
}

// generateCacheKey creates a unique key for cache lookup
func (s *Server) generateCacheKey(query, indexPath, strategy, contentType string, topK int) string {
	return fmt.Sprintf("%s|%s|%s|%s|%d", query, indexPath, strategy, contentType, topK)
}

// getCachedResults retrieves cached search results if available and not expired
func (s *Server) getCachedResults(cacheKey string) ([]SearchResultItem, bool) {
	s.cacheMutex.RLock()
	defer s.cacheMutex.RUnlock()

	entry, exists := s.searchCache[cacheKey]
	if !exists {
		return nil, false
	}

	// Check if expired
	if time.Since(entry.Timestamp) > s.cacheTTL {
		return nil, false
	}

	return entry.Results, true
}

// setCachedResults stores search results in cache
func (s *Server) setCachedResults(cacheKey string, results []SearchResultItem) {
	s.cacheMutex.Lock()
	defer s.cacheMutex.Unlock()

	s.searchCache[cacheKey] = &SearchCacheEntry{
		Results:   results,
		Timestamp: time.Now(),
	}
}

type IndexInfo struct {
	Name          string `json:"name"`
	Path          string `json:"path"`
	DocumentCount int    `json:"documentCount"`
	LastModified  string `json:"lastModified"`
}

type ListIndexesResponse struct {
	Indexes []IndexInfo `json:"indexes"`
}

func (s *Server) handleListIndexes(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// List all .db files in index directory
	files, err := filepath.Glob(filepath.Join(s.config.IndexDir, "*.db"))
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to list indexes: %v", err), http.StatusInternalServerError)
		return
	}

	indexes := make([]IndexInfo, 0, len(files))

	for _, dbFile := range files {
		// Get stats
		count, err := s.engine.GetIndexStats(dbFile)
		if err != nil {
			count = 0
		}

		// Get folder path from metadata
		folderPath, err := s.engine.GetFolderPath(dbFile)
		if err != nil || folderPath == "" {
			// Fallback to database name if no metadata
			dbName := filepath.Base(dbFile)
			folderPath = strings.TrimSuffix(dbName, ".db")
		}

		// Get file info
		info, _ := os.Stat(dbFile)
		lastModified := ""
		if info != nil {
			lastModified = info.ModTime().Format("2006-01-02 15:04:05")
		}

		indexes = append(indexes, IndexInfo{
			Name:          filepath.Base(folderPath),
			Path:          folderPath,
			DocumentCount: count,
			LastModified:  lastModified,
		})
	}

	response := ListIndexesResponse{Indexes: indexes}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

type SearchResultItem struct {
	ChunkID      string                 `json:"chunkId"`
	Content      string                 `json:"content"`
	FilePath     string                 `json:"filePath"`
	FileName     string                 `json:"fileName"`
	ContentType  string                 `json:"contentType"`
	Distance     float64                `json:"similarity"`
	PageNumbers  []int                  `json:"pageNumbers"`
	ChunkIndex   int                    `json:"chunkIndex,omitempty"`
	TotalChunks  int                    `json:"totalChunks,omitempty"`
	FileSize     int64                  `json:"fileSize,omitempty"`
	ModifiedDate string                 `json:"modifiedDate,omitempty"`
	ImageData    string                 `json:"imageData,omitempty"` // Base64 encoded image
	TextMetadata map[string]interface{} `json:"metadata,omitempty"`
}

type SearchResponse struct {
	Results []SearchResultItem `json:"results"`
	Query   string             `json:"query"`
}

func (s *Server) handleSearch(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Parse query parameters
	query := r.URL.Query().Get("q")
	indexPath := r.URL.Query().Get("index")
	strategy := r.URL.Query().Get("strategy")
	topKStr := r.URL.Query().Get("top")
	contentType := r.URL.Query().Get("content_type")

	if query == "" {
		http.Error(w, "Query parameter 'q' is required", http.StatusBadRequest)
		return
	}

	if indexPath == "" {
		http.Error(w, "Query parameter 'index' is required", http.StatusBadRequest)
		return
	}

	// Default values
	if strategy == "" {
		strategy = "semantic"
	}

	topK := 5
	if topKStr != "" {
		if k, err := strconv.Atoi(topKStr); err == nil && k > 0 {
			topK = k
		}
	}

	// Get database path
	absPath, err := filepath.Abs(indexPath)
	if err != nil {
		http.Error(w, fmt.Sprintf("Invalid index path: %v", err), http.StatusBadRequest)
		return
	}

	dbPath := filepath.Join(s.config.IndexDir, generateDBName(absPath))

	// Check if index exists
	if _, err := os.Stat(dbPath); os.IsNotExist(err) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(map[string]string{"error": fmt.Sprintf("Index not found: %s", dbPath)})
		return
	}

	// Generate cache key
	cacheKey := s.generateCacheKey(query, indexPath, strategy, contentType, topK)

	// Check cache first
	if cachedResults, found := s.getCachedResults(cacheKey); found {
		log.Printf("🎯 Cache hit for query: %s", query)
		response := SearchResponse{
			Results: cachedResults,
			Query:   query,
		}
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("X-Cache-Status", "HIT")
		json.NewEncoder(w).Encode(response)
		return
	}

	log.Printf("🔍 Cache miss, performing search for: %s", query)

	// Perform search using the engine
	engineResults, err := s.engine.SearchWithStrategy(query, dbPath, topK, contentType, strategy)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": fmt.Sprintf("Search failed: %v", err)})
		return
	}

	// Convert engine.SearchResult to API response format
	results := make([]SearchResultItem, len(engineResults))
	for i, r := range engineResults {
		// Get file info
		var fileSize int64
		var modifiedDate string
		if fileInfo, err := os.Stat(r.FilePath); err == nil {
			fileSize = fileInfo.Size()
			modifiedDate = fileInfo.ModTime().Format("2006-01-02 15:04:05")
		}

		// Convert image data to base64 string if present
		var imageDataStr string
		if len(r.ImageData) > 0 {
			imageDataStr = string(r.ImageData)
		}

		// Get page numbers - prefer direct field, fallback to metadata
		pageNumbers := r.PageNumbers
		if len(pageNumbers) == 0 {
			pageNumbers = getPageNumbers(r.TextMetadata)
		}

		results[i] = SearchResultItem{
			ChunkID:      r.FilePath, // Use filepath as unique ID for now
			Content:      r.Content,
			FilePath:     r.FilePath,
			FileName:     r.FileName,
			ContentType:  getContentType(r.TextMetadata),
			Distance:     r.Distance,
			PageNumbers:  pageNumbers,
			ChunkIndex:   getChunkIndex(r.TextMetadata),
			TotalChunks:  getTotalChunks(r.TextMetadata),
			FileSize:     fileSize,
			ModifiedDate: modifiedDate,
			ImageData:    imageDataStr,
			TextMetadata: r.TextMetadata,
		}
	}

	// Store results in cache
	s.setCachedResults(cacheKey, results)

	response := SearchResponse{
		Results: results,
		Query:   query,
	}

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("X-Cache-Status", "MISS")
	json.NewEncoder(w).Encode(response)
}

// Helper to extract content_type from metadata
func getContentType(metadata map[string]interface{}) string {
	if ct, ok := metadata["content_type"].(string); ok {
		return ct
	}
	return "text"
}

// Helper to extract page_numbers from metadata
func getPageNumbers(metadata map[string]interface{}) []int {
	if metadata == nil {
		return nil
	}

	// Try direct []int first
	if pn, ok := metadata["page_numbers"].([]int); ok {
		return pn
	}

	// Try []interface{} (common in JSON unmarshaling)
	if pnIface, ok := metadata["page_numbers"]; ok {
		if arr, ok := pnIface.([]interface{}); ok {
			result := make([]int, 0, len(arr))
			for _, v := range arr {
				switch val := v.(type) {
				case int:
					result = append(result, val)
				case int64:
					result = append(result, int(val))
				case float64:
					result = append(result, int(val))
				}
			}
			return result
		}
	}

	return nil
}

// Helper to extract chunk_index from metadata
func getChunkIndex(metadata map[string]interface{}) int {
	if idx, ok := metadata["chunk_index"].(float64); ok {
		return int(idx)
	}
	if idx, ok := metadata["chunk_index"].(int); ok {
		return idx
	}
	return 0
}

// Helper to extract total_chunks from metadata
func getTotalChunks(metadata map[string]interface{}) int {
	if total, ok := metadata["total_chunks"].(float64); ok {
		return int(total)
	}
	if total, ok := metadata["total_chunks"].(int); ok {
		return total
	}
	return 0
}

// generateDBName creates a consistent database name from a folder path
func generateDBName(path string) string {
	// Use the same logic as the CLI: just the base folder name
	base := filepath.Base(path)

	// Clean the name to be filesystem-safe
	safe := strings.Map(func(r rune) rune {
		if r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || r >= '0' && r <= '9' || r == '-' || r == '_' {
			return r
		}
		return '_'
	}, base)

	return safe + ".db"
}

// handleFileAccess serves file content (for opening files from search results)
func (s *Server) handleFileAccess(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	filePath := r.URL.Query().Get("path")
	if filePath == "" {
		http.Error(w, "File path required", http.StatusBadRequest)
		return
	}

	// Security: only allow files that were indexed (exist in our database)
	// Check if file exists in any index
	allowed := false
	files, _ := filepath.Glob(filepath.Join(s.config.IndexDir, "*.db"))
	for _, dbFile := range files {
		dim, err := db.GetDimensionFromDB(dbFile)
		if err != nil {
			continue
		}
		projectDB, err := db.OpenProjectDB(dbFile, dim)
		if err != nil {
			continue
		}
		_, err = projectDB.GetFileByPath(filePath)
		projectDB.Close()
		if err == nil {
			allowed = true
			break
		}
	}

	if !allowed {
		http.Error(w, "File not found in index", http.StatusNotFound)
		return
	}

	// Check if file exists
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		http.Error(w, "File not found", http.StatusNotFound)
		return
	}

	// Serve the file
	http.ServeFile(w, r, filePath)
}

// handleCreateIndex handles creating a new index
func (s *Server) handleCreateIndex(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		FolderPath  string `json:"folderPath"`
		ContentType string `json:"contentType,omitempty"` // "both", "text", or "images"
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "Invalid request body"})
		return
	}

	if req.FolderPath == "" {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "Folder path is required"})
		return
	}

	// Check if folder exists
	absPath, err := filepath.Abs(req.FolderPath)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "Invalid folder path"})
		return
	}

	if stat, err := os.Stat(absPath); err != nil || !stat.IsDir() {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "Folder does not exist"})
		return
	}

	// Generate database path
	dbPath := filepath.Join(s.config.IndexDir, generateDBName(absPath))

	// Create cancellable context for this indexing operation
	ctx, cancel := context.WithCancel(context.Background())

	// Store cancel function
	s.cancelMutex.Lock()
	s.cancelFuncs[absPath] = cancel
	s.cancelMutex.Unlock()

	// Start indexing in background (this can take a while)
	go func() {
		defer func() {
			// Clean up cancel function when done
			s.cancelMutex.Lock()
			delete(s.cancelFuncs, absPath)
			s.cancelMutex.Unlock()
		}()

		log.Printf("Starting indexing of %s...", absPath)

		// Set content type mode based on request
		contentTypeMode := fileprocessor.ContentTypeBoth // Default
		switch req.ContentType {
		case "text":
			contentTypeMode = fileprocessor.ContentTypeText
			log.Printf("📝 Content type mode: text only")
		case "images":
			contentTypeMode = fileprocessor.ContentTypeImages
			log.Printf("🖼️  Content type mode: images only")
		case "both", "":
			contentTypeMode = fileprocessor.ContentTypeBoth
			log.Printf("📄 Content type mode: both text and images")
		}
		s.engine.SetContentTypeMode(contentTypeMode)

		progressCallback := func(current, total int, filename string) {
			// Check if cancelled
			select {
			case <-ctx.Done():
				return
			default:
			}

			msg := fmt.Sprintf(`{"type":"progress","current":%d,"total":%d,"file":"%s"}`,
				current, total, filepath.Base(filename))
			s.broadcastProgress(absPath, msg)
		}

		phaseCallback := func(phase, message string, current, total int) {
			// Check if cancelled
			select {
			case <-ctx.Done():
				return
			default:
			}

			msg := fmt.Sprintf(`{"type":"phase","phase":"%s","message":"%s","current":%d,"total":%d}`,
				phase, message, current, total)
			s.broadcastProgress(absPath, msg)
		}

		err := s.engine.IndexFolder(ctx, absPath, dbPath, false, progressCallback, phaseCallback)

		// Check if it was cancelled
		if ctx.Err() == context.Canceled {
			log.Printf("Indexing cancelled: %s", absPath)
			s.broadcastProgress(absPath, `{"type":"cancelled"}`)
			return
		}

		if err != nil {
			log.Printf("Indexing failed: %v", err)
			s.broadcastProgress(absPath, fmt.Sprintf(`{"type":"error","message":"%s"}`, err.Error()))
		} else {
			log.Printf("Indexing completed: %s", absPath)
			s.broadcastProgress(absPath, `{"type":"complete"}`)
		}
	}()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"message": "Indexing started in background",
		"path":    absPath,
	})
}

// handleIndexStatus returns the status of an index (new/modified/deleted files)
func (s *Server) handleIndexStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	indexPath := r.URL.Query().Get("path")
	if indexPath == "" {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "Index path required"})
		return
	}

	// Get database path
	dbPath := filepath.Join(s.config.IndexDir, generateDBName(indexPath))

	// Check if index exists
	if _, err := os.Stat(dbPath); os.IsNotExist(err) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(map[string]string{"error": "Index not found"})
		return
	}

	// Get index status
	status, err := s.engine.GetIndexStatus(dbPath, indexPath)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": fmt.Sprintf("Failed to get status: %v", err)})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(status)
}

// handleUpdateIndex triggers an incremental re-index
func (s *Server) handleUpdateIndex(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		IndexPath string `json:"indexPath"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "Invalid request body"})
		return
	}

	if req.IndexPath == "" {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "Index path required"})
		return
	}

	// Get database path
	dbPath := filepath.Join(s.config.IndexDir, generateDBName(req.IndexPath))

	// Check if index exists
	if _, err := os.Stat(dbPath); os.IsNotExist(err) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(map[string]string{"error": "Index not found"})
		return
	}

	// Create cancellable context for this indexing operation
	ctx, cancel := context.WithCancel(context.Background())

	// Store cancel function
	s.cancelMutex.Lock()
	s.cancelFuncs[req.IndexPath] = cancel
	s.cancelMutex.Unlock()

	// Start incremental indexing in background
	go func() {
		defer func() {
			// Clean up cancel function when done
			s.cancelMutex.Lock()
			delete(s.cancelFuncs, req.IndexPath)
			s.cancelMutex.Unlock()
		}()

		log.Printf("Starting incremental re-index of %s...", req.IndexPath)

		progressCallback := func(current, total int, filename string) {
			// Check if cancelled
			select {
			case <-ctx.Done():
				return
			default:
			}

			msg := fmt.Sprintf(`{"type":"progress","current":%d,"total":%d,"file":"%s"}`,
				current, total, filepath.Base(filename))
			s.broadcastProgress(req.IndexPath, msg)
		}

		phaseCallback := func(phase, message string, current, total int) {
			// Check if cancelled
			select {
			case <-ctx.Done():
				return
			default:
			}

			msg := fmt.Sprintf(`{"type":"phase","phase":"%s","message":"%s","current":%d,"total":%d}`,
				phase, message, current, total)
			s.broadcastProgress(req.IndexPath, msg)
		}

		err := s.engine.IndexFolder(ctx, req.IndexPath, dbPath, true, progressCallback, phaseCallback)

		// Check if it was cancelled
		if ctx.Err() == context.Canceled {
			log.Printf("Re-indexing cancelled: %s", req.IndexPath)
			s.broadcastProgress(req.IndexPath, `{"type":"cancelled"}`)
			return
		}

		if err != nil {
			log.Printf("Re-indexing failed: %v", err)
			s.broadcastProgress(req.IndexPath, fmt.Sprintf(`{"type":"error","message":"%s"}`, err.Error()))
		} else {
			log.Printf("Re-indexing completed: %s", req.IndexPath)
			s.broadcastProgress(req.IndexPath, `{"type":"complete"}`)
		}
	}()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"message": "Re-indexing started in background",
		"path":    req.IndexPath,
	})
}

// handleIndexProgress streams indexing progress via SSE
func (s *Server) handleIndexProgress(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	indexPath := r.URL.Query().Get("path")
	if indexPath == "" {
		http.Error(w, "Index path required", http.StatusBadRequest)
		return
	}

	// Set SSE headers
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	// Create channel for this client
	clientChan := make(chan string, 10)

	// Register client
	s.progressMutex.Lock()
	s.progressClients[indexPath] = append(s.progressClients[indexPath], clientChan)
	s.progressMutex.Unlock()

	// Clean up on disconnect
	defer func() {
		s.progressMutex.Lock()
		clients := s.progressClients[indexPath]
		for i, ch := range clients {
			if ch == clientChan {
				s.progressClients[indexPath] = append(clients[:i], clients[i+1:]...)
				break
			}
		}
		s.progressMutex.Unlock()
		close(clientChan)
	}()

	// Send initial connection message
	fmt.Fprintf(w, "data: {\"type\":\"connected\"}\n\n")
	if f, ok := w.(http.Flusher); ok {
		f.Flush()
	}

	// Stream progress updates
	for {
		select {
		case msg, ok := <-clientChan:
			if !ok {
				return
			}
			fmt.Fprintf(w, "data: %s\n\n", msg)
			if f, ok := w.(http.Flusher); ok {
				f.Flush()
			}
		case <-r.Context().Done():
			return
		}
	}
}

// broadcastProgress sends progress update to all listening clients
func (s *Server) broadcastProgress(indexPath, message string) {
	s.progressMutex.RLock()
	clients := s.progressClients[indexPath]
	s.progressMutex.RUnlock()

	for _, client := range clients {
		select {
		case client <- message:
		default:
			// Client buffer full, skip
		}
	}
}

// handleOpenFolder opens a folder in the system file explorer
func (s *Server) handleOpenFolder(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		Path string
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}

	if req.Path == "" {
		http.Error(w, "Path is required", http.StatusBadRequest)
		return
	}

	// Verify path exists
	if _, err := os.Stat(req.Path); os.IsNotExist(err) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(map[string]string{"error": "Folder not found"})
		return
	}

	// Open folder in system file explorer
	var cmd *exec.Cmd
	pathToOpen := req.Path

	// Check if running in WSL
	if runtime.GOOS == "linux" {
		// Check if we're in WSL by looking for /mnt/c
		if _, err := os.Stat("/mnt/c"); err == nil {
			// Convert WSL path to Windows path if needed
			if strings.HasPrefix(pathToOpen, "/mnt/") {
				// Already a /mnt path, convert to Windows format
				// /mnt/c/Users/... -> C:\Users\...
				parts := strings.Split(pathToOpen, "/")
				if len(parts) >= 3 {
					drive := strings.ToUpper(parts[2])
					windowsPath := drive + ":\\" + strings.Join(parts[3:], "\\")
					pathToOpen = windowsPath
				}
			} else if strings.HasPrefix(pathToOpen, "/home/") {
				// WSL home directory, need to get Windows equivalent
				// Use wslpath to convert
				out, err := exec.Command("wslpath", "-w", pathToOpen).Output()
				if err == nil {
					pathToOpen = strings.TrimSpace(string(out))
				}
			}
			// Use explorer.exe from WSL
			cmd = exec.Command("/mnt/c/Windows/explorer.exe", pathToOpen)
		} else {
			// Regular Linux, use xdg-open if available
			cmd = exec.Command("xdg-open", pathToOpen)
		}
	} else if runtime.GOOS == "windows" {
		cmd = exec.Command("explorer", pathToOpen)
	} else if runtime.GOOS == "darwin" {
		cmd = exec.Command("open", pathToOpen)
	} else {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": "Unsupported operating system"})
		return
	}

	if err := cmd.Start(); err != nil {
		log.Printf("Failed to open folder: %v (path: %s)", err, pathToOpen)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": fmt.Sprintf("Failed to open folder: %v", err)})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"success": "Folder opened"})
}

// handleOpenFileLocation opens the file location in the system file explorer and selects the file
func (s *Server) handleOpenFileLocation(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		Path string
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}

	if req.Path == "" {
		http.Error(w, "Path is required", http.StatusBadRequest)
		return
	}

	// Verify file exists
	if _, err := os.Stat(req.Path); os.IsNotExist(err) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(map[string]string{"error": "File not found"})
		return
	}

	// Open file location in system file explorer and select the file
	var cmd *exec.Cmd
	pathToOpen := req.Path

	// Check if running in WSL
	if runtime.GOOS == "linux" {
		// Check if we're in WSL by looking for /mnt/c
		if _, err := os.Stat("/mnt/c"); err == nil {
			// Convert WSL path to Windows path if needed
			if strings.HasPrefix(pathToOpen, "/mnt/") {
				// Already a /mnt path, convert to Windows format
				// /mnt/c/Users/... -> C:\Users\...
				parts := strings.Split(pathToOpen, "/")
				if len(parts) >= 3 {
					drive := strings.ToUpper(parts[2])
					windowsPath := drive + ":\\" + strings.Join(parts[3:], "\\")
					pathToOpen = windowsPath
				}
			} else if strings.HasPrefix(pathToOpen, "/home/") {
				// WSL home directory, need to get Windows equivalent
				// Use wslpath to convert
				out, err := exec.Command("wslpath", "-w", pathToOpen).Output()
				if err == nil {
					pathToOpen = strings.TrimSpace(string(out))
				}
			}
			// Use explorer.exe from WSL with /select flag to select the file
			cmd = exec.Command("/mnt/c/Windows/explorer.exe", "/select,", pathToOpen)
		} else {
			// Regular Linux, open parent directory
			parentDir := filepath.Dir(pathToOpen)
			cmd = exec.Command("xdg-open", parentDir)
		}
	} else if runtime.GOOS == "windows" {
		// Windows: use explorer /select to select the file
		cmd = exec.Command("explorer", "/select,"+pathToOpen)
	} else if runtime.GOOS == "darwin" {
		// macOS: use open -R to reveal the file
		cmd = exec.Command("open", "-R", pathToOpen)
	} else {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": "Unsupported operating system"})
		return
	}

	if err := cmd.Start(); err != nil {
		log.Printf("Failed to open file location: %v (path: %s)", err, pathToOpen)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": fmt.Sprintf("Failed to open file location: %v", err)})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"success": "File location opened"})
}

// handleCancelIndex cancels an active indexing operation
func (s *Server) handleCancelIndex(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		IndexPath string `json:"indexPath"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "Invalid request body"})
		return
	}

	if req.IndexPath == "" {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "Index path required"})
		return
	}

	// Get cancel function
	s.cancelMutex.RLock()
	cancelFunc, exists := s.cancelFuncs[req.IndexPath]
	s.cancelMutex.RUnlock()

	if !exists {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(map[string]string{"error": "No active indexing operation for this path"})
		return
	}

	// Cancel the operation
	cancelFunc()
	log.Printf("🛑 Cancelled indexing: %s", req.IndexPath)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"message": "Indexing cancelled",
		"path":    req.IndexPath,
	})
}

// handleDeleteIndex deletes an index database
func (s *Server) handleDeleteIndex(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		IndexPath string `json:"indexPath"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "Invalid request body"})
		return
	}

	if req.IndexPath == "" {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "Index path required"})
		return
	}

	// Get database path
	dbPath := filepath.Join(s.config.IndexDir, generateDBName(req.IndexPath))

	// Check if index exists
	if _, err := os.Stat(dbPath); os.IsNotExist(err) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(map[string]string{"error": "Index not found"})
		return
	}

	// Delete the database file
	if err := os.Remove(dbPath); err != nil {
		log.Printf("Failed to delete index: %v", err)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": fmt.Sprintf("Failed to delete index: %v", err)})
		return
	}

	log.Printf("🗑️  Deleted index: %s", dbPath)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"message": "Index deleted successfully",
		"path":    req.IndexPath,
	})
}
