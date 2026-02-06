package server

import (
	"testing"
)

func TestGetContentType(t *testing.T) {
	tests := []struct {
		name     string
		metadata map[string]interface{}
		expected string
	}{
		{
			name:     "text content type",
			metadata: map[string]interface{}{"content_type": "text"},
			expected: "text",
		},
		{
			name:     "image content type",
			metadata: map[string]interface{}{"content_type": "image"},
			expected: "image",
		},
		{
			name:     "missing content type defaults to text",
			metadata: map[string]interface{}{},
			expected: "text",
		},
		{
			name:     "nil metadata defaults to text",
			metadata: nil,
			expected: "text",
		},
		{
			name:     "non-string content type defaults to text",
			metadata: map[string]interface{}{"content_type": 123},
			expected: "text",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := getContentType(tt.metadata)
			if result != tt.expected {
				t.Errorf("getContentType(%v) = %q, want %q", tt.metadata, result, tt.expected)
			}
		})
	}
}

func TestGetPageNumbers(t *testing.T) {
	tests := []struct {
		name     string
		metadata map[string]interface{}
		expected []int
	}{
		{
			name:     "nil metadata",
			metadata: nil,
			expected: nil,
		},
		{
			name:     "no page_numbers key",
			metadata: map[string]interface{}{},
			expected: nil,
		},
		{
			name:     "direct int slice",
			metadata: map[string]interface{}{"page_numbers": []int{1, 2, 3}},
			expected: []int{1, 2, 3},
		},
		{
			name:     "interface slice with float64 (JSON unmarshaled)",
			metadata: map[string]interface{}{"page_numbers": []interface{}{float64(1), float64(2)}},
			expected: []int{1, 2},
		},
		{
			name:     "interface slice with int",
			metadata: map[string]interface{}{"page_numbers": []interface{}{1, 2}},
			expected: []int{1, 2},
		},
		{
			name:     "interface slice with int64",
			metadata: map[string]interface{}{"page_numbers": []interface{}{int64(5)}},
			expected: []int{5},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := getPageNumbers(tt.metadata)
			if tt.expected == nil {
				if result != nil {
					t.Errorf("getPageNumbers(%v) = %v, want nil", tt.metadata, result)
				}
				return
			}
			if len(result) != len(tt.expected) {
				t.Errorf("getPageNumbers(%v) returned %d items, want %d", tt.metadata, len(result), len(tt.expected))
				return
			}
			for i, v := range result {
				if v != tt.expected[i] {
					t.Errorf("getPageNumbers(%v)[%d] = %d, want %d", tt.metadata, i, v, tt.expected[i])
				}
			}
		})
	}
}

func TestGetChunkIndex(t *testing.T) {
	tests := []struct {
		name     string
		metadata map[string]interface{}
		expected int
	}{
		{
			name:     "float64 value (JSON unmarshaled)",
			metadata: map[string]interface{}{"chunk_index": float64(3)},
			expected: 3,
		},
		{
			name:     "int value",
			metadata: map[string]interface{}{"chunk_index": 5},
			expected: 5,
		},
		{
			name:     "missing key returns 0",
			metadata: map[string]interface{}{},
			expected: 0,
		},
		{
			name:     "nil metadata returns 0",
			metadata: nil,
			expected: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := getChunkIndex(tt.metadata)
			if result != tt.expected {
				t.Errorf("getChunkIndex(%v) = %d, want %d", tt.metadata, result, tt.expected)
			}
		})
	}
}

func TestGetTotalChunks(t *testing.T) {
	tests := []struct {
		name     string
		metadata map[string]interface{}
		expected int
	}{
		{
			name:     "float64 value",
			metadata: map[string]interface{}{"total_chunks": float64(10)},
			expected: 10,
		},
		{
			name:     "int value",
			metadata: map[string]interface{}{"total_chunks": 7},
			expected: 7,
		},
		{
			name:     "missing key returns 0",
			metadata: map[string]interface{}{},
			expected: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := getTotalChunks(tt.metadata)
			if result != tt.expected {
				t.Errorf("getTotalChunks(%v) = %d, want %d", tt.metadata, result, tt.expected)
			}
		})
	}
}

func TestGenerateDBName(t *testing.T) {
	tests := []struct {
		name     string
		path     string
		expected string
	}{
		{
			name:     "simple folder name",
			path:     "/home/user/documents",
			expected: "documents.db",
		},
		{
			name:     "folder with spaces becomes underscores",
			path:     "/home/user/my documents",
			expected: "my_documents.db",
		},
		{
			name:     "folder with special characters",
			path:     "/home/user/my-project_v2",
			expected: "my-project_v2.db",
		},
		{
			name:     "only base name is used",
			path:     "/a/b/c/target",
			expected: "target.db",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := generateDBName(tt.path)
			if result != tt.expected {
				t.Errorf("generateDBName(%q) = %q, want %q", tt.path, result, tt.expected)
			}
		})
	}
}
