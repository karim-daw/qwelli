package voyage

// MultimodalInput represents either text or image input for embedding
type MultimodalInput struct {
	Type        string // "text" or "image"
	Text        string // For text inputs
	ImageBase64 string // For image inputs (base64-encoded, without data URI prefix)
	ImageFormat string // For image inputs: "jpeg", "png", "gif", etc. (used to add data URI prefix)
	ImagePixels int    // For image inputs: pixel count (width * height) for token calculation
}

// embeddingContentItem represents a content item in the Voyage multimodal API
type embeddingContentItem struct {
	Type        string `json:"type"` // "text", "image_base64", or "video_base64"
	Text        string `json:"text,omitempty"`
	ImageBase64 string `json:"image_base64,omitempty"`
	VideoBase64 string `json:"video_base64,omitempty"`
}

// embeddingInput represents an input object in the Voyage multimodal API
type embeddingInput struct {
	Content []embeddingContentItem `json:"content"`
}

// embeddingRequest represents the request to Voyage's multimodal embedding API
type embeddingRequest struct {
	Model  string           `json:"model"`
	Inputs []embeddingInput `json:"inputs"`
}

// embeddingResponse represents the response from Voyage's multimodal embedding API
type embeddingResponse struct {
	Object string `json:"object"`
	Data   []struct {
		Object    string    `json:"object"`
		Embedding []float64 `json:"embedding"`
		Index     int       `json:"index"`
	} `json:"data"`
	Model string `json:"model"`
	Usage struct {
		TextTokens  int `json:"text_tokens"`
		ImagePixels int `json:"image_pixels"`
		VideoPixels int `json:"video_pixels"`
		TotalTokens int `json:"total_tokens"`
	} `json:"usage"`
}

// RerankInput represents a document to be reranked
type RerankInput struct {
	Content  string
	Metadata map[string]interface{} // Optional metadata to preserve through reranking
}

// RerankResult represents a reranked document with its score
type RerankResult struct {
	Index          int
	RelevanceScore float64
	Content        string
	Metadata       map[string]interface{}
}

// rerankRequest represents the request to Voyage's rerank API
type rerankRequest struct {
	Query     string   `json:"query"`
	Documents []string `json:"documents"`
	Model     string   `json:"model"`
	TopK      *int     `json:"top_k,omitempty"` // Optional: limit number of results
}

// rerankResponse represents the response from Voyage's rerank API
type rerankResponse struct {
	Object string `json:"object"`
	Data   []struct {
		Index          int     `json:"index"`
		RelevanceScore float64 `json:"relevance_score"`
	} `json:"data"`
	Model string `json:"model"`
	Usage struct {
		TotalTokens int `json:"total_tokens"`
	} `json:"usage"`
}
