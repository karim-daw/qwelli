package service

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/karim-daw/qwelli/internal/config"
	"github.com/karim-daw/qwelli/internal/db"
	"github.com/karim-daw/qwelli/internal/engine"
	"github.com/karim-daw/qwelli/internal/voyage"
)

// Service is the single entry point for all business operations.
// It owns the DB lifecycle — callers never open databases directly.
type Service struct {
	config       *config.Config
	voyageClient voyage.ClientInterface
	engine       *engine.Engine
}

// New creates a Service with an injected Voyage client.
// This is the only constructor - dependencies must be provided.
func New(cfg *config.Config, client voyage.ClientInterface) *Service {
	eng := engine.NewEngine(client, cfg.EnableMultimodal)

	// Resolve worker counts — 0 means auto-detect (~90% of CPU cores)
	fileWorkers := cfg.ParallelWorkers
	if fileWorkers <= 0 {
		fileWorkers = config.DefaultWorkerCount()
	}
	maxEmbedConcurrency := cfg.MaxConcurrentEmbeddings
	if maxEmbedConcurrency <= 0 {
		maxEmbedConcurrency = 5
	}
	eng.SetParallelProcessing(cfg.EnableParallel, fileWorkers, maxEmbedConcurrency)

	pdfWorkers := cfg.ParallelPDFWorkers
	if pdfWorkers <= 0 {
		pdfWorkers = config.DefaultWorkerCount()
	}
	eng.SetParallelPDFProcessing(cfg.EnableParallelPDF, pdfWorkers)

	return &Service{
		config:       cfg,
		voyageClient: client,
		engine:       eng,
	}
}

// Load is a convenience function that handles the full workflow:
// loads config, creates Voyage client, and returns a Service.
func Load() (*Service, error) {
	cfg, err := config.Load()
	if err != nil {
		return nil, err
	}

	client, err := voyage.NewClient(voyage.ClientConfig{
		APIKey:               cfg.APIKey,
		EmbeddingModel:       cfg.Model,
		EmbeddingEndpoint:    cfg.Endpoint,
		MaxConcurrentBatches: cfg.MaxConcurrentEmbeddings,
		RerankModel:          cfg.RerankModel,
		RerankEndpoint:       cfg.RerankEndpoint,
	})
	if err != nil {
		return nil, fmt.Errorf("create voyage client: %w", err)
	}

	return New(cfg, client), nil
}

// Engine returns the engine (used by server for direct engine access).
func (s *Service) Engine() *engine.Engine            { return s.engine }
func (s *Service) VoyageClient() voyage.ClientInterface { return s.voyageClient }
func (s *Service) Config() *config.Config            { return s.config }

// GenerateDBPath returns the database file path for a given folder.
func (s *Service) GenerateDBPath(folderPath string) (string, error) {
	abs, err := filepath.Abs(folderPath)
	if err != nil {
		return "", fmt.Errorf("invalid path: %w", err)
	}
	return filepath.Join(s.config.IndexDir, safeDBName(abs)), nil
}

// OpenDB opens the project database for a folder path.
// Caller must call Close() on the returned DB.
func (s *Service) OpenDB(folderPath string) (*db.ProjectDB, error) {
	dbPath, err := s.GenerateDBPath(folderPath)
	if err != nil {
		return nil, err
	}
	dim, err := db.GetDimensionFromDB(dbPath)
	if err != nil {
		return nil, fmt.Errorf("index not found for %s", folderPath)
	}
	return db.OpenProjectDB(dbPath, dim)
}

// IsFileInIndex checks if a file exists in any index.
func (s *Service) IsFileInIndex(filePath string) bool {
	files, _ := filepath.Glob(filepath.Join(s.config.IndexDir, "*.db"))
	for _, dbFile := range files {
		dim, err := db.GetDimensionFromDB(dbFile)
		if err != nil {
			continue
		}
		pdb, err := db.OpenProjectDB(dbFile, dim)
		if err != nil {
			continue
		}
		_, err = pdb.GetFileByPath(filePath)
		pdb.Close()
		if err == nil {
			return true
		}
	}
	return false
}

func safeDBName(folderPath string) string {
	base := filepath.Base(folderPath)
	safe := strings.Map(func(r rune) rune {
		if r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' ||
			r >= '0' && r <= '9' || r == '-' || r == '_' {
			return r
		}
		return '_'
	}, base)
	return safe + ".db"
}
