package processor

import (
	"fmt"
	"os"
	"sync"

	"github.com/joho/godotenv"
	"github.com/unidoc/unipdf/v4/common/license"
)

var (
	licenseInitOnce sync.Once
	licenseInitErr  error
)

// init initializes the UniPDF license once when the package is loaded
func init() {
	// Try to load .env file (ignore errors if not found)
	_ = godotenv.Load()
	_ = godotenv.Load("../.env")
	_ = godotenv.Load("../../.env")
	_ = godotenv.Load("../../../.env")

	licenseInitOnce.Do(func() {
		apiKey := os.Getenv("UNIDOC_LICENSE_API_KEY")
		if apiKey == "" {
			licenseInitErr = fmt.Errorf("UNIDOC_LICENSE_API_KEY environment variable not set")
			return
		}

		licenseInitErr = license.SetMeteredKey(apiKey)
		if licenseInitErr != nil {
			licenseInitErr = fmt.Errorf("failed to set UniPDF license: %w", licenseInitErr)
		}
	})
}
