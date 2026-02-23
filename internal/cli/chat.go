package cli

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"strings"

	"github.com/karim-daw/qwelli/internal/agent"
	"github.com/karim-daw/qwelli/internal/config"
	"github.com/karim-daw/qwelli/internal/service"
	"github.com/spf13/cobra"
)

func NewChatCmd() *cobra.Command {
	var indexPath, model string

	cmd := &cobra.Command{
		Use:   "chat",
		Short: "Interactive AI chat about your indexed documents",
		Long:  "Start an interactive chat session powered by Claude. Ask natural-language questions about your indexed document collection and get cited, synthesized answers.",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runChat(indexPath, model)
		},
	}
	cmd.Flags().StringVarP(&indexPath, "index", "i", "", "Path to indexed folder (required)")
	cmd.Flags().StringVar(&model, "model", "", "Override the Foundry model name")
	cmd.MarkFlagRequired("index")
	return cmd
}

func runChat(indexPath, modelOverride string) error {
	absPath, err := filepath.Abs(indexPath)
	if err != nil {
		return fmt.Errorf("invalid path: %w", err)
	}

	cfg, err := config.Load()
	if err != nil {
		return err
	}

	if cfg.FoundryEndpoint == "" {
		return fmt.Errorf("FOUNDRY_ENDPOINT not set — configure via env var or config.yaml")
	}
	if cfg.FoundryAPIKey == "" {
		return fmt.Errorf("FOUNDRY_API_KEY not set — configure via env var or config.yaml")
	}

	model := cfg.FoundryModel
	if modelOverride != "" {
		model = modelOverride
	}

	svc, err := service.Load()
	if err != nil {
		return err
	}

	ag, err := agent.New(cfg.FoundryEndpoint, cfg.FoundryAPIKey, model, svc, absPath)
	if err != nil {
		return err
	}

	fmt.Printf("Qwelli Chat — connected to %s\n", model)
	fmt.Printf("Index: %s\n", absPath)
	fmt.Println("Type 'exit' to quit, 'clear' to reset conversation.")
	fmt.Println()

	scanner := bufio.NewScanner(os.Stdin)

	// Handle Ctrl+C: cancel current turn, not the session
	sessionCtx, sessionCancel := context.WithCancel(context.Background())
	defer sessionCancel()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt)

	for {
		fmt.Print("> ")
		if !scanner.Scan() {
			break // EOF
		}

		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}

		switch strings.ToLower(line) {
		case "exit", "quit":
			fmt.Println("Goodbye.")
			return nil
		case "clear":
			ag.ClearHistory()
			fmt.Println("Conversation cleared.")
			fmt.Println()
			continue
		}

		// Create a cancellable context for this turn
		turnCtx, turnCancel := context.WithCancel(sessionCtx)

		// Listen for Ctrl+C during this turn
		done := make(chan struct{})
		go func() {
			select {
			case <-sigCh:
				fmt.Fprintln(os.Stderr, "\n  [cancelled]")
				turnCancel()
			case <-done:
			}
		}()

		err := ag.Chat(turnCtx, line)
		close(done)
		turnCancel()

		if err != nil {
			if turnCtx.Err() != nil {
				// Cancelled by Ctrl+C — continue the session
				fmt.Println()
				continue
			}
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		}

		fmt.Println()
	}

	return nil
}
