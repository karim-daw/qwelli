package server

import (
	"context"
	"net/http"
	"time"
)

func (s *Server) handleShutdown(w http.ResponseWriter, r *http.Request) {
	if !requireMethod(w, r, http.MethodPost) {
		return
	}
	if s.httpServer == nil {
		jsonError(w, http.StatusInternalServerError, "server not ready")
		return
	}
	jsonOK(w, map[string]string{"status": "shutting down"})
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = s.httpServer.Shutdown(ctx)
	}()
}
