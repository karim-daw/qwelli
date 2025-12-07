package cli

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/karim-daw/qwelli/internal/config"
	"github.com/spf13/cobra"
)

func NewInitCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "init",
		Short: "Initialize qwelli configuration",
		Long:  "Set up your qwelli configuration including API keys and preferences",
		RunE:  runInit,
	}
}

func runInit(cmd *cobra.Command, args []string) error {
	reader := bufio.NewReader(os.Stdin)

	fmt.Println("🚀 Welcome to Qwelli!")
	fmt.Println("Let's set up your configuration.\n")

	// Start with defaults
	cfg := config.DefaultConfig()

	// Ask for API key
	fmt.Print("Enter your OpenAI API key: ")
	apiKey, _ := reader.ReadString('\n')
	apiKey = strings.TrimSpace(apiKey)
	if apiKey == "" {
		return fmt.Errorf("API key is required")
	}
	cfg.APIKey = apiKey

	// Ask for model (with default)
	fmt.Printf("Enter embedding model [%s]: ", cfg.Model)
	model, _ := reader.ReadString('\n')
	model = strings.TrimSpace(model)
	if model != "" {
		cfg.Model = model
	}

	// Ask for endpoint (with default)
	fmt.Printf("Enter API endpoint [%s]: ", cfg.Endpoint)
	endpoint, _ := reader.ReadString('\n')
	endpoint = strings.TrimSpace(endpoint)
	if endpoint != "" {
		cfg.Endpoint = endpoint
	}

	// Save configuration
	if err := cfg.Save(); err != nil {
		return fmt.Errorf("failed to save config: %w", err)
	}

	// Create index directory
	if err := cfg.EnsureIndexDir(); err != nil {
		return fmt.Errorf("failed to create index directory: %w", err)
	}

	fmt.Println("\n✅ Configuration saved!")
	fmt.Printf("📁 Config file: %s\n", config.ConfigPath())
	fmt.Printf("📁 Index directory: %s\n", cfg.IndexDir)
	fmt.Println("\nYou can now run 'qwelli index <folder>' to start indexing!")

	return nil
}
