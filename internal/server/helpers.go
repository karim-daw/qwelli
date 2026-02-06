package server

import (
	"encoding/json"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/karim-daw/qwelli/internal/engine/fileprocessor"
)

func jsonOK(w http.ResponseWriter, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(v)
}

func jsonError(w http.ResponseWriter, status int, msg string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(map[string]string{"error": msg})
}

func requireMethod(w http.ResponseWriter, r *http.Request, method string) bool {
	if r.Method != method {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return false
	}
	return true
}

func metaString(m map[string]interface{}, key, fallback string) string {
	if v, ok := m[key].(string); ok && v != "" {
		return v
	}
	return fallback
}

func metaInt(m map[string]interface{}, key string) int {
	switch v := m[key].(type) {
	case int:
		return v
	case float64:
		return int(v)
	}
	return 0
}

func metaPageNumbers(m map[string]interface{}) []int {
	if m == nil {
		return nil
	}
	if pn, ok := m["page_numbers"].([]int); ok {
		return pn
	}
	if arr, ok := m["page_numbers"].([]interface{}); ok {
		out := make([]int, 0, len(arr))
		for _, v := range arr {
			switch n := v.(type) {
			case int:
				out = append(out, n)
			case int64:
				out = append(out, int(n))
			case float64:
				out = append(out, int(n))
			}
		}
		return out
	}
	return nil
}

func buildExplorerCmd(path string, selectFile bool) *exec.Cmd {
	path = toNativePath(path)
	switch runtime.GOOS {
	case "linux":
		if _, err := os.Stat("/mnt/c"); err == nil { // WSL
			if selectFile {
				return exec.Command("/mnt/c/Windows/explorer.exe", "/select,", path)
			}
			return exec.Command("/mnt/c/Windows/explorer.exe", path)
		}
		if selectFile {
			return exec.Command("xdg-open", filepath.Dir(path))
		}
		return exec.Command("xdg-open", path)
	case "windows":
		if selectFile {
			return exec.Command("explorer", "/select,"+path)
		}
		return exec.Command("explorer", path)
	case "darwin":
		if selectFile {
			return exec.Command("open", "-R", path)
		}
		return exec.Command("open", path)
	}
	return nil
}

func toNativePath(p string) string {
	if runtime.GOOS != "linux" {
		return p
	}
	if _, err := os.Stat("/mnt/c"); err != nil {
		return p
	}
	if strings.HasPrefix(p, "/mnt/") {
		parts := strings.Split(p, "/")
		if len(parts) >= 3 {
			return strings.ToUpper(parts[2]) + ":\\" + strings.Join(parts[3:], "\\")
		}
	}
	if strings.HasPrefix(p, "/home/") {
		if out, err := exec.Command("wslpath", "-w", p).Output(); err == nil {
			return strings.TrimSpace(string(out))
		}
	}
	return p
}

func parseContentTypeMode(s string) fileprocessor.ContentTypeMode {
	switch s {
	case "text":
		return fileprocessor.ContentTypeText
	case "images":
		return fileprocessor.ContentTypeImages
	default:
		return fileprocessor.ContentTypeBoth
	}
}
