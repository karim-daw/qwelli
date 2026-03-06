package cli

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/karim-daw/qwelli/internal/service"
	"github.com/spf13/cobra"
)

//

func NewIndexCmd() *cobra.Command {
	var incremental, multimodal bool

	cmd := &cobra.Command{
		Use:   "index <folder>",
		Short: "Index a folder for semantic search",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runIndex(args[0], incremental, multimodal)
		},
	}
	cmd.Flags().BoolVarP(&incremental, "incremental", "i", false, "Only index new or changed files")
	cmd.Flags().BoolVarP(&multimodal, "multimodal", "m", false, "Enable multimodal indexing")
	return cmd
}

func runIndex(folderPath string, incremental, multimodal bool) error {
	svc, err := service.Load()
	if err != nil {
		return err
	}
	if err := svc.Config().EnsureIndexDir(); err != nil {
		return err
	}

	absPath, err := filepath.Abs(folderPath)
	if err != nil {
		return fmt.Errorf("invalid path: %w", err)
	}
	if _, err := os.Stat(absPath); os.IsNotExist(err) {
		return fmt.Errorf("folder does not exist: %s", absPath)
	}

	dbPath, err := svc.GenerateDBPath(absPath)
	if err != nil {
		return err
	}

	fmt.Printf("📂 Indexing folder: %s\n", absPath)
	fmt.Printf("💾 Database: %s\n\n", dbPath)

	enableMultimodal := multimodal || svc.Config().EnableMultimodal
	if enableMultimodal {
		fmt.Println("🖼️  Multimodal indexing enabled")
	}

	// If incremental, show status first
	if incremental {
		if _, err := os.Stat(dbPath); os.IsNotExist(err) {
			fmt.Println("⚠️  Database not found. Performing full index instead.")
			incremental = false
		} else {
			status, err := svc.GetIndexStatus(absPath)
			if err != nil {
				return fmt.Errorf("get index status: %w", err)
			}
			if len(status.ToAdd) == 0 && len(status.ToUpdate) == 0 && len(status.ToDelete) == 0 {
				fmt.Println("✓ Index is already up to date.")
				return nil
			}
			fmt.Printf("📊 Changes: %d to add, %d to update, %d to delete\n\n",
				len(status.ToAdd), len(status.ToUpdate), len(status.ToDelete))
		}
	}

	start := time.Now()
	var lastProgress string
	opts := service.IndexOptions{Incremental: incremental, Multimodal: enableMultimodal}

	indexFn := svc.CreateIndex
	if incremental {
		indexFn = svc.UpdateIndex
	}

	err = indexFn(context.Background(), absPath, opts, func(current, total int, filename string) {
		progress := fmt.Sprintf("📄 Processing %d/%d: %s", current, total, filepath.Base(filename))
		fmt.Print("\r" + strings.Repeat(" ", len(lastProgress)) + "\r" + progress)
		lastProgress = progress
	})
	fmt.Print("\r" + strings.Repeat(" ", len(lastProgress)) + "\r")
	if err != nil {
		return fmt.Errorf("indexing failed: %w", err)
	}

	fmt.Printf("✅ Indexed in %v\n", time.Since(start))
	fmt.Printf("🔍 Search with: qwelli search \"your query\" --index %s\n", absPath)
	return nil
}
