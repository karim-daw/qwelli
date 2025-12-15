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

type Engine struct {
	apiKey   string
	model    string
	endpoint string
}

func NewEngine(apiKey, model, endpoint string) *Engine {
	return &Engine{apiKey: apiKey, model: model, endpoint: endpoint}
}

type SearchResult struct {
	FilePath string
	FileName string
	Distance float64
	Content  string
}

func (e *Engine) IndexFolder(folderPath, dbPath string, progressCallback func(current, total int, filename string)) error {
	// Detect dimension
	embedder, err := indexer.NewEmbedder(e.apiKey, e.model, e.endpoint)
	if err != nil {
		return err
	}

	testEmbed, err := embedder.Embed("test")
	if err != nil {
		return err
	}
	dimension := len(testEmbed)

	// Check if model changed - if so, delete DB to start fresh
	if existingDim, err := db.GetDimensionFromDB(dbPath); err == nil {
		tempDB, _ := db.OpenProjectDB(dbPath, existingDim)
		if tempDB != nil {
			storedModel, _ := tempDB.GetMetadata("model")
			tempDB.Close()
			if storedModel != "" && storedModel != e.model {
				fmt.Printf("⚠️  Model changed from '%s' to '%s' - recreating database...\n", storedModel, e.model)
				os.Remove(dbPath)
			}
		}
	}

	projectDB, err := db.OpenProjectDB(dbPath, dimension)
	if err != nil {
		return err
	}
	defer projectDB.Close()

	// Store metadata
	projectDB.SetMetadata("dimension", fmt.Sprintf("%d", dimension))
	projectDB.SetMetadata("model", e.model)
	projectDB.SetMetadata("folder_path", folderPath)

	// Scan and process files
	files, err := scanFolder(folderPath)
	if err != nil {
		return err
	}
	if len(files) == 0 {
		return fmt.Errorf("no files found in %s", folderPath)
	}

	var docs []db.Document
	var contents []string

	for i, f := range files {
		if progressCallback != nil {
			progressCallback(i+1, len(files), f)
		}

		info, err := os.Stat(f)
		if err != nil || info.Size() > 500*1024 {
			continue
		}

		content, err := os.ReadFile(f)
		if err != nil {
			continue
		}

		metadata, _ := json.Marshal(map[string]string{
			"indexed_at": time.Now().Format(time.RFC3339),
			"file_name":  filepath.Base(f),
		})

		docs = append(docs, db.Document{
			ID:         generateDocID(f),
			Path:       f,
			FileType:   getFileType(f),
			ModifiedAt: info.ModTime(),
			Size:       info.Size(),
			Metadata:   metadata,
			Content:    string(content),
		})
		contents = append(contents, string(content))
	}

	// Insert documents
	for _, doc := range docs {
		if err := projectDB.InsertDocument(doc); err != nil {
			return err
		}
	}

	// Generate and insert embeddings
	embeddings, err := embedder.EmbedBatch(contents)
	if err != nil {
		return err
	}

	for i, vec := range embeddings {
		if err := projectDB.InsertEmbedding(db.Embedding{DocID: docs[i].ID, Vector: vec}); err != nil {
			return err
		}
	}

	return projectDB.BuildHNSWIndex()
}

func (e *Engine) Search(query string, dbPath string, topK int) ([]SearchResult, error) {
	embedder, err := indexer.NewEmbedder(e.apiKey, e.model, e.endpoint)
	if err != nil {
		return nil, err
	}

	queryVec, err := embedder.Embed(query)
	if err != nil {
		return nil, err
	}

	projectDB, err := db.OpenProjectDB(dbPath, len(queryVec))
	if err != nil {
		return nil, err
	}
	defer projectDB.Close()

	results, err := projectDB.SearchANN(queryVec, topK)
	if err != nil {
		return nil, err
	}

	var out []SearchResult
	for _, r := range results {
		doc, err := projectDB.GetDocument(r.DocID)
		if err != nil {
			continue
		}
		out = append(out, SearchResult{
			FilePath: doc.Path,
			FileName: filepath.Base(doc.Path),
			Distance: r.Distance,
			Content:  strings.TrimSpace(doc.Content),
		})
	}
	return out, nil
}

func (e *Engine) GetIndexStats(dbPath string) (int, error) {
	dim, err := db.GetDimensionFromDB(dbPath)
	if err != nil {
		return 0, err
	}
	projectDB, err := db.OpenProjectDB(dbPath, dim)
	if err != nil {
		return 0, err
	}
	defer projectDB.Close()

	embeddings, err := projectDB.LoadAllEmbeddings()
	return len(embeddings), err
}

func (e *Engine) GetFolderPath(dbPath string) (string, error) {
	dim, err := db.GetDimensionFromDB(dbPath)
	if err != nil {
		return "", err
	}
	projectDB, err := db.OpenProjectDB(dbPath, dim)
	if err != nil {
		return "", err
	}
	defer projectDB.Close()
	return projectDB.GetMetadata("folder_path")
}

// Helper functions

func scanFolder(root string) ([]string, error) {
	var files []string
	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			name := d.Name()
			if strings.HasPrefix(name, ".") || name == "node_modules" || name == "vendor" {
				return filepath.SkipDir
			}
			return nil
		}
		if strings.HasPrefix(d.Name(), ".") || !isTextFile(path) {
			return nil
		}
		files = append(files, path)
		return nil
	})
	return files, err
}

func isTextFile(path string) bool {
	ext := strings.ToLower(filepath.Ext(path))
	textExts := map[string]bool{
		".txt": true, ".md": true, ".go": true, ".py": true, ".js": true, ".ts": true,
		".tsx": true, ".jsx": true, ".java": true, ".c": true, ".cpp": true, ".h": true,
		".rs": true, ".rb": true, ".php": true, ".cs": true, ".swift": true,
		".html": true, ".css": true, ".scss": true, ".yaml": true, ".yml": true,
		".toml": true, ".sh": true, ".sql": true, ".proto": true, ".graphql": true,
	}
	return textExts[ext]
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
