package fileprocessor

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

// IsSupported returns true if the file type is supported for processing
func IsSupported(fileType string) bool {
	return SupportedTextExtensions[fileType] || SupportedPDFExtensions[fileType]
}

// IsTextFile returns true if the file type is a text file
func IsTextFile(fileType string) bool {
	return SupportedTextExtensions[fileType]
}

// IsPDFFile returns true if the file type is a PDF
func IsPDFFile(fileType string) bool {
	return SupportedPDFExtensions[fileType]
}
