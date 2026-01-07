package fileprocessor

import (
	"testing"
)

func TestSupportedTextExtensions(t *testing.T) {
	// Test some key text extensions
	tests := []struct {
		ext  string
		want bool
	}{
		{"txt", true},
		{"md", true},
		{"go", true},
		{"py", true},
		{"js", true},
		{"pdf", false}, // PDF is not in text extensions
		{"exe", false},
		{"", false},
	}

	for _, tt := range tests {
		t.Run(tt.ext, func(t *testing.T) {
			got := SupportedTextExtensions[tt.ext]
			if got != tt.want {
				t.Errorf("SupportedTextExtensions[%s] = %v, want %v", tt.ext, got, tt.want)
			}
		})
	}
}

func TestSupportedPDFExtensions(t *testing.T) {
	if !SupportedPDFExtensions["pdf"] {
		t.Error("PDF should be in SupportedPDFExtensions")
	}

	if SupportedPDFExtensions["txt"] {
		t.Error("txt should not be in SupportedPDFExtensions")
	}
}

func TestIsSupported(t *testing.T) {
	tests := []struct {
		name     string
		fileType string
		want     bool
	}{
		{"text file", "txt", true},
		{"go file", "go", true},
		{"pdf file", "pdf", true},
		{"markdown", "md", true},
		{"python", "py", true},
		{"executable", "exe", false},
		{"binary", "bin", false},
		{"unknown", "xyz", false},
		{"empty", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := IsSupported(tt.fileType)
			if got != tt.want {
				t.Errorf("IsSupported(%s) = %v, want %v", tt.fileType, got, tt.want)
			}
		})
	}
}

func TestIsTextFile(t *testing.T) {
	tests := []struct {
		name     string
		fileType string
		want     bool
	}{
		{"text", "txt", true},
		{"markdown", "md", true},
		{"go", "go", true},
		{"pdf", "pdf", false},
		{"unknown", "xyz", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := IsTextFile(tt.fileType)
			if got != tt.want {
				t.Errorf("IsTextFile(%s) = %v, want %v", tt.fileType, got, tt.want)
			}
		})
	}
}

func TestIsPDFFile(t *testing.T) {
	tests := []struct {
		name     string
		fileType string
		want     bool
	}{
		{"pdf", "pdf", true},
		{"text", "txt", false},
		{"unknown", "xyz", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := IsPDFFile(tt.fileType)
			if got != tt.want {
				t.Errorf("IsPDFFile(%s) = %v, want %v", tt.fileType, got, tt.want)
			}
		})
	}
}
