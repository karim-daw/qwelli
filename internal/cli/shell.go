package cli

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/karim-daw/qwelli/internal/config"
	"github.com/karim-daw/qwelli/internal/service"
	"github.com/spf13/cobra"
)

var (
	currentIndex    string
	cachedIndexList []string
)

func NewShellCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "shell",
		Short: "Start interactive shell",
		RunE:  runShell,
	}
}

func runShell(cmd *cobra.Command, args []string) error {
	stat, err := os.Stdin.Stat()
	if err != nil {
		return fmt.Errorf("check stdin: %w", err)
	}
	if (stat.Mode() & os.ModeCharDevice) == 0 {
		return fmt.Errorf("shell requires an interactive terminal")
	}

	reader := bufio.NewReader(os.Stdin)

	if cfg, err := config.Load(); err != nil {
		fmt.Printf("⚠️  Warning: %v\nRun 'init' to set up configuration first\n", err)
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
			return fmt.Errorf("read input: %w", err)
		}
		input = strings.TrimSpace(input)
		if input == "" {
			continue
		}

		parts := strings.Fields(input)
		command, cmdArgs := parts[0], parts[1:]

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
			handleShellIndex(cmdArgs)
		case "search":
			handleShellSearch(cmdArgs, reader)
		case "use":
			handleShellUse(cmdArgs)
		case "list":
			if err := runListShell(); err != nil {
				fmt.Printf("❌ Error: %v\n", err)
			}
		case "status":
			p := currentIndex
			if len(cmdArgs) > 0 {
				p = cmdArgs[0]
			}
			if p == "" {
				fmt.Println("❌ No index specified. Use 'use <folder>' first")
			} else if err := runStatus(p); err != nil {
				fmt.Printf("❌ Error: %v\n", err)
			}
		case "model":
			handleShellModel(cmdArgs)
		case "delete", "remove", "rm":
			handleShellDelete(cmdArgs, reader)
		default:
			fmt.Printf("❌ Unknown command: %s\nType 'help' for commands\n", command)
		}
		fmt.Println()
	}
}

func handleShellIndex(args []string) {
	if len(args) == 0 {
		fmt.Println("❌ Usage: index <folder> [--incremental]")
		return
	}
	incremental := false
	var filtered []string
	for _, a := range args {
		if a == "--incremental" || a == "-i" {
			incremental = true
		} else {
			filtered = append(filtered, a)
		}
	}
	if len(filtered) == 0 {
		fmt.Println("❌ Usage: index <folder> [--incremental]")
		return
	}
	if err := runIndex(filtered[0], incremental, false); err != nil {
		fmt.Printf("❌ Error: %v\n", err)
	} else {
		currentIndex = filtered[0]
	}
}

func handleShellSearch(args []string, reader *bufio.Reader) {
	if len(args) == 0 {
		fmt.Println("❌ Usage: search <query> [--top N] [--strategy semantic|keyword|hybrid]")
		return
	}
	indexPath := currentIndex
	if indexPath == "" {
		fmt.Print("Index path: ")
		line, _ := reader.ReadString('\n')
		indexPath = strings.TrimSpace(line)
	}
	if indexPath == "" {
		fmt.Println("❌ No index specified. Use 'use <folder>' first")
		return
	}
	query, topN, textOnly, imagesOnly, strategy := parseSearchArgs(args)
	if err := runSearchShell(query, indexPath, topN, textOnly, imagesOnly, strategy); err != nil {
		fmt.Printf("❌ Error: %v\n", err)
	}
}

func handleShellUse(args []string) {
	if len(args) == 0 {
		fmt.Println("❌ Usage: use <folder|number>")
		return
	}
	if num, err := strconv.Atoi(args[0]); err == nil {
		if err := useIndexByNumber(num); err != nil {
			fmt.Printf("❌ Error: %v\n", err)
		} else {
			fmt.Printf("✅ Using index: %s\n", currentIndex)
		}
	} else {
		currentIndex = args[0]
		fmt.Printf("✅ Using index: %s\n", currentIndex)
	}
}

func handleShellModel(args []string) {
	cfg, err := config.Load()
	if err != nil {
		fmt.Printf("❌ Error: %v\n", err)
		return
	}
	if len(args) == 0 {
		fmt.Printf("📊 Current model: %s\n", cfg.Model)
		return
	}
	cfg.Model = args[0]
	if err := cfg.Save(); err != nil {
		fmt.Printf("❌ Error: %v\n", err)
	} else {
		fmt.Printf("✅ Model changed to: %s\n", cfg.Model)
		fmt.Println("⚠️  Re-index to use the new model")
	}
}

func handleShellDelete(args []string, reader *bufio.Reader) {
	indexPath := currentIndex
	if len(args) > 0 {
		if num, err := strconv.Atoi(args[0]); err == nil {
			if len(cachedIndexList) == 0 {
				fmt.Println("❌ No cached index list. Run 'list' first")
				return
			}
			if num < 1 || num > len(cachedIndexList) {
				fmt.Printf("❌ Invalid number %d (1-%d)\n", num, len(cachedIndexList))
				return
			}
			indexPath = cachedIndexList[num-1]
		} else {
			indexPath = args[0]
		}
	}
	if indexPath == "" {
		fmt.Println("❌ No index specified. Usage: delete <folder|#>")
		return
	}
	if err := deleteIndexInteractive(indexPath, reader); err != nil {
		fmt.Printf("❌ Error: %v\n", err)
		return
	}
	if currentIndex != "" {
		a, _ := filepath.Abs(indexPath)
		b, _ := filepath.Abs(currentIndex)
		if a != "" && a == b {
			currentIndex = ""
		}
	}
	cachedIndexList = nil
}

func runListShell() error {
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
		cachedIndexList = nil
		return nil
	}

	fmt.Printf("📚 Indexed folders (%d):\n\n", len(indexes))
	cachedIndexList = make([]string, len(indexes))
	for i, idx := range indexes {
		cachedIndexList[i] = idx.FolderPath
		fmt.Printf("%d. %s\n", i+1, idx.FolderPath)
		fmt.Printf("   📄 Documents: %d\n", idx.DocumentCount)
		fmt.Printf("   💾 Database: %s\n", idx.DBPath)
		if !idx.LastModified.IsZero() {
			fmt.Printf("   🕐 Last modified: %s\n", idx.LastModified.Format("2006-01-02 15:04:05"))
		}
		fmt.Println()
	}
	fmt.Println("💡 Tip: Use 'use <number>' to select an index")
	return nil
}

func useIndexByNumber(num int) error {
	if len(cachedIndexList) == 0 {
		return fmt.Errorf("no cached list. Run 'list' first")
	}
	if num < 1 || num > len(cachedIndexList) {
		return fmt.Errorf("invalid number %d (1-%d)", num, len(cachedIndexList))
	}
	currentIndex = cachedIndexList[num-1]
	return nil
}

func parseSearchArgs(args []string) (query string, topN int, textOnly, imagesOnly bool, strategy string) {
	topN = 5
	strategy = "semantic"
	var queryParts []string

	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--top", "-t":
			if i+1 < len(args) {
				if n, err := strconv.Atoi(args[i+1]); err == nil {
					topN = n
				}
				i++
			}
		case "--images-only", "--image-only":
			imagesOnly = true
		case "--text-only":
			textOnly = true
		case "--strategy":
			if i+1 < len(args) {
				strategy = args[i+1]
				i++
			}
		default:
			queryParts = append(queryParts, args[i])
		}
	}
	return strings.Join(queryParts, " "), topN, textOnly, imagesOnly, strategy
}

func printShellHelp() {
	fmt.Println(`Commands:
  init                  - Initialize configuration
  index <folder> [-i]   - Index a folder
  use <folder|#>        - Set current index
  search <query> [opts] - Search current index
    --top N             - Number of results (default: 5)
    --images-only       - Image chunks only
    --text-only         - Text chunks only
    --strategy <name>   - semantic, keyword, or hybrid
  list                  - List all indexes
  status [folder]       - Show index status
  delete <folder|#>     - Delete an index
  model [name]          - Show/change embedding model
  clear                 - Clear screen
  help                  - Show this help
  exit                  - Exit shell`)
}
