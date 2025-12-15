package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/karim-daw/qwelli/internal/config"
	"github.com/karim-daw/qwelli/internal/engine"
	"github.com/spf13/cobra"
)

func NewIndexCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "index <folder>",
		Short: "Index a folder for semantic search",
		Long:  "Recursively index all files in a folder and generate embeddings",
		Args:  cobra.ExactArgs(1),
		RunE:  runIndex,
	}
}

func runIndex(cmd *cobra.Command, args []string) error {
	folderPath := args[0]

	// Resolve to absolute path
	absPath, err := filepath.Abs(folderPath)
	if err != nil {
		return fmt.Errorf("invalid folder path: %w", err)
	}

	// Check if folder exists
	if _, err := filepath.Abs(absPath); err != nil {
		return fmt.Errorf("folder does not exist: %s", absPath)
	}

	// Load config
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("%w", err)
	}

	// Ensure index directory exists
	if err := cfg.EnsureIndexDir(); err != nil {
		return err
	}

	// Generate database name from folder path
	dbName := generateDBName(absPath)
	dbPath := filepath.Join(cfg.IndexDir, dbName)

	fmt.Printf("📂 Indexing folder: %s\n", absPath)
	fmt.Printf("💾 Database: %s\n\n", dbPath)

	// Check if the index exists
	if _, err := os.Stat(dbPath); os.IsNotExist(err) {
		return fmt.Errorf("index not found: %s", dbPath)
	}
	// Create engine
	eng := engine.NewEngine(cfg.APIKey, cfg.Model, cfg.Endpoint)

	// Index with progress
	start := time.Now()
	var lastProgress string

	err = eng.IndexFolder(absPath, dbPath, func(current, total int, filename string) {
		progress := fmt.Sprintf("📄 Processing %d/%d: %s", current, total, filepath.Base(filename))

		// Clear previous line and print new progress
		fmt.Print("\r" + strings.Repeat(" ", len(lastProgress)) + "\r")
		fmt.Print(progress)
		lastProgress = progress
	})

	// Clear progress line
	fmt.Print("\r" + strings.Repeat(" ", len(lastProgress)) + "\r")

	if err != nil {
		return fmt.Errorf("indexing failed: %w", err)
	}

	fmt.Printf("✅ Indexing completed in %v\n", time.Since(start))
	fmt.Printf("🔍 You can now search with: qwelli search \"your query\" --index %s\n", absPath)

	return nil
}

// generateDBName creates a safe database filename from a folder path
func generateDBName(folderPath string) string {
	// Use the folder name + hash for uniqueness
	base := filepath.Base(folderPath)

	// Clean the name to be filesystem-safe
	safe := strings.Map(func(r rune) rune {
		if r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || r >= '0' && r <= '9' || r == '-' || r == '_' {
			return r
		}
		return '_'
	}, base)

	return safe + ".db"
}
