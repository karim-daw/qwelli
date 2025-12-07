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

	count, err := eng.GetIndexStats(dbPath)
	if err != nil {
		return fmt.Errorf("failed to get stats: %w", err)
	}

	fmt.Printf("📊 Index Status: %s\n\n", absPath)
	fmt.Printf("📄 Indexed documents: %d\n", count)
	fmt.Printf("💾 Database: %s\n", dbPath)
	fmt.Printf("💽 Database size: %.2f MB\n", float64(info.Size())/(1024*1024))
	fmt.Printf("🕐 Last indexed: %s\n", info.ModTime().Format("2006-01-02 15:04:05"))

	return nil
}
