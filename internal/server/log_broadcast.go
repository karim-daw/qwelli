package server

import (
	"encoding/json"
	"io"
	"os"
	"strings"
	"sync"
)

// BroadcastWriter is an io.Writer that tees output to an underlying writer
// (typically os.Stderr) and broadcasts each log line to terminal clients.
type BroadcastWriter struct {
	mu   sync.Mutex
	orig io.Writer
}

// NewBroadcastWriter creates a BroadcastWriter that tees to orig.
func NewBroadcastWriter(orig io.Writer) *BroadcastWriter {
	if orig == nil {
		orig = os.Stderr
	}
	return &BroadcastWriter{orig: orig}
}

// Write implements io.Writer. It forwards the bytes to the original writer
// and broadcasts each non-empty line to connected terminal clients.
func (w *BroadcastWriter) Write(p []byte) (n int, err error) {
	w.mu.Lock()
	n, err = w.orig.Write(p)
	w.mu.Unlock()

	lines := strings.Split(string(p), "\n")
	for _, line := range lines {
		line = strings.TrimRight(line, "\r")
		if line == "" {
			continue
		}
		level := detectLogLevel(line)
		encoded, _ := json.Marshal(line)
		BroadcastTerminalOutput(`{"type":"log","level":"` + level + `","message":` + string(encoded) + `}`)
	}
	return
}

// detectLogLevel infers a log level from message content.
func detectLogLevel(msg string) string {
	lower := strings.ToLower(msg)
	switch {
	case strings.Contains(msg, "⚠️") || strings.Contains(lower, "warn"):
		return "warning"
	case strings.Contains(msg, "❌") || strings.Contains(lower, "error") ||
		strings.Contains(lower, "failed") || strings.Contains(lower, "panic"):
		return "error"
	case strings.Contains(msg, "✅") || strings.Contains(msg, "✓") ||
		strings.Contains(lower, "completed"):
		return "success"
	default:
		return "info"
	}
}
