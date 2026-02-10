package server

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"

	"github.com/karim-daw/qwelli/internal/service"
)

func (s *Server) broadcastProgress(indexPath, message string) {
	s.progressMutex.RLock()
	clients := s.progressClients[indexPath]
	s.progressMutex.RUnlock()
	for _, c := range clients {
		select {
		case c <- message:
		default:
		}
	}
}

func (s *Server) handleCreateIndex(w http.ResponseWriter, r *http.Request) {
	if !requireMethod(w, r, http.MethodPost) {
		return
	}
	var req struct {
		FolderPath  string `json:"folderPath"`
		ContentType string `json:"contentType,omitempty"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonError(w, http.StatusBadRequest, "Invalid request")
		return
	}
	if req.FolderPath == "" {
		jsonError(w, http.StatusBadRequest, "Folder path required")
		return
	}
	absPath, err := filepath.Abs(req.FolderPath)
	if err != nil {
		jsonError(w, http.StatusBadRequest, "Invalid path")
		return
	}
	if stat, err := os.Stat(absPath); err != nil || !stat.IsDir() {
		jsonError(w, http.StatusBadRequest, "Folder does not exist")
		return
	}

	s.service.Engine().SetContentTypeMode(parseContentTypeMode(req.ContentType))
	s.startIndexing(absPath, false)
	jsonOK(w, map[string]string{"message": "Indexing started", "path": absPath})
}

func (s *Server) handleUpdateIndex(w http.ResponseWriter, r *http.Request) {
	if !requireMethod(w, r, http.MethodPost) {
		return
	}
	var req struct {
		IndexPath string `json:"indexPath"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonError(w, http.StatusBadRequest, "Invalid request")
		return
	}
	if req.IndexPath == "" {
		jsonError(w, http.StatusBadRequest, "Index path required")
		return
	}
	dbPath, err := s.service.GenerateDBPath(req.IndexPath)
	if err != nil {
		jsonError(w, http.StatusBadRequest, fmt.Sprintf("Invalid path: %v", err))
		return
	}
	if _, err := os.Stat(dbPath); os.IsNotExist(err) {
		jsonError(w, http.StatusNotFound, "Index not found")
		return
	}

	s.startIndexing(req.IndexPath, true)
	jsonOK(w, map[string]string{"message": "Re-indexing started", "path": req.IndexPath})
}

// startIndexing runs indexing in a background goroutine with SSE progress.
func (s *Server) startIndexing(folderPath string, incremental bool) {
	ctx, cancel := context.WithCancel(context.Background())

	s.cancelMutex.Lock()
	s.cancelFuncs[folderPath] = cancel
	s.cancelMutex.Unlock()

	go func() {
		defer func() {
			s.cancelMutex.Lock()
			delete(s.cancelFuncs, folderPath)
			s.cancelMutex.Unlock()
		}()

		kind := "full"
		if incremental {
			kind = "incremental"
		}
		logMsg := fmt.Sprintf("Starting %s indexing: %s", kind, folderPath)
		log.Printf(logMsg)
		BroadcastTerminalOutput(fmt.Sprintf(`{"type":"log","level":"info","message":"%s"}`, logMsg))

		progressCb := func(current, total int, filename string) {
			select {
			case <-ctx.Done():
				return
			default:
			}
			s.broadcastProgress(folderPath, fmt.Sprintf(
				`{"type":"progress","current":%d,"total":%d,"file":"%s"}`,
				current, total, filepath.Base(filename)))
			BroadcastTerminalOutput(fmt.Sprintf(
				`{"type":"log","level":"info","message":"📄 [%d/%d] %s"}`,
				current, total, filepath.Base(filename)))
		}
		phaseCb := func(phase, message string, current, total int) {
			select {
			case <-ctx.Done():
				return
			default:
			}
			s.broadcastProgress(folderPath, fmt.Sprintf(
				`{"type":"phase","phase":"%s","message":"%s","current":%d,"total":%d}`,
				phase, message, current, total))
			BroadcastTerminalOutput(fmt.Sprintf(
				`{"type":"log","level":"info","message":"⚙️ [%s] %s"}`,
				phase, message))
		}

		opts := service.IndexOptions{Incremental: incremental}
		var err error
		if incremental {
			err = s.service.UpdateIndex(ctx, folderPath, opts, progressCb, phaseCb)
		} else {
			err = s.service.CreateIndex(ctx, folderPath, opts, progressCb, phaseCb)
		}

		if ctx.Err() == context.Canceled {
			logMsg := fmt.Sprintf("Indexing cancelled: %s", folderPath)
			log.Printf(logMsg)
			BroadcastTerminalOutput(fmt.Sprintf(`{"type":"log","level":"warning","message":"%s"}`, logMsg))
			s.broadcastProgress(folderPath, `{"type":"cancelled"}`)
			return
		}
		if err != nil {
			logMsg := fmt.Sprintf("Indexing failed: %v", err)
			log.Printf(logMsg)
			BroadcastTerminalOutput(fmt.Sprintf(`{"type":"log","level":"error","message":"%s"}`, logMsg))
			s.broadcastProgress(folderPath, fmt.Sprintf(`{"type":"error","message":"%s"}`, err.Error()))
		} else {
			logMsg := fmt.Sprintf("Indexing completed: %s", folderPath)
			log.Printf(logMsg)
			BroadcastTerminalOutput(fmt.Sprintf(`{"type":"log","level":"success","message":"%s"}`, logMsg))
			s.broadcastProgress(folderPath, `{"type":"complete"}`)
		}
	}()
}

func (s *Server) handleDeleteIndex(w http.ResponseWriter, r *http.Request) {
	if !requireMethod(w, r, http.MethodPost) {
		return
	}
	var req struct {
		IndexPath string `json:"indexPath"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonError(w, http.StatusBadRequest, "Invalid request")
		return
	}
	if req.IndexPath == "" {
		jsonError(w, http.StatusBadRequest, "Index path required")
		return
	}
	if err := s.service.DeleteIndex(req.IndexPath); err != nil {
		jsonError(w, http.StatusInternalServerError, fmt.Sprintf("Delete failed: %v", err))
		return
	}
	logMsg := fmt.Sprintf("🗑️  Deleted index: %s", req.IndexPath)
	log.Printf(logMsg)
	BroadcastTerminalOutput(fmt.Sprintf(`{"type":"log","level":"info","message":"%s"}`, logMsg))
	jsonOK(w, map[string]string{"message": "Deleted", "path": req.IndexPath})
}

func (s *Server) handleIndexStatus(w http.ResponseWriter, r *http.Request) {
	if !requireMethod(w, r, http.MethodGet) {
		return
	}
	indexPath := r.URL.Query().Get("path")
	if indexPath == "" {
		jsonError(w, http.StatusBadRequest, "Index path required")
		return
	}
	status, err := s.service.GetIndexStatus(indexPath)
	if err != nil {
		jsonError(w, http.StatusInternalServerError, fmt.Sprintf("Status failed: %v", err))
		return
	}
	jsonOK(w, status)
}

func (s *Server) handleIndexProgress(w http.ResponseWriter, r *http.Request) {
	if !requireMethod(w, r, http.MethodGet) {
		return
	}
	indexPath := r.URL.Query().Get("path")
	if indexPath == "" {
		http.Error(w, "Index path required", http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	ch := make(chan string, 10)
	s.progressMutex.Lock()
	s.progressClients[indexPath] = append(s.progressClients[indexPath], ch)
	s.progressMutex.Unlock()

	defer func() {
		s.progressMutex.Lock()
		clients := s.progressClients[indexPath]
		for i, c := range clients {
			if c == ch {
				s.progressClients[indexPath] = append(clients[:i], clients[i+1:]...)
				break
			}
		}
		s.progressMutex.Unlock()
		close(ch)
	}()

	fmt.Fprintf(w, "data: {\"type\":\"connected\"}\n\n")
	if f, ok := w.(http.Flusher); ok {
		f.Flush()
	}

	for {
		select {
		case msg, ok := <-ch:
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

func (s *Server) handleCancelIndex(w http.ResponseWriter, r *http.Request) {
	if !requireMethod(w, r, http.MethodPost) {
		return
	}
	var req struct {
		IndexPath string `json:"indexPath"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonError(w, http.StatusBadRequest, "Invalid request")
		return
	}
	if req.IndexPath == "" {
		jsonError(w, http.StatusBadRequest, "Index path required")
		return
	}

	s.cancelMutex.RLock()
	cancel, exists := s.cancelFuncs[req.IndexPath]
	s.cancelMutex.RUnlock()

	if !exists {
		jsonError(w, http.StatusNotFound, "No active indexing")
		return
	}
	cancel()
	log.Printf("🛑 Cancelled: %s", req.IndexPath)
	jsonOK(w, map[string]string{"message": "Cancelled", "path": req.IndexPath})
}
