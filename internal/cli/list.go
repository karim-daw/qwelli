package cli

import (
	"fmt"
	"path/filepath"

	"github.com/karim-daw/qwelli/internal/config"
	"github.com/karim-daw/qwelli/internal/engine"
	"github.com/karim-daw/qwelli/internal/voyage"
	"github.com/spf13/cobra"
)

func NewListCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List all indexed folders",
		Long:  "Show all folders that have been indexed",
		RunE:  runList,
	}
}

func runList(cmd *cobra.Command, args []string) error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}

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

	eng := engine.NewEngine(voyageClient, true)

	// With PostgreSQL, we have a single database - show its stats
	count, err := eng.GetIndexStats(cfg.DatabaseURL)
	if err != nil {
		return fmt.Errorf("failed to get database stats: %w", err)
	}

	if count == 0 {
		fmt.Println("No documents indexed yet.")
		fmt.Println("Run 'qwelli index <folder>' to start indexing.")
		return nil
	}

	// Get folder path from metadata
	folderPath, err := eng.GetFolderPath(cfg.DatabaseURL)
	if err != nil || folderPath == "" {
		folderPath = "default"
	}

	fmt.Printf("📚 Indexed database:\n\n")
	fmt.Printf("   📁 Folder: %s\n", folderPath)
	fmt.Printf("   📄 Documents: %d\n", count)
	fmt.Printf("   🗄️  Database: PostgreSQL with pgvector\n")
	fmt.Println()

	return nil
}

func NewStatusCmd() *cobra.Command {
	var indexPath string

	cmd := &cobra.Command{
		Use:   "status",
		Short: "Show status of an indexed folder",
		Long:  "Display statistics about an indexed folder",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runStatus(indexPath)
		},
	}

	cmd.Flags().StringVarP(&indexPath, "index", "i", "", "Path to indexed folder (required)")
	cmd.MarkFlagRequired("index")

	return cmd
}

func runStatus(indexPath string) error {
	absPath, err := filepath.Abs(indexPath)
	if err != nil {
		return fmt.Errorf("invalid index path: %w", err)
	}

	cfg, err := config.Load()
	if err != nil {
		return err
	}

	dbPath := cfg.DatabaseURL

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

	eng := engine.NewEngine(voyageClient, true)

	// Get index status with pending changes
	status, err := eng.GetIndexStatus(dbPath, absPath)
	if err != nil {
		return fmt.Errorf("failed to get index status: %w", err)
	}

	count, err := eng.GetIndexStats(dbPath)
	if err != nil {
		return fmt.Errorf("failed to get stats: %w", err)
	}

	fmt.Printf("📊 Index Status: %s\n\n", absPath)
	fmt.Printf("📄 Indexed chunks: %d\n", count)
	fmt.Printf("📁 Total files: %d\n", status.Total)
	fmt.Printf("✅ Up to date: %d\n", status.UpToDate)
	fmt.Printf("🗄️  Database: PostgreSQL with pgvector\n")
	fmt.Printf("🔄 Reranker: %s\n\n", map[bool]string{true: "enabled", false: "disabled"}[cfg.EnableReranker])

	// Show pending changes (git-style)
	if len(status.ToAdd) > 0 {
		fmt.Printf("Files to add (%d):\n", len(status.ToAdd))
		for _, f := range status.ToAdd {
			fmt.Printf("  + %s (%s, %d bytes)\n", filepath.Base(f.Path), f.FileType, f.Size)
		}
		fmt.Println()
	}

	if len(status.ToUpdate) > 0 {
		fmt.Printf("Files to update (%d):\n", len(status.ToUpdate))
		for _, f := range status.ToUpdate {
			fmt.Printf("  ~ %s (%s, %d bytes)\n", filepath.Base(f.Path), f.FileType, f.Size)
		}
		fmt.Println()
	}

	if len(status.ToDelete) > 0 {
		fmt.Printf("Files to delete (%d):\n", len(status.ToDelete))
		for _, f := range status.ToDelete {
			fmt.Printf("  - %s (%s)\n", filepath.Base(f.Path), f.FileType)
		}
		fmt.Println()
	}

	if len(status.ToAdd) == 0 && len(status.ToUpdate) == 0 && len(status.ToDelete) == 0 {
		fmt.Println("✓ Index is up to date. No changes detected.")
	} else {
		fmt.Printf("Summary: %d to add, %d to update, %d to delete\n",
			len(status.ToAdd), len(status.ToUpdate), len(status.ToDelete))
		fmt.Printf("Run 'qwelli index --incremental %s' to apply changes\n", absPath)
	}

	return nil
}
