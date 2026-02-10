package server

import (
	"fmt"
	"net/http"
	"sync"
)

var (
	terminalClients      []chan string
	terminalClientsMutex sync.RWMutex
	terminalHistory      []string
	terminalHistoryMutex sync.RWMutex
	maxTerminalHistory   = 100
)

// BroadcastTerminalOutput sends a log message to all connected terminal clients
func BroadcastTerminalOutput(message string) {
	// Add to history
	terminalHistoryMutex.Lock()
	terminalHistory = append(terminalHistory, message)
	if len(terminalHistory) > maxTerminalHistory {
		terminalHistory = terminalHistory[len(terminalHistory)-maxTerminalHistory:]
	}
	terminalHistoryMutex.Unlock()

	// Broadcast to connected clients
	terminalClientsMutex.RLock()
	clients := make([]chan string, len(terminalClients))
	copy(clients, terminalClients)
	terminalClientsMutex.RUnlock()

	for _, c := range clients {
		select {
		case c <- message:
		default:
			// Client buffer full, skip
		}
	}
}

func (s *Server) handleTerminalStream(w http.ResponseWriter, r *http.Request) {
	if !requireMethod(w, r, http.MethodGet) {
		return
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	ch := make(chan string, 50)
	terminalClientsMutex.Lock()
	terminalClients = append(terminalClients, ch)
	terminalClientsMutex.Unlock()

	defer func() {
		terminalClientsMutex.Lock()
		for i, c := range terminalClients {
			if c == ch {
				terminalClients = append(terminalClients[:i], terminalClients[i+1:]...)
				break
			}
		}
		terminalClientsMutex.Unlock()
		close(ch)
	}()

	// Send connection message
	fmt.Fprintf(w, "data: {\"type\":\"connected\"}\n\n")
	if f, ok := w.(http.Flusher); ok {
		f.Flush()
	}

	// Send history
	terminalHistoryMutex.RLock()
	for _, msg := range terminalHistory {
		fmt.Fprintf(w, "data: %s\n\n", msg)
	}
	terminalHistoryMutex.RUnlock()
	if f, ok := w.(http.Flusher); ok {
		f.Flush()
	}

	// Stream new messages
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
