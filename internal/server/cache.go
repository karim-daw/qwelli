package server

import (
	"fmt"
	"time"
)

// SearchCacheEntry represents a cached search result
type SearchCacheEntry struct {
	Results   []SearchResultItem
	Timestamp time.Time
}

// Cache-related methods

func (s *Server) cleanupExpiredCache() {
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		s.cacheMutex.Lock()
		now := time.Now()
		for key, entry := range s.searchCache {
			if now.Sub(entry.Timestamp) > s.cacheTTL {
				delete(s.searchCache, key)
			}
		}
		s.cacheMutex.Unlock()
	}
}

func (s *Server) generateCacheKey(query, indexPath, strategy, contentType string, topK int) string {
	return fmt.Sprintf("%s|%s|%s|%s|%d", query, indexPath, strategy, contentType, topK)
}

func (s *Server) getCachedResults(cacheKey string) ([]SearchResultItem, bool) {
	s.cacheMutex.RLock()
	defer s.cacheMutex.RUnlock()

	entry, ok := s.searchCache[cacheKey]
	if !ok {
		return nil, false
	}

	// Check if entry is still valid
	if time.Since(entry.Timestamp) > s.cacheTTL {
		return nil, false
	}

	return entry.Results, true
}

func (s *Server) setCachedResults(cacheKey string, results []SearchResultItem) {
	s.cacheMutex.Lock()
	defer s.cacheMutex.Unlock()

	s.searchCache[cacheKey] = &SearchCacheEntry{
		Results:   results,
		Timestamp: time.Now(),
	}
}
