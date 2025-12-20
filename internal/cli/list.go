package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/karim-daw/qwelli/internal/config"
	"github.com/karim-daw/qwelli/internal/engine"
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

	// List all .db files in index directory
	files, err := filepath.Glob(filepath.Join(cfg.IndexDir, "*.db"))
	if err != nil {
		return fmt.Errorf("failed to list indexes: %w", err)
	}

	if len(files) == 0 {
		fmt.Println("No indexed folders found.")
		fmt.Println("Run 'qwelli index <folder>' to create your first index.")
		return nil
	}

	fmt.Printf("📚 Indexed folders (%d):\n\n", len(files))

	eng := engine.NewEngine(cfg.APIKey, cfg.Model, cfg.Endpoint)

	for i, dbFile := range files {
		// Get stats
		count, err := eng.GetIndexStats(dbFile)
		if err != nil {
			count = 0
		}

		// Get folder path from metadata
		folderPath, err := eng.GetFolderPath(dbFile)
		if err != nil || folderPath == "" {
			// Fallback to database name if no metadata
			dbName := filepath.Base(dbFile)
			folderPath = strings.TrimSuffix(dbName, ".db")
		}

		// Get file info
		info, _ := os.Stat(dbFile)

		fmt.Printf("%d. %s\n", i+1, folderPath)
		fmt.Printf("   📄 Documents: %d\n", count)
		fmt.Printf("   💾 Database: %s\n", dbFile)
		if info != nil {
			fmt.Printf("   🕐 Last modified: %s\n", info.ModTime().Format("2006-01-02 15:04:05"))
		}
		fmt.Println()
	}

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

	dbName := generateDBName(absPath)
	dbPath := filepath.Join(cfg.IndexDir, dbName)

	// Check if database exists
	info, err := os.Stat(dbPath)
	if err != nil {
		return fmt.Errorf("index not found for %s. Run 'qwelli index %s' first", absPath, absPath)
	}

	eng := engine.NewEngine(cfg.APIKey, cfg.Model, cfg.Endpoint)

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
	fmt.Printf("💾 Database: %s\n", dbPath)
	fmt.Printf("💽 Database size: %.2f MB\n", float64(info.Size())/(1024*1024))
	fmt.Printf("🕐 Last indexed: %s\n\n", info.ModTime().Format("2006-01-02 15:04:05"))

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
