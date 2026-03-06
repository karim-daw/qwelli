package server

import (
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"
)

// skipDirNames are directory names (lowercased) that are generated/system
// artifacts or OS infrastructure — not user content. Matched case-insensitively.
var skipDirNames = map[string]bool{
	// Package manager caches / build output
	"node_modules": true, ".npm": true, ".yarn": true, ".pnpm-store": true,
	"__pycache__": true, ".venv": true, "venv": true, "env": true,
	"dist": true, "build": true, "out": true, "target": true,
	".next": true, ".nuxt": true, ".svelte-kit": true,
	".cargo": true, ".gradle": true,

	// VCS internals
	".git": true, ".svn": true, ".hg": true, ".bzr": true,

	// Windows network share / filesystem junk
	"$recycle.bin": true, "recycler": true,
	"system volume information": true,
	"~snapshot": true, ".snapshot": true, // NetApp/CIFS snapshot dirs
	"dfsrprivate": true,                  // DFS replication private dir

	// macOS metadata on SMB shares
	".spotlight-v100": true, ".trashes": true,
	".fseventsd": true, ".temporaryitems": true,

	// Windows OS dirs (catch-all if user accidentally roots at C:\)
	"windows": true, "program files": true, "program files (x86)": true,
	"programdata": true, "recovery": true, "winsxs": true,
}

const browseMaxEntries = 500

// indexedPathsCache is a short-lived (10s) cache of indexed folder paths used
// by the browse handler to set the isIndexed flag without calling ListIndexes
// on every request.
var (
	indexedPathsCacheMu  sync.RWMutex
	indexedPathsCacheMap map[string]bool
	indexedPathsCacheAt  time.Time
)

func (s *Server) getIndexedPaths() map[string]bool {
	indexedPathsCacheMu.RLock()
	if time.Since(indexedPathsCacheAt) < 10*time.Second && indexedPathsCacheMap != nil {
		p := indexedPathsCacheMap
		indexedPathsCacheMu.RUnlock()
		return p
	}
	indexedPathsCacheMu.RUnlock()

	indexes, _ := s.service.ListIndexes()
	paths := make(map[string]bool, len(indexes))
	for _, idx := range indexes {
		paths[browseNormPath(idx.FolderPath)] = true
	}

	indexedPathsCacheMu.Lock()
	indexedPathsCacheMap = paths
	indexedPathsCacheAt = time.Now()
	indexedPathsCacheMu.Unlock()

	return paths
}

func browseNormPath(p string) string {
	c := filepath.Clean(p)
	if runtime.GOOS == "windows" {
		return strings.ToLower(c)
	}
	return c
}

// browseParent returns the parent directory of clean, or "" if already at root.
// Handles both local drive roots (C:\) and UNC share roots (\\server\share).
func browseParent(clean string) string {
	// UNC share root: VolumeName equals the full path (e.g. \\server\share)
	if filepath.VolumeName(clean) == clean {
		return ""
	}
	d := filepath.Dir(clean)
	if d == clean {
		return ""
	}
	return d
}

func (s *Server) handleBrowse(w http.ResponseWriter, r *http.Request) {
	if !requireMethod(w, r, http.MethodGet) {
		return
	}

	path := r.URL.Query().Get("path")
	if path == "" {
		// No path — signal frontend to show root picker
		jsonOK(w, BrowseResponse{Entries: []BrowseEntry{}, Parent: "", Total: 0})
		return
	}

	clean := filepath.Clean(path)

	info, err := os.Stat(clean)
	if err != nil {
		jsonError(w, http.StatusNotFound, "Path not found")
		return
	}
	if !info.IsDir() {
		jsonError(w, http.StatusBadRequest, "Path is not a directory")
		return
	}

	rawEntries, err := os.ReadDir(clean)
	if err != nil {
		jsonError(w, http.StatusInternalServerError, "Failed to read directory")
		return
	}

	indexedPaths := s.getIndexedPaths()

	// Split into dirs and files (os.ReadDir returns alphabetical order already)
	var dirs, files []os.DirEntry
	for _, e := range rawEntries {
		name := e.Name()
		// Skip hidden (dot-prefix) and known system/generated names
		if strings.HasPrefix(name, ".") || skipDirNames[strings.ToLower(name)] {
			continue
		}
		if e.Type().IsDir() {
			dirs = append(dirs, e)
		} else {
			files = append(files, e)
		}
	}

	total := len(dirs) + len(files)
	entries := make([]BrowseEntry, 0, min(total, browseMaxEntries))
	added := 0

	addEntry := func(e os.DirEntry, isDir bool) {
		entPath := filepath.Join(clean, e.Name())
		be := BrowseEntry{
			Name:  e.Name(),
			Path:  entPath,
			IsDir: isDir,
		}
		if !isDir {
			// Only stat files for size/mtime — avoid extra syscalls on dirs
			if fi, err := e.Info(); err == nil {
				be.Size = fi.Size()
				be.ModifiedAt = fi.ModTime().Format(time.RFC3339)
			}
		} else if indexedPaths[browseNormPath(entPath)] {
			be.IsIndexed = true
		}
		entries = append(entries, be)
		added++
	}

	for _, e := range dirs {
		if added >= browseMaxEntries {
			break
		}
		addEntry(e, true)
	}
	for _, e := range files {
		if added >= browseMaxEntries {
			break
		}
		addEntry(e, false)
	}

	jsonOK(w, BrowseResponse{
		Entries:   entries,
		Parent:    browseParent(clean),
		Total:     total,
		Truncated: total > browseMaxEntries,
	})
}
