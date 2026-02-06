package server

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
)

func (s *Server) handleFileAccess(w http.ResponseWriter, r *http.Request) {
	if !requireMethod(w, r, http.MethodGet) {
		return
	}
	filePath := r.URL.Query().Get("path")
	if filePath == "" {
		jsonError(w, http.StatusBadRequest, "File path required")
		return
	}
	if !s.service.IsFileInIndex(filePath) {
		jsonError(w, http.StatusNotFound, "File not in index")
		return
	}
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		jsonError(w, http.StatusNotFound, "File not found")
		return
	}
	http.ServeFile(w, r, filePath)
}

func (s *Server) handleOpenFolder(w http.ResponseWriter, r *http.Request) {
	s.openInExplorer(w, r, false)
}

func (s *Server) handleOpenFileLocation(w http.ResponseWriter, r *http.Request) {
	s.openInExplorer(w, r, true)
}

func (s *Server) openInExplorer(w http.ResponseWriter, r *http.Request, selectFile bool) {
	if !requireMethod(w, r, http.MethodPost) {
		return
	}
	var req struct{ Path string }
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Path == "" {
		jsonError(w, http.StatusBadRequest, "Path required")
		return
	}
	if _, err := os.Stat(req.Path); os.IsNotExist(err) {
		jsonError(w, http.StatusNotFound, "Path not found")
		return
	}
	cmd := buildExplorerCmd(req.Path, selectFile)
	if cmd == nil {
		jsonError(w, http.StatusInternalServerError, "Unsupported OS")
		return
	}
	if err := cmd.Start(); err != nil {
		jsonError(w, http.StatusInternalServerError, fmt.Sprintf("Open failed: %v", err))
		return
	}
	jsonOK(w, map[string]string{"success": "ok"})
}
