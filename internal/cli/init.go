package cli

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
)

func NewInitCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "init",
		Short: "Initialize qwelli configuration",
		Long:  "Set up your qwelli configuration using environment variables",
		RunE:  runInit,
	}
}

func runInit(cmd *cobra.Command, args []string) error {
	reader := bufio.NewReader(os.Stdin)

	fmt.Println("🚀 Welcome to Qwelli!")
	fmt.Println("Qwelli now uses environment variables for configuration.")
	fmt.Println()

	// Collect required configuration
	fmt.Print("Enter your PostgreSQL connection string\n")
	fmt.Print("(format: postgresql://user:password@host:port/database): ")
	dbURL, _ := reader.ReadString('\n')
	dbURL = strings.TrimSpace(dbURL)
	if dbURL == "" {
		return fmt.Errorf("database URL is required")
	}

	fmt.Print("\nEnter your Voyage AI API key: ")
	apiKey, _ := reader.ReadString('\n')
	apiKey = strings.TrimSpace(apiKey)
	if apiKey == "" {
		return fmt.Errorf("API key is required")
	}

	// Create .env file
	envContent := fmt.Sprintf(`# Qwelli Configuration
DATABASE_URL=%s
VOYAGE_API_KEY=%s

# Optional (with defaults)
VOYAGE_MODEL=voyage-multimodal-3
PORT=8080
ENABLE_RERANKER=true
`, dbURL, apiKey)

	envFile := ".env"
	if err := os.WriteFile(envFile, []byte(envContent), 0600); err != nil {
		return fmt.Errorf("failed to create .env file: %w", err)
	}

	fmt.Println("\n✅ Configuration saved to .env file!")
	fmt.Println("📁 File: .env")
	fmt.Println("\nYou can now run:")
	fmt.Println("  qwelli index <folder>  - Index a folder")
	fmt.Println("  qwelli serve           - Start the web server")
	fmt.Println("\nNote: Make sure your PostgreSQL database is running and accessible!")

	return nil
}
