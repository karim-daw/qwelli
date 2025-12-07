package engine

import (
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/karim-daw/qwelli/internal/db"
	"github.com/karim-daw/qwelli/internal/indexer"
)

// Engine manages indexing and searching operations
type Engine struct {
	apiKey   string
	model    string
	endpoint string
}

// NewEngine creates a new engine instance
func NewEngine(apiKey, model, endpoint string) *Engine {
	return &Engine{
		apiKey:   apiKey,
		model:    model,
		endpoint: endpoint,
	}
}

// IndexFolder indexes all files in a folder
func (e *Engine) IndexFolder(folderPath, dbPath string, progressCallback func(current, total int, filename string)) error {
	// Check if database already exists with stored dimension and model
	dimension, err := db.GetDimensionFromDB(dbPath)
	storedModel := ""
	needsRedetection := false

	if err != nil {
		// Database doesn't exist - need to detect
		needsRedetection = true
	} else {
		// Database exists - check if model changed
		tempDB, err := db.OpenProjectDB(dbPath, dimension)
		if err == nil {
			storedModel, _ = tempDB.GetMetadata("model")
			tempDB.Close()

			if storedModel != e.model {
				// Model changed! Need to re-detect dimension
				needsRedetection = true
				fmt.Printf("⚠️  Model changed from '%s' to '%s' - re-detecting dimension...\n", storedModel, e.model)
			}
		}
	}

	if needsRedetection {
		// Auto-detect dimension for new database or model change
		embedder, err := indexer.NewEmbedder(e.apiKey, e.model, e.endpoint)
		if err != nil {
			return fmt.Errorf("failed to initialize embedder: %w", err)
		}

		// Get embedding dimension once
		testEmbed, err := embedder.Embed("test")
		if err != nil {
			return fmt.Errorf("failed to get embedding dimension: %w", err)
		}
		dimension = len(testEmbed)
	}

	// Open/create database (will use stored dimension if exists)
	projectDB, err := db.OpenProjectDB(dbPath, dimension)
	if err != nil {
		return fmt.Errorf("failed to open database: %w", err)
	}
	defer projectDB.Close()

	// Store metadata: dimension, model, and folder path
	if err := projectDB.SetMetadata("dimension", fmt.Sprintf("%d", dimension)); err != nil {
		return fmt.Errorf("failed to store dimension: %w", err)
	}
	if err := projectDB.SetMetadata("model", e.model); err != nil {
		return fmt.Errorf("failed to store model: %w", err)
	}
	if err := projectDB.SetMetadata("folder_path", folderPath); err != nil {
		return fmt.Errorf("failed to store folder path: %w", err)
	}

	// Initialize embedder for indexing
	embedder, err := indexer.NewEmbedder(e.apiKey, e.model, e.endpoint)
	if err != nil {
		return fmt.Errorf("failed to initialize embedder: %w", err)
	}

	// Scan folder
	files, err := scanFolder(folderPath)
	if err != nil {
		return fmt.Errorf("failed to scan folder: %w", err)
	}

	if len(files) == 0 {
		return fmt.Errorf("no files found in %s", folderPath)
	}

	// Prepare documents
	var docs []db.Document
	var fileContents []string
	var fileNames []string

	for i, file := range files {
		if progressCallback != nil {
			progressCallback(i+1, len(files), file.Path)
		}

		// Get file info first to check size
		info, err := os.Stat(file.Path)
		if err != nil {
			continue
		}

		// Skip files larger than 500KB (likely generated files)
		const maxFileSize = 500 * 1024 // 500KB
		if info.Size() > maxFileSize {
			continue
		}

		// Read file content
		content, err := os.ReadFile(file.Path)
		if err != nil {
			continue // Skip files we can't read
		}

		// Generate document ID from file path
		docID := generateDocID(file.Path)

		// Determine file type
		fileType := getFileType(file.Path)

		// Create metadata
		metadata := map[string]interface{}{
			"indexed_at": time.Now().Format(time.RFC3339),
			"file_name":  filepath.Base(file.Path),
			"folder":     folderPath,
		}
		metadataJSON, _ := json.Marshal(metadata)

		// Create document
		doc := db.Document{
			ID:         docID,
			Path:       file.Path,
			FileType:   fileType,
			ModifiedAt: info.ModTime(),
			Size:       info.Size(),
			Metadata:   metadataJSON,
			Content:    string(content),
		}

		docs = append(docs, doc)
		fileContents = append(fileContents, string(content))
		fileNames = append(fileNames, filepath.Base(file.Path))
	}

	// Insert documents
	for _, doc := range docs {
		if err := projectDB.InsertDocument(doc); err != nil {
			return fmt.Errorf("failed to insert document: %w", err)
		}
	}

	// Generate embeddings in batch
	embeddings, err := embedder.EmbedBatch(fileContents)
	if err != nil {
		return fmt.Errorf("failed to generate embeddings: %w", err)
	}

	// Get or create model_id
	modelID, err := projectDB.GetOrCreateModel(e.model, dimension)
	if err != nil {
		return fmt.Errorf("failed to get model: %w", err)
	}

	// Insert embeddings
	for i, embedding := range embeddings {
		emb := db.Embedding{
			DocID:   docs[i].ID,
			ModelID: modelID,
			Vector:  embedding,
		}
		if err := projectDB.InsertEmbedding(emb); err != nil {
			return fmt.Errorf("failed to insert embedding: %w", err)
		}
	}

	// Build HNSW index
	if err := projectDB.BuildHNSWIndex(); err != nil {
		return fmt.Errorf("failed to build index: %w", err)
	}

	return nil
}

// SearchResult represents a search result
type SearchResult struct {
	FilePath string
	FileName string
	Distance float64
	Content  string
}

// Search performs a semantic search
func (e *Engine) Search(query string, dbPath string, topK int) ([]SearchResult, error) {
	// Initialize embedder for query
	embedder, err := indexer.NewEmbedder(e.apiKey, e.model, e.endpoint)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize embedder: %w", err)
	}

	// Generate query embedding
	queryVector, err := embedder.Embed(query)
	if err != nil {
		return nil, fmt.Errorf("failed to embed query: %w", err)
	}

	dimension := len(queryVector)

	// Open database
	projectDB, err := db.OpenProjectDB(dbPath, dimension)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}
	defer projectDB.Close()

	// Get model_id for current model (will verify dimension matches)
	modelID, err := projectDB.GetOrCreateModel(e.model, dimension)
	if err != nil {
		return nil, fmt.Errorf("failed to get model: %w", err)
	}

	// Search using model_id
	results, err := projectDB.SearchANN(queryVector, topK, modelID)
	if err != nil {
		return nil, fmt.Errorf("search failed: %w", err)
	}

	// Convert to SearchResult
	searchResults := make([]SearchResult, 0, len(results))
	for _, result := range results {
		doc, err := projectDB.GetDocument(result.DocID)
		if err != nil {
			continue
		}

		searchResults = append(searchResults, SearchResult{
			FilePath: doc.Path,
			FileName: filepath.Base(doc.Path),
			Distance: result.Distance,
			Content:  strings.TrimSpace(doc.Content),
		})
	}

	return searchResults, nil
}

// GetIndexStats returns statistics about an index
func (e *Engine) GetIndexStats(dbPath string) (int, error) {
	// Get dimension from database metadata (no API call needed!)
	dimension, err := db.GetDimensionFromDB(dbPath)
	if err != nil {
		return 0, fmt.Errorf("failed to get dimension: %w", err)
	}

	projectDB, err := db.OpenProjectDB(dbPath, dimension)
	if err != nil {
		return 0, fmt.Errorf("failed to open database: %w", err)
	}
	defer projectDB.Close()

	embeddings, err := projectDB.LoadAllEmbeddings()
	if err != nil {
		return 0, fmt.Errorf("failed to load embeddings: %w", err)
	}

	return len(embeddings), nil
}

// GetFolderPath retrieves the original folder path from database metadata
func (e *Engine) GetFolderPath(dbPath string) (string, error) {
	dimension, err := db.GetDimensionFromDB(dbPath)
	if err != nil {
		return "", fmt.Errorf("failed to get dimension: %w", err)
	}

	projectDB, err := db.OpenProjectDB(dbPath, dimension)
	if err != nil {
		return "", fmt.Errorf("failed to open database: %w", err)
	}
	defer projectDB.Close()

	folderPath, err := projectDB.GetMetadata("folder_path")
	if err != nil {
		return "", err
	}

	return folderPath, nil
}

// Helper functions

func scanFolder(root string) ([]struct{ Path string }, error) {
	var files []struct{ Path string }

	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		// Skip hidden directories (.git, .cursor, node_modules, etc.)
		if d.IsDir() {
			name := d.Name()
			if strings.HasPrefix(name, ".") || name == "node_modules" || name == "vendor" {
				return filepath.SkipDir
			}
			return nil
		}

		// Skip hidden files
		if strings.HasPrefix(d.Name(), ".") {
			return nil
		}

		// Skip binary/non-text files by extension
		if !isTextFile(path) {
			return nil
		}

		files = append(files, struct{ Path string }{Path: path})
		return nil
	})

	return files, err
}

// isTextFile checks if a file is likely a text file based on extension
func isTextFile(path string) bool {
	ext := strings.ToLower(filepath.Ext(path))

	// Only allow explicit text file extensions - no JSON, XML, or files without extensions
	textExtensions := map[string]bool{
		// Plain text
		".txt": true, ".md": true, ".markdown": true,
		// Code
		".go": true, ".py": true, ".js": true, ".ts": true, ".tsx": true, ".jsx": true,
		".java": true, ".c": true, ".cpp": true, ".h": true, ".hpp": true,
		".rs": true, ".rb": true, ".php": true, ".cs": true, ".swift": true,
		// Web
		".html": true, ".css": true, ".scss": true, ".sass": true, ".less": true,
		// Config (small only)
		".yaml": true, ".yml": true, ".toml": true, ".env": true,
		// Scripts
		".sh": true, ".bash": true, ".zsh": true,
		// Other
		".sql": true, ".proto": true, ".graphql": true, ".gql": true,
	}

	return textExtensions[ext]
}

func generateDocID(path string) string {
	hash := md5.Sum([]byte(path))
	return hex.EncodeToString(hash[:])
}

func getFileType(path string) string {
	ext := strings.ToLower(filepath.Ext(path))
	if ext == "" {
		return "unknown"
	}
	return ext[1:]
}
