package extraction

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"image"
	"strings"
)

// ImageData represents validated image data ready for embedding
type ImageData struct {
	Base64 string
	Format string
	Pixels int
}

// ValidationStats tracks statistics about image validation
type ValidationStats struct {
	EmptyBase64      int
	InvalidBase64    int
	EmptyDecoded     int
	TooSmallBytes    int
	TooLargeBytes    int
	InvalidFormat    int
	DecodeFailed     int
	TooSmallDims     int
	TooLargeDims     int
	BadAspectRatio   int
	ExceedsVoyageMax int
	ReencodeFailed   int
	Accepted         int
}

// Config holds validation configuration
type Config struct {
	MinImageSizeBytes int
	MaxImageSizeBytes int
	MinWidth          int
	MinHeight         int
	MinPixels         int
	MaxWidth          int
	MaxHeight         int
	MaxPixels         int
	MinAspectRatio    float64
	MaxAspectRatio    float64
	VoyageMaxPixels   int
}

// DefaultConfig returns sensible defaults for image validation
func DefaultConfig() Config {
	return Config{
		MinImageSizeBytes: 100,
		MaxImageSizeBytes: 10 * 1024 * 1024, // 10MB
		MinWidth:          500,
		MinHeight:         350,
		MinPixels:         175_000,
		MaxWidth:          4000,
		MaxHeight:         3000,
		MaxPixels:         12_000_000,
		MinAspectRatio:    0.1,
		MaxAspectRatio:    10.0,
		VoyageMaxPixels:   16_000_000,
	}
}

// ImageValidator validates and prepares images for embedding
type ImageValidator struct {
	config Config
	stats  ValidationStats
}

// NewImageValidator creates a new image validator with default config
func NewImageValidator() *ImageValidator {
	return &ImageValidator{
		config: DefaultConfig(),
	}
}

// NewImageValidatorWithConfig creates a new image validator with custom config
func NewImageValidatorWithConfig(config Config) *ImageValidator {
	return &ImageValidator{
		config: config,
	}
}

// ValidateAndPrepare validates an image chunk and prepares it for embedding
// Returns the prepared ImageData and true if valid, or nil and false if invalid
func (v *ImageValidator) ValidateAndPrepare(imageBase64 string, chunkIndex int) (*ImageData, bool) {
	// Validate base64 - skip if empty or invalid
	if imageBase64 == "" || len(imageBase64) < 10 {
		v.stats.EmptyBase64++
		return nil, false
	}

	// Remove any whitespace/newlines that might have been introduced
	imageBase64 = strings.TrimSpace(imageBase64)
	imageBase64 = strings.ReplaceAll(imageBase64, "\n", "")
	imageBase64 = strings.ReplaceAll(imageBase64, "\r", "")

	// Remove data URI prefix if present (e.g., "data:image/png;base64,")
	if strings.HasPrefix(imageBase64, "data:") {
		commaIdx := strings.Index(imageBase64, ",")
		if commaIdx > 0 && commaIdx < len(imageBase64)-1 {
			imageBase64 = imageBase64[commaIdx+1:]
		}
	}

	// Validate base64 by attempting to decode it
	decoded, err := base64.StdEncoding.DecodeString(imageBase64)
	if err != nil {
		v.stats.InvalidBase64++
		return nil, false
	}

	// Additional validation: decoded data should be reasonable size
	if len(decoded) == 0 {
		v.stats.EmptyDecoded++
		return nil, false
	}

	// Skip images that are too small
	if len(decoded) < v.config.MinImageSizeBytes {
		v.stats.TooSmallBytes++
		return nil, false
	}

	// Check if image is too large
	if len(decoded) > v.config.MaxImageSizeBytes {
		v.stats.TooLargeBytes++
		return nil, false
	}

	// Check if it's a valid image format (basic check - should start with image magic bytes)
	if len(decoded) < 4 {
		v.stats.TooSmallBytes++
		return nil, false
	}

	isValidImage := (decoded[0] == 0xFF && decoded[1] == 0xD8 && decoded[2] == 0xFF) || // JPEG
		(decoded[0] == 0x89 && decoded[1] == 0x50 && decoded[2] == 0x4E && decoded[3] == 0x47) || // PNG
		(decoded[0] == 0x47 && decoded[1] == 0x49 && decoded[2] == 0x46 && decoded[3] == 0x38) // GIF

	if !isValidImage {
		v.stats.InvalidFormat++
		return nil, false
	}

	// Additional check: try to decode and verify image can be parsed
	img, format, err := image.Decode(bytes.NewReader(decoded))
	if err != nil {
		v.stats.DecodeFailed++
		return nil, false
	}

	// Get image dimensions
	bounds := img.Bounds()
	width := bounds.Dx()
	height := bounds.Dy()
	pixelCount := width * height

	// Skip images that are too small
	if width < v.config.MinWidth || height < v.config.MinHeight || pixelCount < v.config.MinPixels {
		v.stats.TooSmallDims++
		return nil, false
	}

	// Skip images that are too large
	if width > v.config.MaxWidth || height > v.config.MaxHeight || pixelCount > v.config.MaxPixels {
		v.stats.TooLargeDims++
		return nil, false
	}

	// Check aspect ratio
	aspectRatio := float64(width) / float64(height)
	if aspectRatio < v.config.MinAspectRatio || aspectRatio > v.config.MaxAspectRatio {
		v.stats.BadAspectRatio++
		return nil, false
	}

	// Voyage API compliance check
	if pixelCount > v.config.VoyageMaxPixels {
		v.stats.ExceedsVoyageMax++
		return nil, false
	}

	// Check file size (Voyage limit is 20MB)
	maxImageSizeBytes := 20 * 1024 * 1024
	if len(decoded) > maxImageSizeBytes {
		v.stats.TooLargeBytes++
		return nil, false
	}

	// Re-encode base64 to ensure it's clean and properly formatted
	cleanBase64 := base64.StdEncoding.EncodeToString(decoded)

	// Verify the re-encoded base64 can be decoded back
	testDecoded, err := base64.StdEncoding.DecodeString(cleanBase64)
	if err != nil || len(testDecoded) != len(decoded) {
		v.stats.ReencodeFailed++
		return nil, false
	}

	// Image passed all validation
	v.stats.Accepted++

	// Normalize format name
	formatLower := strings.ToLower(format)

	return &ImageData{
		Base64: cleanBase64,
		Format: formatLower,
		Pixels: pixelCount,
	}, true
}

// GetStats returns the current validation statistics
func (v *ImageValidator) GetStats() ValidationStats {
	return v.stats
}

// ResetStats resets the validation statistics
func (v *ImageValidator) ResetStats() {
	v.stats = ValidationStats{}
}

// FormatStats returns a formatted string of validation statistics
func (v *ImageValidator) FormatStats() string {
	stats := v.stats
	totalSkipped := stats.EmptyBase64 + stats.InvalidBase64 + stats.EmptyDecoded +
		stats.TooSmallBytes + stats.TooLargeBytes + stats.InvalidFormat +
		stats.DecodeFailed + stats.TooSmallDims + stats.TooLargeDims +
		stats.BadAspectRatio + stats.ExceedsVoyageMax + stats.ReencodeFailed

	if totalSkipped == 0 && stats.Accepted == 0 {
		return ""
	}

	var skipReasons []string
	if stats.EmptyBase64 > 0 {
		skipReasons = append(skipReasons, fmt.Sprintf("%d empty", stats.EmptyBase64))
	}
	if stats.InvalidBase64 > 0 {
		skipReasons = append(skipReasons, fmt.Sprintf("%d invalid base64", stats.InvalidBase64))
	}
	if stats.TooSmallDims > 0 {
		skipReasons = append(skipReasons, fmt.Sprintf("%d too small", stats.TooSmallDims))
	}
	if stats.TooLargeDims > 0 {
		skipReasons = append(skipReasons, fmt.Sprintf("%d too large", stats.TooLargeDims))
	}
	if stats.BadAspectRatio > 0 {
		skipReasons = append(skipReasons, fmt.Sprintf("%d bad aspect ratio", stats.BadAspectRatio))
	}
	if stats.DecodeFailed > 0 {
		skipReasons = append(skipReasons, fmt.Sprintf("%d decode failed", stats.DecodeFailed))
	}
	if stats.InvalidFormat > 0 {
		skipReasons = append(skipReasons, fmt.Sprintf("%d invalid format", stats.InvalidFormat))
	}
	if stats.TooLargeBytes > 0 {
		skipReasons = append(skipReasons, fmt.Sprintf("%d too large (bytes)", stats.TooLargeBytes))
	}

	if len(skipReasons) > 0 {
		return fmt.Sprintf("📊 Image validation: %d accepted, %d skipped (%s)", stats.Accepted, totalSkipped, strings.Join(skipReasons, ", "))
	}
	return fmt.Sprintf("📊 Image validation: %d accepted", stats.Accepted)
}
