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
	fmt.Println("Let's set up your configuration.")

	// Start with defaults
	cfg := config.DefaultConfig()

	// Use Voyage AI as the provider
	cfg.EmbeddingProvider = "voyage"
	defaultModel := "voyage-multimodal-3"
	defaultEndpoint := "https://api.voyageai.com/v1/multimodalembeddings"
	cfg.EnableMultimodal = true

	// Ask for API key
	fmt.Print("Enter your Voyage AI API key: ")
	apiKey, _ := reader.ReadString('\n')
	apiKey = strings.TrimSpace(apiKey)
	if apiKey == "" {
		return fmt.Errorf("API key is required")
	}
	cfg.APIKey = apiKey

	// Ask for model (with default based on provider)
	if defaultModel != "" {
		cfg.Model = defaultModel
	}
	fmt.Printf("Enter embedding model [%s]: ", cfg.Model)
	model, _ := reader.ReadString('\n')
	model = strings.TrimSpace(model)
	if model != "" {
		cfg.Model = model
	}

	// Ask for endpoint (with default based on provider)
	if defaultEndpoint != "" {
		cfg.Endpoint = defaultEndpoint
	}
	fmt.Printf("Enter API endpoint [%s]: ", cfg.Endpoint)
	endpoint, _ := reader.ReadString('\n')
	endpoint = strings.TrimSpace(endpoint)
	if endpoint != "" {
		cfg.Endpoint = endpoint
	}

	// Ask about multimodal (default enabled for Voyage)
	fmt.Print("Enable multimodal indexing (extract images from PDFs)? [Y/n]: ")
	multimodalInput, _ := reader.ReadString('\n')
	multimodalInput = strings.TrimSpace(strings.ToLower(multimodalInput))
	if multimodalInput == "n" || multimodalInput == "no" {
		cfg.EnableMultimodal = false
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
