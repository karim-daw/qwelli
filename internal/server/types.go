package server

import "time"

// IndexInfo represents an indexed folder in API responses.
type IndexInfo struct {
	Name          string `json:"name"`
	Path          string `json:"path"`
	DocumentCount int    `json:"documentCount"`
	LastModified  string `json:"lastModified"`
}

// SearchResultItem represents a single search result in API responses.
type SearchResultItem struct {
	ChunkID      string                 `json:"chunkId"`
	Content      string                 `json:"content"`
	FilePath     string                 `json:"filePath"`
	FileName     string                 `json:"fileName"`
	ContentType  string                 `json:"contentType"`
	Distance     float64                `json:"similarity"`
	PageNumbers  []int                  `json:"pageNumbers"`
	ChunkIndex   int                    `json:"chunkIndex,omitempty"`
	TotalChunks  int                    `json:"totalChunks,omitempty"`
	FileSize     int64                  `json:"fileSize,omitempty"`
	ModifiedDate string                 `json:"modifiedDate,omitempty"`
	ImageData    string                 `json:"imageData,omitempty"`
	TextMetadata map[string]interface{} `json:"metadata,omitempty"`
}

// SearchResponse wraps search results with the query.
type SearchResponse struct {
	Results []SearchResultItem `json:"results"`
	Query   string             `json:"query"`
}

// searchCacheEntry holds cached search results with a timestamp.
type searchCacheEntry struct {
	Results   []SearchResultItem
	Timestamp time.Time
}

// ConfigResponse is returned by GET /api/config.
type ConfigResponse struct {
	APIKeyMasked string `json:"apiKeyMasked"`
	Model        string `json:"model"`
	Endpoint     string `json:"endpoint"`
}

// UpdateConfigRequest is the POST body for /api/config/update.
// Pointer fields: nil means "keep current value".
type UpdateConfigRequest struct {
	APIKey   *string `json:"apiKey"`
	Model    *string `json:"model"`
	Endpoint *string `json:"endpoint"`
}

// UpdateConfigResponse is returned by POST /api/config/update.
type UpdateConfigResponse struct {
	Status   string   `json:"status"`
	Warnings []string `json:"warnings,omitempty"`
}
