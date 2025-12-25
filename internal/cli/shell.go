package cli

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/karim-daw/qwelli/internal/config"
	"github.com/karim-daw/qwelli/internal/engine"
	"github.com/spf13/cobra"
)

var currentIndex string
var cachedIndexList []string

func NewShellCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "shell",
		Short: "Start interactive shell",
		RunE:  runShell,
	}
}

func runShell(cmd *cobra.Command, args []string) error {
	// Check if stdin is available
	stat, err := os.Stdin.Stat()
	if err != nil {
		return fmt.Errorf("failed to check stdin: %w", err)
	}

	// Ensure we're running in an interactive terminal
	if (stat.Mode() & os.ModeCharDevice) == 0 {
		return fmt.Errorf("shell command requires an interactive terminal\nPlease run this command directly in your terminal, not through a pipe or redirect")
	}

	reader := bufio.NewReader(os.Stdin)

	cfg, err := config.Load()
	if err != nil {
		fmt.Printf("⚠️  Warning: %v\n", err)
		fmt.Println("Run 'init' to set up configuration first")
	} else {
		fmt.Printf("📊 Current model: %s\n\n", cfg.Model)
	}

	fmt.Println("🔍 Qwelli Interactive Shell")
	fmt.Println("Type 'help' for commands, 'exit' to quit")

	for {
		prompt := "qwelli"
		if currentIndex != "" {
			prompt = fmt.Sprintf("qwelli [%s]", currentIndex)
		}
		fmt.Printf("%s> ", prompt)

		input, err := reader.ReadString('\n')
		if err != nil {
			if err.Error() == "EOF" {
				fmt.Println("\n👋 Goodbye!")
				return nil
			}
			return fmt.Errorf("error reading input: %w", err)
		}

		input = strings.TrimSpace(input)
		if input == "" {
			continue
		}

		parts := strings.Fields(input)
		command := parts[0]
		cmdArgs := parts[1:]

		switch command {
		case "exit", "quit", "q":
			fmt.Println("👋 Goodbye!")
			return nil

		case "help", "?":
			printShellHelp()

		case "clear", "cls":
			fmt.Print("\033[H\033[2J")

		case "init":
			if err := runInit(cmd, cmdArgs); err != nil {
				fmt.Printf("❌ Error: %v\n", err)
			}

		case "index":
			if len(cmdArgs) == 0 {
				fmt.Println("❌ Usage: index <folder> [--incremental]")
				continue
			}
			// Check for incremental flag
			incremental := false
			for i, arg := range cmdArgs {
				if arg == "--incremental" || arg == "-i" {
					incremental = true
					// Remove flag from args
					cmdArgs = append(cmdArgs[:i], cmdArgs[i+1:]...)
					break
				}
			}
			if len(cmdArgs) == 0 {
				fmt.Println("❌ Usage: index <folder> [--incremental]")
				continue
			}
			if err := runIndex(cmdArgs[0], incremental, false); err != nil {
				fmt.Printf("❌ Error: %v\n", err)
			} else {
				currentIndex = cmdArgs[0]
			}

		case "search":
			if len(cmdArgs) == 0 {
				fmt.Println("❌ Usage: search <query> [--top N] [--images-only] [--text-only] [--strategy semantic|keyword|hybrid]")
				continue
			}

			indexPath := currentIndex
			if indexPath == "" {
				fmt.Print("Index path: ")
				indexInput, _ := reader.ReadString('\n')
				indexPath = strings.TrimSpace(indexInput)
			}

			if indexPath == "" {
				fmt.Println("❌ No index specified. Use 'index <folder>' first or 'use <folder>'")
				continue
			}

			// Parse query and flags
			query, topN, textOnly, imagesOnly, strategy := parseSearchArgs(cmdArgs)
			if err := runSearchShell(query, indexPath, topN, textOnly, imagesOnly, strategy); err != nil {
				fmt.Printf("❌ Error: %v\n", err)
			}

		case "use":
			if len(cmdArgs) == 0 {
				fmt.Println("❌ Usage: use <folder|number>")
				fmt.Println("   Use 'list' to see numbered options")
				continue
			}

			// Check if argument is a number
			if num, err := strconv.Atoi(cmdArgs[0]); err == nil {
				// Numeric selection - get from cached list
				if err := useIndexByNumber(num); err != nil {
					fmt.Printf("❌ Error: %v\n", err)
					fmt.Println("   Run 'list' to see available indexes")
				} else {
					fmt.Printf("✅ Using index: %s\n", currentIndex)
				}
			} else {
				// Path-based selection
				currentIndex = cmdArgs[0]
				fmt.Printf("✅ Using index: %s\n", currentIndex)
			}

		case "list":
			if err := runListShell(cmd, cmdArgs); err != nil {
				fmt.Printf("❌ Error: %v\n", err)
			}

		case "status":
			indexPath := currentIndex
			if len(cmdArgs) > 0 {
				indexPath = cmdArgs[0]
			}
			if indexPath == "" {
				fmt.Println("❌ No index specified. Use 'use <folder>' first")
				continue
			}
			if err := runStatus(indexPath); err != nil {
				fmt.Printf("❌ Error: %v\n", err)
			}

		case "model":
			cfg, err := config.Load()
			if err != nil {
				fmt.Printf("❌ Error: %v\n", err)
				continue
			}
			if len(cmdArgs) == 0 {
				fmt.Printf("📊 Current model: %s\n", cfg.Model)
			} else {
				cfg.Model = cmdArgs[0]
				if err := cfg.Save(); err != nil {
					fmt.Printf("❌ Error: %v\n", err)
				} else {
					fmt.Printf("✅ Model changed to: %s\n", cfg.Model)
					fmt.Println("⚠️  Re-index to use the new model")
				}
			}

		case "delete", "remove", "rm":
			indexPath := currentIndex
			if len(cmdArgs) > 0 {
				indexPath = cmdArgs[0]
			}
			if indexPath == "" {
				fmt.Println("❌ No index specified. Usage: delete <folder|#>")
				fmt.Println("   Use 'list' to see available indexes")
				continue
			}

			// Check if argument is a number
			if len(cmdArgs) > 0 {
				if num, err := strconv.Atoi(cmdArgs[0]); err == nil {
					// Numeric selection - get from cached list
					if len(cachedIndexList) == 0 {
						fmt.Println("❌ No cached index list. Run 'list' first")
						continue
					}
					if num < 1 || num > len(cachedIndexList) {
						fmt.Printf("❌ Invalid number %d. Must be between 1 and %d\n", num, len(cachedIndexList))
						continue
					}
					indexPath = cachedIndexList[num-1]
				}
			}

			// Normalize paths for comparison
			deletedPathAbs, err := filepath.Abs(indexPath)
			if err != nil {
				deletedPathAbs = indexPath // Fallback to original if abs fails
			}
			currentPathAbs := ""
			if currentIndex != "" {
				if abs, err := filepath.Abs(currentIndex); err == nil {
					currentPathAbs = abs
				} else {
					currentPathAbs = currentIndex // Fallback
				}
			}

			if err := deleteIndexInteractive(indexPath, reader); err != nil {
				fmt.Printf("❌ Error: %v\n", err)
			} else {
				// Clear current index if it was deleted (compare normalized paths)
				if currentIndex != "" && deletedPathAbs == currentPathAbs {
					currentIndex = ""
				}
				// Clear cached list to force refresh
				cachedIndexList = nil
			}

		default:
			fmt.Printf("❌ Unknown command: %s\nType 'help' for available commands\n", command)
		}
		fmt.Println()
	}
}

func printShellHelp() {
	fmt.Println(`Available commands:

  init                  - Initialize configuration
  index <folder>        - Index a folder (sets as current)
  use <folder|#>        - Set current index folder (use number from list)
  search <query> [opts] - Search current index
    --top N, -t N       - Number of results (default: 5)
    --images-only       - Search only image chunks
    --text-only         - Search only text chunks
  list                  - List all indexed folders (with numbers)
  status [folder]       - Show index status
  delete <folder|#>     - Delete an index (asks for confirmation)
  model [name]          - Show or change embedding model

  clear                 - Clear screen
  help                  - Show this help
  exit                  - Exit shell

Examples:
  list                  - Show all indexes with numbers
  use 1                 - Use the first index from the list
  use ./my-folder       - Use a specific folder path
  delete 1              - Delete the first index from the list
  delete ./my-folder    - Delete index for specific folder
  search hello          - Search for "hello" (top 5 results)
  search hello --top 3  - Search for "hello" (top 3 results)
  search hello --images-only - Search only images`)
}

// runListShell lists all indexed folders and caches them for numeric selection
func runListShell(_ *cobra.Command, _ []string) error {
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
		fmt.Println("Run 'index <folder>' to create your first index.")
		cachedIndexList = nil
		return nil
	}

	fmt.Printf("📚 Indexed folders (%d):\n\n", len(files))

	eng := engine.NewEngine(cfg.APIKey, cfg.Model, cfg.Endpoint, cfg.EnableMultimodal)
	cachedIndexList = make([]string, len(files))

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

		// Cache the folder path
		cachedIndexList[i] = folderPath

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

	fmt.Println("💡 Tip: Use 'use <number>' to quickly select an index")

	return nil
}

// useIndexByNumber sets the current index based on the number from the list
func useIndexByNumber(num int) error {
	if len(cachedIndexList) == 0 {
		return fmt.Errorf("no cached index list. Run 'list' first")
	}

	if num < 1 || num > len(cachedIndexList) {
		return fmt.Errorf("invalid number %d. Must be between 1 and %d", num, len(cachedIndexList))
	}

	currentIndex = cachedIndexList[num-1]
	return nil
}

// parseSearchArgs parses search command arguments to extract query, --top flag, content type filters, and strategy
func parseSearchArgs(args []string) (query string, topN int, textOnly bool, imagesOnly bool, strategy string) {
	topN = 5              // default
	strategy = "semantic" // default
	queryParts := []string{}

	for i := 0; i < len(args); i++ {
		if args[i] == "--top" || args[i] == "-t" {
			// Next arg should be the number
			if i+1 < len(args) {
				if n, err := strconv.Atoi(args[i+1]); err == nil {
					topN = n
					i++ // skip the number
					continue
				}
			}
		} else if args[i] == "--images-only" || args[i] == "--image-only" {
			imagesOnly = true
		} else if args[i] == "--text-only" {
			textOnly = true
		} else if args[i] == "--strategy" {
			// Next arg should be the strategy name
			if i+1 < len(args) {
				strategy = args[i+1]
				i++ // skip the strategy name
				continue
			}
		} else {
			queryParts = append(queryParts, args[i])
		}
	}

	query = strings.Join(queryParts, " ")
	return query, topN, textOnly, imagesOnly, strategy
}
