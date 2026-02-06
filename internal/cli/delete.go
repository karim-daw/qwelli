package cli

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/karim-daw/qwelli/internal/service"
	"github.com/spf13/cobra"
)

func NewDeleteCmd() *cobra.Command {
	var indexPath string
	var force bool

	cmd := &cobra.Command{
		Use:   "delete",
		Short: "Delete an indexed folder",
		RunE: func(cmd *cobra.Command, args []string) error {
			return deleteIndex(indexPath, force, bufio.NewReader(os.Stdin))
		},
	}
	cmd.Flags().StringVarP(&indexPath, "index", "i", "", "Path to indexed folder (required)")
	cmd.Flags().BoolVarP(&force, "force", "f", false, "Skip confirmation")
	cmd.MarkFlagRequired("index")
	return cmd
}

// deleteIndexInteractive is used by the shell command.
func deleteIndexInteractive(indexPath string, reader *bufio.Reader) error {
	return deleteIndex(indexPath, false, reader)
}

func deleteIndex(indexPath string, force bool, reader *bufio.Reader) error {
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

	fmt.Printf("📊 Index to delete:\n\n")
	fmt.Printf("   📁 Folder: %s\n", absPath)
	fmt.Printf("   💾 Database: %s\n", dbPath)
	fmt.Printf("   💽 Size: %.2f MB\n\n", float64(found.SizeBytes)/(1024*1024))

	if !force {
		fmt.Print("⚠️  Delete this index? (yes/no): ")
		resp, err := reader.ReadString('\n')
		if err != nil {
			return fmt.Errorf("read response: %w", err)
		}
		resp = strings.ToLower(strings.TrimSpace(resp))
		if resp != "yes" && resp != "y" {
			fmt.Println("❌ Cancelled")
			return nil
		}
	}

	if err := svc.DeleteIndex(absPath); err != nil {
		return err
	}
	fmt.Printf("✅ Index deleted: %s\n", absPath)
	return nil
}
