package server

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/karim-daw/qwelli/internal/service"
)

func (s *Server) handleGetConfig(w http.ResponseWriter, r *http.Request) {
	if !requireMethod(w, r, http.MethodGet) {
		return
	}

	cfg := s.service.Config()

	masked := ""
	if len(cfg.APIKey) > 4 {
		masked = "****" + cfg.APIKey[len(cfg.APIKey)-4:]
	} else if cfg.APIKey != "" {
		masked = "****"
	}

	jsonOK(w, ConfigResponse{
		APIKeyMasked: masked,
		Model:        cfg.Model,
		Endpoint:     cfg.Endpoint,
	})
}

func (s *Server) handleUpdateConfig(w http.ResponseWriter, r *http.Request) {
	if !requireMethod(w, r, http.MethodPost) {
		return
	}

	// Refuse if indexing is active
	s.cancelMutex.RLock()
	active := len(s.cancelFuncs)
	s.cancelMutex.RUnlock()
	if active > 0 {
		jsonError(w, http.StatusConflict, "Cannot update config while indexing is active")
		return
	}

	var req UpdateConfigRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	update := service.ConfigUpdate{
		APIKey:   req.APIKey,
		Model:    req.Model,
		Endpoint: req.Endpoint,
	}

	warnings, err := s.service.UpdateConfig(update)
	if err != nil {
		jsonError(w, http.StatusBadRequest, err.Error())
		return
	}

	// Broadcast to terminal
	var parts []string
	if req.APIKey != nil {
		parts = append(parts, "API key")
	}
	if req.Model != nil {
		parts = append(parts, fmt.Sprintf("model=%s", *req.Model))
	}
	if req.Endpoint != nil {
		parts = append(parts, fmt.Sprintf("endpoint=%s", *req.Endpoint))
	}
	if len(parts) > 0 {
		BroadcastTerminalOutput(fmt.Sprintf(
			`{"type":"log","level":"info","message":"⚙️ Config updated: %s"}`,
			strings.Join(parts, ", ")))
	}

	jsonOK(w, UpdateConfigResponse{
		Status:   "ok",
		Warnings: warnings,
	})
}
