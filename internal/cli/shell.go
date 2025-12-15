package cli

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/karim-daw/qwelli/internal/config"
	"github.com/spf13/cobra"
)

var currentIndex string

func NewShellCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "shell",
		Short: "Start interactive shell",
		RunE:  runShell,
	}
}

func runShell(cmd *cobra.Command, args []string) error {
	reader := bufio.NewReader(os.Stdin)

	cfg, err := config.Load()
	if err != nil {
		fmt.Printf("⚠️  Warning: %v\n", err)
		fmt.Println("Run 'init' to set up configuration first\n")
	} else {
		fmt.Printf("📊 Current model: %s\n\n", cfg.Model)
	}

	fmt.Println("🔍 Qwelli Interactive Shell")
	fmt.Println("Type 'help' for commands, 'exit' to quit\n")

	for {
		prompt := "qwelli"
		if currentIndex != "" {
			prompt = fmt.Sprintf("qwelli [%s]", currentIndex)
		}
		fmt.Printf("%s> ", prompt)

		input, err := reader.ReadString('\n')
		if err != nil {
			return err
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
				fmt.Println("❌ Usage: index <folder>")
				continue
			}
			if err := runIndex(cmd, cmdArgs); err != nil {
				fmt.Printf("❌ Error: %v\n", err)
			} else {
				currentIndex = cmdArgs[0]
			}

		case "search":
			if len(cmdArgs) == 0 {
				fmt.Println("❌ Usage: search <query>")
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

			query := strings.Join(cmdArgs, " ")
			if err := runSearch(query, indexPath, 5); err != nil {
				fmt.Printf("❌ Error: %v\n", err)
			}

		case "use":
			if len(cmdArgs) == 0 {
				fmt.Println("❌ Usage: use <folder>")
				continue
			}
			currentIndex = cmdArgs[0]
			fmt.Printf("✅ Using index: %s\n", currentIndex)

		case "list":
			if err := runList(cmd, cmdArgs); err != nil {
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

		default:
			fmt.Printf("❌ Unknown command: %s\nType 'help' for available commands\n", command)
		}
		fmt.Println()
	}
}

func printShellHelp() {
	fmt.Println(`Available commands:

  init              - Initialize configuration
  index <folder>    - Index a folder (sets as current)
  use <folder>      - Set current index folder
  search <query>    - Search current index
  list              - List all indexed folders
  status [folder]   - Show index status
  model [name]      - Show or change embedding model

  clear             - Clear screen
  help              - Show this help
  exit              - Exit shell`)
}
