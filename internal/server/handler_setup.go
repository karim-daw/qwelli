package server

import (
	"encoding/json"
	"net/http"

	"github.com/karim-daw/qwelli/internal/config"
)

// handleSetupStatus returns whether the app needs first-run setup (config missing)
func handleSetupStatus(w http.ResponseWriter, r *http.Request) {
	if !requireMethod(w, r, http.MethodGet) {
		return
	}
	jsonOK(w, map[string]bool{"needsSetup": !config.Exists()})
}

// handleSetupSave accepts config from the setup form and saves it.
// The SetupServer provides this as a method so it can trigger restart after save.
func (ss *SetupServer) handleSetupSave(w http.ResponseWriter, r *http.Request) {
	if !requireMethod(w, r, http.MethodPost) {
		return
	}
	var req struct {
		APIKey            string `json:"apiKey"`
		Model             string `json:"model"`
		Endpoint          string `json:"endpoint"`
		EnableMultimodal  bool   `json:"enableMultimodal"`
		EnableReranker    bool   `json:"enableReranker"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonError(w, http.StatusBadRequest, "Invalid request")
		return
	}
	if req.APIKey == "" {
		jsonError(w, http.StatusBadRequest, "API key is required")
		return
	}

	cfg := config.DefaultConfig()
	cfg.APIKey = req.APIKey
	if req.Model != "" {
		cfg.Model = req.Model
	}
	if req.Endpoint != "" {
		cfg.Endpoint = req.Endpoint
	}
	cfg.EnableMultimodal = req.EnableMultimodal
	cfg.EnableReranker = req.EnableReranker
	cfg.EmbeddingProvider = "voyage"

	if err := cfg.Save(); err != nil {
		jsonError(w, http.StatusInternalServerError, "Failed to save config: "+err.Error())
		return
	}
	if err := cfg.EnsureIndexDir(); err != nil {
		jsonError(w, http.StatusInternalServerError, "Failed to create index directory: "+err.Error())
		return
	}

	jsonOK(w, map[string]string{"status": "ok"})
	ss.requestRestart()
}
