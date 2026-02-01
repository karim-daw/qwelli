package cli

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/karim-daw/qwelli/internal/config"
	"github.com/karim-daw/qwelli/internal/engine"
	"github.com/karim-daw/qwelli/internal/voyage"
	"github.com/spf13/cobra"
)

func NewIndexCmd() *cobra.Command {
	var incremental bool
	var multimodal bool

	cmd := &cobra.Command{
		Use:   "index <folder>",
		Short: "Index a folder for semantic search",
		Long:  "Recursively index all files in a folder and generate embeddings",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runIndex(args[0], incremental, multimodal)
		},
	}

	cmd.Flags().BoolVarP(&incremental, "incremental", "i", false, "Only index new or changed files")
	cmd.Flags().BoolVarP(&multimodal, "multimodal", "m", false, "Enable multimodal indexing (extract and index images from PDFs)")

	return cmd
}

func runIndex(folderPath string, incremental bool, multimodal bool) error {

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

	// Use configured database URL
	dbPath := cfg.DatabaseURL

	fmt.Printf("📂 Indexing folder: %s\n", absPath)

	// Check if folder exists
	if _, err := os.Stat(absPath); os.IsNotExist(err) {
		return fmt.Errorf("folder does not exist: %s", absPath)
	}

	fmt.Printf("🗄️  Database: PostgreSQL with pgvector\n\n")

	// Create voyage client (always multimodal now)
	voyageClient, err := voyage.NewClient(voyage.ClientConfig{
		APIKey:            cfg.VoyageAPIKey,
		EmbeddingModel:    cfg.VoyageModel,
		EmbeddingEndpoint: cfg.VoyageEmbeddingEndpoint,
		RerankModel:       cfg.VoyageRerankModel,
		RerankEndpoint:    cfg.VoyageRerankEndpoint,
	})
	if err != nil {
		return fmt.Errorf("failed to create voyage client: %w", err)
	}

	// Create engine (always multimodal now)
	eng := engine.NewEngine(voyageClient, true)

	if true {
		fmt.Printf("🖼️  Multimodal indexing enabled (text + images)\n")
	}

	// If incremental, show status first
	if incremental {
		// Check if database exists
		if _, err := os.Stat(dbPath); os.IsNotExist(err) {
			fmt.Println("⚠️  Database not found. Performing full index instead of incremental.")
			incremental = false
		} else {
			status, err := eng.GetIndexStatus(dbPath, absPath)
			if err != nil {
				return fmt.Errorf("failed to get index status: %w", err)
			}

			if len(status.ToAdd) == 0 && len(status.ToUpdate) == 0 && len(status.ToDelete) == 0 {
				fmt.Println("✓ Index is already up to date. No changes detected.")
				return nil
			}

			// Show summary
			fmt.Printf("📊 Detected changes: %d to add, %d to update, %d to delete\n\n",
				len(status.ToAdd), len(status.ToUpdate), len(status.ToDelete))
		}
	}

	// Index with progress
	start := time.Now()
	var lastProgress string

	var textChunks, imageChunks int
	err = eng.IndexFolder(context.Background(), absPath, dbPath, incremental, func(current, total int, filename string) {
		progress := fmt.Sprintf("📄 Processing %d/%d: %s", current, total, filepath.Base(filename))
		progress += fmt.Sprintf(" (text: %d, images: %d)", textChunks, imageChunks)

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

	elapsed := time.Since(start)
	fmt.Printf("✅ Total indexing time: %v (includes file processing, embedding generation, and HNSW index rebuild)\n", elapsed)
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
