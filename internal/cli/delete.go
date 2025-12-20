package cli

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/karim-daw/qwelli/internal/config"
	"github.com/spf13/cobra"
)

func NewDeleteCmd() *cobra.Command {
	var indexPath string
	var force bool

	cmd := &cobra.Command{
		Use:   "delete",
		Short: "Delete an indexed folder",
		Long:  "Delete the database for an indexed folder",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runDelete(indexPath, force)
		},
	}

	cmd.Flags().StringVarP(&indexPath, "index", "i", "", "Path to indexed folder (required)")
	cmd.Flags().BoolVarP(&force, "force", "f", false, "Skip confirmation prompt")
	cmd.MarkFlagRequired("index")

	return cmd
}

func runDelete(indexPath string, force bool) error {
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
		if os.IsNotExist(err) {
			return fmt.Errorf("index not found for %s", absPath)
		}
		return fmt.Errorf("failed to check index: %w", err)
	}

	// Show what will be deleted
	fmt.Printf("📊 Index to delete:\n\n")
	fmt.Printf("   📁 Folder: %s\n", absPath)
	fmt.Printf("   💾 Database: %s\n", dbPath)
	fmt.Printf("   💽 Size: %.2f MB\n", float64(info.Size())/(1024*1024))
	fmt.Println()

	// Ask for confirmation unless force flag is set
	if !force {
		fmt.Print("⚠️  Are you sure you want to delete this index? This cannot be undone. (yes/no): ")
		reader := bufio.NewReader(os.Stdin)
		response, err := reader.ReadString('\n')
		if err != nil {
			return fmt.Errorf("failed to read response: %w", err)
		}

		response = strings.ToLower(strings.TrimSpace(response))
		if response != "yes" && response != "y" {
			fmt.Println("❌ Deletion cancelled")
			return nil
		}
	}

	// Delete the database file
	if err := os.Remove(dbPath); err != nil {
		return fmt.Errorf("failed to delete index: %w", err)
	}

	fmt.Printf("✅ Index deleted successfully: %s\n", absPath)
	return nil
}

// deleteIndexInteractive is used by the shell command
func deleteIndexInteractive(indexPath string, reader *bufio.Reader) error {
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
		if os.IsNotExist(err) {
			return fmt.Errorf("index not found for %s", absPath)
		}
		return fmt.Errorf("failed to check index: %w", err)
	}

	// Show what will be deleted
	fmt.Printf("📊 Index to delete:\n\n")
	fmt.Printf("   📁 Folder: %s\n", absPath)
	fmt.Printf("   💾 Database: %s\n", dbPath)
	fmt.Printf("   💽 Size: %.2f MB\n\n", float64(info.Size())/(1024*1024))

	// Ask for confirmation
	fmt.Print("⚠️  Are you sure you want to delete this index? Type 'yes' to confirm: ")
	response, err := reader.ReadString('\n')
	if err != nil {
		return fmt.Errorf("failed to read response: %w", err)
	}

	response = strings.ToLower(strings.TrimSpace(response))
	if response != "yes" {
		return fmt.Errorf("deletion cancelled (you must type 'yes' to confirm)")
	}

	// Delete the database file
	if err := os.Remove(dbPath); err != nil {
		return fmt.Errorf("failed to delete index: %w", err)
	}

	fmt.Printf("✅ Index deleted successfully: %s\n", absPath)
	return nil
}
