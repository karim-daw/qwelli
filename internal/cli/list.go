package cli

import (
	"fmt"
	"path/filepath"

	"github.com/karim-daw/qwelli/internal/service"
	"github.com/spf13/cobra"
)

func NewListCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List all indexed folders",
		RunE:  runList,
	}
}

func runList(cmd *cobra.Command, args []string) error {
	svc, err := service.Load()
	if err != nil {
		return err
	}

	indexes, err := svc.ListIndexes()
	if err != nil {
		return fmt.Errorf("list indexes: %w", err)
	}
	if len(indexes) == 0 {
		fmt.Println("No indexed folders found.")
		fmt.Println("Run 'qwelli index <folder>' to create your first index.")
		return nil
	}

	fmt.Printf("📚 Indexed folders (%d):\n\n", len(indexes))
	for i, idx := range indexes {
		fmt.Printf("%d. %s\n", i+1, idx.FolderPath)
		fmt.Printf("   📄 Documents: %d\n", idx.DocumentCount)
		fmt.Printf("   💾 Database: %s\n", idx.DBPath)
		if !idx.LastModified.IsZero() {
			fmt.Printf("   🕐 Last modified: %s\n", idx.LastModified.Format("2006-01-02 15:04:05"))
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
		return fmt.Errorf("invalid path: %w", err)
	}

	svc, err := service.Load()
	if err != nil {
		return err
	}

	dbPath, err := svc.GenerateDBPath(absPath)
	if err != nil {
		return err
	}

	// Find matching index
	indexes, err := svc.ListIndexes()
	if err != nil {
		return fmt.Errorf("check index: %w", err)
	}
	var found *service.IndexInfo
	for _, idx := range indexes {
		if idx.DBPath == dbPath {
			found = &idx
			break
		}
	}
	if found == nil {
		return fmt.Errorf("index not found for %s", absPath)
	}

	status, err := svc.GetIndexStatus(absPath)
	if err != nil {
		return fmt.Errorf("get status: %w", err)
	}
	count, _ := svc.GetIndexStats(absPath)

	fmt.Printf("📊 Index Status: %s\n\n", absPath)
	fmt.Printf("📄 Indexed chunks: %d\n", count)
	fmt.Printf("📁 Total files: %d\n", status.Total)
	fmt.Printf("✅ Up to date: %d\n", status.UpToDate)
	fmt.Printf("💾 Database: %s\n", dbPath)
	fmt.Printf("💽 Size: %.2f MB\n", float64(found.SizeBytes)/(1024*1024))
	fmt.Printf("🕐 Last indexed: %s\n\n", found.LastModified.Format("2006-01-02 15:04:05"))

	// Show pending changes
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
		fmt.Println("✓ Index is up to date.")
	} else {
		fmt.Printf("Summary: %d to add, %d to update, %d to delete\n", len(status.ToAdd), len(status.ToUpdate), len(status.ToDelete))
		fmt.Printf("Run 'qwelli index --incremental %s' to apply changes\n", absPath)
	}
	return nil
}
