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
	voyageClient *voyage.Client
	engine       *engine.Engine
}

// New creates a Service from config. This is the only constructor.
func New(cfg *config.Config) (*Service, error) {
	vc, err := voyage.NewClient(voyage.ClientConfig{
		APIKey:            cfg.APIKey,
		EmbeddingModel:    cfg.Model,
		EmbeddingEndpoint: cfg.Endpoint,
		RerankModel:       cfg.RerankModel,
		RerankEndpoint:    cfg.RerankEndpoint,
	})
	if err != nil {
		return nil, fmt.Errorf("create voyage client: %w", err)
	}
	return &Service{
		config:       cfg,
		voyageClient: vc,
		engine:       engine.NewEngine(vc, cfg.EnableMultimodal),
	}, nil
}

// Load is a convenience that loads config and creates a Service in one call.
// Eliminates the 2-step config.Load() + service.New() boilerplate.
func Load() (*Service, error) {
	cfg, err := config.Load()
	if err != nil {
		return nil, err
	}
	return New(cfg)
}

// Engine returns the engine (used by server for direct engine access).
func (s *Service) Engine() *engine.Engine     { return s.engine }
func (s *Service) VoyageClient() *voyage.Client { return s.voyageClient }
func (s *Service) Config() *config.Config       { return s.config }

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
