package fileprocessor

import (
	"path/filepath"
	"strings"
)

// GetFileTypeFromPath extracts the file type from a file path
func GetFileTypeFromPath(path string) string {
	ext := strings.ToLower(filepath.Ext(path))
	if ext == "" {
		return "unknown"
	}
	return ext[1:]
}

// SupportedTextExtensions defines all supported text file extensions
var SupportedTextExtensions = map[string]bool{
	"txt": true, "md": true, "go": true, "py": true, "js": true, "ts": true,
	"tsx": true, "jsx": true, "java": true, "c": true, "cpp": true, "h": true,
	"rs": true, "rb": true, "php": true, "cs": true, "swift": true,
	"html": true, "css": true, "scss": true, "yaml": true, "yml": true,
	"toml": true, "sh": true, "proto": true, "graphql": true,
}

// SupportedPDFExtensions defines supported PDF extensions
var SupportedPDFExtensions = map[string]bool{
	"pdf": true,
}

// SupportedImageExtensions defines supported standalone image file extensions
var SupportedImageExtensions = map[string]bool{
	"jpg":  true,
	"jpeg": true,
	"png":  true,
	"gif":  true,
}

// IsSupported returns true if the file type is supported for processing
func IsSupported(fileType string) bool {
	return SupportedTextExtensions[fileType] || SupportedPDFExtensions[fileType] || SupportedImageExtensions[fileType]
}

// IsTextFile returns true if the file type is a text file
func IsTextFile(fileType string) bool {
	return SupportedTextExtensions[fileType]
}

// IsPDFFile returns true if the file type is a PDF
func IsPDFFile(fileType string) bool {
	return SupportedPDFExtensions[fileType]
}

// IsImageFile returns true if the file type is a supported standalone image
func IsImageFile(fileType string) bool {
	return SupportedImageExtensions[fileType]
}
