package cli

import (
	"bufio"
	"fmt"
	"path/filepath"

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

	// With PostgreSQL, we delete indexed files from the database
	// For now, this command is simplified - in the future we could delete by folder path
	fmt.Printf("⚠️  Delete functionality with PostgreSQL is not yet implemented.\n")
	fmt.Printf("   Folder path: %s\n", absPath)
	fmt.Printf("\n")
	fmt.Printf("To manually delete, connect to your PostgreSQL database and run:\n")
	fmt.Printf("   DELETE FROM files WHERE path LIKE '%s%%';\n", absPath)
	return fmt.Errorf("delete not yet implemented for PostgreSQL")
}

// deleteIndexInteractive is used by the shell command
func deleteIndexInteractive(indexPath string, reader *bufio.Reader) error {
	absPath, err := filepath.Abs(indexPath)
	if err != nil {
		return fmt.Errorf("invalid index path: %w", err)
	}

	// Delete not yet implemented for PostgreSQL
	fmt.Printf("⚠️  Delete functionality with PostgreSQL is not yet implemented.\n")
	fmt.Printf("   Folder path: %s\n", absPath)
	return fmt.Errorf("delete not yet implemented for PostgreSQL")
}
