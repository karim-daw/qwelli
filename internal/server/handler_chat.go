package server

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"

	"github.com/karim-daw/qwelli/internal/agent"
)

type chatRequest struct {
	Index   string `json:"index"`
	Message string `json:"message"`
}

type chatClearRequest struct {
	Index string `json:"index"`
}

func (s *Server) handleChat(w http.ResponseWriter, r *http.Request) {
	if !requireMethod(w, r, http.MethodPost) {
		return
	}

	var req chatRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if req.Index == "" || req.Message == "" {
		jsonError(w, http.StatusBadRequest, "index and message are required")
		return
	}

	cfg := s.service.Config()
	if cfg.FoundryEndpoint == "" || cfg.FoundryAPIKey == "" {
		jsonError(w, http.StatusBadRequest, "Foundry endpoint and API key must be configured for chat")
		return
	}

	// Get or create agent session for this index
	ag, err := s.getOrCreateAgent(req.Index)
	if err != nil {
		jsonError(w, http.StatusInternalServerError, fmt.Sprintf("failed to create agent: %v", err))
		return
	}

	// Set up SSE response
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	flusher, ok := w.(http.Flusher)
	if !ok {
		jsonError(w, http.StatusInternalServerError, "streaming not supported")
		return
	}

	ctx := r.Context()

	err = ag.ChatStream(ctx, req.Message, func(ev agent.ChatEvent) {
		data, err := json.Marshal(ev)
		if err != nil {
			log.Printf("chat: marshal event error: %v", err)
			return
		}
		select {
		case <-ctx.Done():
			return
		default:
			fmt.Fprintf(w, "data: %s\n\n", data)
			flusher.Flush()
		}
	})

	if err != nil {
		// If the context was cancelled (client disconnect), don't write an error
		if ctx.Err() != nil {
			return
		}
		errEvent, _ := json.Marshal(agent.ChatEvent{Type: "error", Text: err.Error()})
		fmt.Fprintf(w, "data: %s\n\n", errEvent)
		flusher.Flush()
	}
}

func (s *Server) handleChatClear(w http.ResponseWriter, r *http.Request) {
	if !requireMethod(w, r, http.MethodPost) {
		return
	}

	var req chatClearRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if req.Index == "" {
		jsonError(w, http.StatusBadRequest, "index is required")
		return
	}

	s.chatMutex.Lock()
	delete(s.chatAgents, req.Index)
	s.chatMutex.Unlock()

	jsonOK(w, map[string]string{"status": "cleared"})
}

func (s *Server) getOrCreateAgent(indexPath string) (*agent.Agent, error) {
	s.chatMutex.RLock()
	ag, ok := s.chatAgents[indexPath]
	s.chatMutex.RUnlock()
	if ok {
		return ag, nil
	}

	s.chatMutex.Lock()
	defer s.chatMutex.Unlock()

	// Double-check after acquiring write lock
	if ag, ok := s.chatAgents[indexPath]; ok {
		return ag, nil
	}

	cfg := s.service.Config()
	model := cfg.FoundryModel
	if model == "" {
		model = "claude-sonnet-4-6"
	}

	ag, err := agent.New(cfg.FoundryEndpoint, cfg.FoundryAPIKey, model, s.service, indexPath)
	if err != nil {
		return nil, err
	}

	s.chatAgents[indexPath] = ag
	return ag, nil
}
