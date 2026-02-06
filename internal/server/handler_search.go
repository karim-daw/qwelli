package server

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"github.com/karim-daw/qwelli/internal/engine"
)

func (s *Server) cleanupExpiredCache() {
	for range time.NewTicker(time.Minute).C {
		s.cacheMutex.Lock()
		now := time.Now()
		for k, v := range s.searchCache {
			if now.Sub(v.Timestamp) > s.cacheTTL {
				delete(s.searchCache, k)
			}
		}
		s.cacheMutex.Unlock()
	}
}

func (s *Server) getCached(key string) ([]SearchResultItem, bool) {
	s.cacheMutex.RLock()
	defer s.cacheMutex.RUnlock()
	if e, ok := s.searchCache[key]; ok && time.Since(e.Timestamp) <= s.cacheTTL {
		return e.Results, true
	}
	return nil, false
}

func (s *Server) setCache(key string, results []SearchResultItem) {
	s.cacheMutex.Lock()
	s.searchCache[key] = &searchCacheEntry{Results: results, Timestamp: time.Now()}
	s.cacheMutex.Unlock()
}

func (s *Server) handleListIndexes(w http.ResponseWriter, r *http.Request) {
	if !requireMethod(w, r, http.MethodGet) {
		return
	}
	indexes, err := s.service.ListIndexes()
	if err != nil {
		jsonError(w, http.StatusInternalServerError, fmt.Sprintf("list indexes: %v", err))
		return
	}
	apiIndexes := make([]IndexInfo, 0, len(indexes))
	for _, idx := range indexes {
		apiIndexes = append(apiIndexes, IndexInfo{
			Name:          filepath.Base(idx.FolderPath),
			Path:          idx.FolderPath,
			DocumentCount: idx.DocumentCount,
			LastModified:  idx.LastModified.Format("2006-01-02 15:04:05"),
		})
	}
	jsonOK(w, map[string]interface{}{"indexes": apiIndexes})
}

func (s *Server) handleSearch(w http.ResponseWriter, r *http.Request) {
	if !requireMethod(w, r, http.MethodGet) {
		return
	}

	q := r.URL.Query()
	query, indexPath := q.Get("q"), q.Get("index")
	strategy, contentType := q.Get("strategy"), q.Get("content_type")

	if query == "" {
		jsonError(w, http.StatusBadRequest, "Query parameter 'q' is required")
		return
	}
	if indexPath == "" {
		jsonError(w, http.StatusBadRequest, "Query parameter 'index' is required")
		return
	}
	if strategy == "" {
		strategy = "semantic"
	}
	topK := 5
	if k, err := strconv.Atoi(q.Get("top")); err == nil && k > 0 {
		topK = k
	}

	absPath, err := filepath.Abs(indexPath)
	if err != nil {
		jsonError(w, http.StatusBadRequest, fmt.Sprintf("Invalid path: %v", err))
		return
	}
	dbPath, err := s.service.GenerateDBPath(absPath)
	if err != nil {
		jsonError(w, http.StatusBadRequest, fmt.Sprintf("Invalid path: %v", err))
		return
	}
	if _, err := os.Stat(dbPath); os.IsNotExist(err) {
		jsonError(w, http.StatusNotFound, "Index not found")
		return
	}

	// Check cache
	cacheKey := fmt.Sprintf("%s|%s|%s|%s|%d", query, indexPath, strategy, contentType, topK)
	if cached, ok := s.getCached(cacheKey); ok {
		log.Printf("🎯 Cache hit: %s", query)
		w.Header().Set("X-Cache-Status", "HIT")
		jsonOK(w, SearchResponse{Results: cached, Query: query})
		return
	}

	// Search via service
	log.Printf("🔍 Searching: %s", query)
	engineResults, err := s.service.SearchByDBPath(dbPath, query, topK, contentType, strategy)
	if err != nil {
		jsonError(w, http.StatusInternalServerError, fmt.Sprintf("Search failed: %v", err))
		return
	}

	// Apply reranking if enabled
	if s.service.Config().EnableReranker && len(engineResults) > 0 {
		engineResults = s.rerankResults(r.Context(), query, engineResults)
	}

	results := toAPIResults(engineResults)
	s.setCache(cacheKey, results)
	w.Header().Set("X-Cache-Status", "MISS")
	jsonOK(w, SearchResponse{Results: results, Query: query})
}

func (s *Server) rerankResults(ctx context.Context, query string, results []engine.SearchResult) []engine.SearchResult {
	docs := make([]string, len(results))
	for i, r := range results {
		docs[i] = r.Content
	}
	reranked, err := s.service.VoyageClient().Rerank(ctx, query, docs)
	if err != nil {
		log.Printf("⚠️  Rerank failed: %v", err)
		return results
	}
	out := make([]engine.SearchResult, 0, len(reranked))
	for _, item := range reranked {
		if item.Index >= 0 && item.Index < len(results) {
			r := results[item.Index]
			r.Distance = -item.RelevanceScore
			out = append(out, r)
		}
	}
	return out
}

func toAPIResults(engineResults []engine.SearchResult) []SearchResultItem {
	results := make([]SearchResultItem, len(engineResults))
	for i, er := range engineResults {
		var fileSize int64
		var modDate string
		if fi, err := os.Stat(er.FilePath); err == nil {
			fileSize = fi.Size()
			modDate = fi.ModTime().Format("2006-01-02 15:04:05")
		}
		pn := er.PageNumbers
		if len(pn) == 0 {
			pn = metaPageNumbers(er.TextMetadata)
		}
		results[i] = SearchResultItem{
			ChunkID: er.FilePath, Content: er.Content,
			FilePath: er.FilePath, FileName: er.FileName,
			ContentType:  metaString(er.TextMetadata, "content_type", "text"),
			Distance:     er.Distance,
			PageNumbers:  pn,
			ChunkIndex:   metaInt(er.TextMetadata, "chunk_index"),
			TotalChunks:  metaInt(er.TextMetadata, "total_chunks"),
			FileSize:     fileSize,
			ModifiedDate: modDate,
			ImageData:    string(er.ImageData),
			TextMetadata: er.TextMetadata,
		}
	}
	return results
}
