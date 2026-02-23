package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/anthropics/anthropic-sdk-go"
	"github.com/karim-daw/qwelli/internal/engine"
	"github.com/karim-daw/qwelli/internal/service"
)

const maxReadFileSize = 50 * 1024 // 50KB

// toolDefs returns the Anthropic tool definitions for the agent.
func toolDefs() []anthropic.ToolUnionParam {
	return []anthropic.ToolUnionParam{
		{OfTool: &anthropic.ToolParam{
			Name:        "search",
			Description: anthropic.String("Search the indexed document collection using semantic, keyword, or hybrid strategies. Returns matching chunks with file paths, page numbers, and text previews."),
			InputSchema: anthropic.ToolInputSchemaParam{
				Properties: map[string]any{
					"query": map[string]any{
						"type":        "string",
						"description": "The search query — a natural language question or keywords.",
					},
					"strategy": map[string]any{
						"type":        "string",
						"enum":        []string{"semantic", "keyword", "hybrid"},
						"description": "Search strategy. Semantic finds conceptually similar content, keyword does exact matching, hybrid combines both. Default: semantic.",
					},
					"top": map[string]any{
						"type":        "integer",
						"description": "Number of results to return (1-20). Default: 8.",
					},
				},
				Required: []string{"query"},
			},
		}},
		{OfTool: &anthropic.ToolParam{
			Name:        "status",
			Description: anthropic.String("Check the index status — shows total files, files up-to-date, and pending changes (new, modified, deleted files)."),
			InputSchema: anthropic.ToolInputSchemaParam{
				Properties: map[string]any{},
			},
		}},
		{OfTool: &anthropic.ToolParam{
			Name:        "read_file",
			Description: anthropic.String("Read the full text contents of a file within the indexed folder. Only works for text-based files (not PDFs or images). Use search to find content in PDFs."),
			InputSchema: anthropic.ToolInputSchemaParam{
				Properties: map[string]any{
					"path": map[string]any{
						"type":        "string",
						"description": "Absolute path to the file to read. Must be within the indexed folder.",
					},
				},
				Required: []string{"path"},
			},
		}},
		{OfTool: &anthropic.ToolParam{
			Name:        "list_dir",
			Description: anthropic.String("List the contents of a directory within the indexed folder. Returns file names, sizes, and whether each entry is a directory."),
			InputSchema: anthropic.ToolInputSchemaParam{
				Properties: map[string]any{
					"path": map[string]any{
						"type":        "string",
						"description": "Absolute path to the directory to list. Must be within the indexed folder.",
					},
				},
				Required: []string{"path"},
			},
		}},
		{OfTool: &anthropic.ToolParam{
			Name:        "index_update",
			Description: anthropic.String("Perform an incremental update of the index — scans for new, modified, and deleted files, then re-indexes as needed. Use after confirming changes via the status tool."),
			InputSchema: anthropic.ToolInputSchemaParam{
				Properties: map[string]any{},
			},
		}},
	}
}

// executeTool dispatches a tool call to the appropriate handler.
// Returns the tool result string and whether it's an error.
func executeTool(ctx context.Context, svc *service.Service, indexPath, dbPath, name string, rawInput json.RawMessage) (string, bool) {
	switch name {
	case "search":
		return execSearch(svc, dbPath, rawInput)
	case "status":
		return execStatus(svc, indexPath)
	case "read_file":
		return execReadFile(indexPath, rawInput)
	case "list_dir":
		return execListDir(indexPath, rawInput)
	case "index_update":
		return execIndexUpdate(ctx, svc, indexPath)
	default:
		return fmt.Sprintf("unknown tool: %s", name), true
	}
}

// --- Tool implementations ---

type searchInput struct {
	Query    string `json:"query"`
	Strategy string `json:"strategy"`
	Top      int    `json:"top"`
}

type searchResultJSON struct {
	FilePath    string `json:"file_path"`
	FileName    string `json:"file_name"`
	Distance    float64 `json:"distance"`
	Preview     string `json:"preview"`
	PageNumbers []int  `json:"page_numbers,omitempty"`
}

func execSearch(svc *service.Service, dbPath string, rawInput json.RawMessage) (string, bool) {
	var in searchInput
	if err := json.Unmarshal(rawInput, &in); err != nil {
		return fmt.Sprintf("invalid input: %v", err), true
	}
	if in.Query == "" {
		return "query is required", true
	}
	if in.Strategy == "" {
		in.Strategy = "semantic"
	}
	if in.Top <= 0 {
		in.Top = 8
	}
	if in.Top > 20 {
		in.Top = 20
	}

	results, err := svc.SearchByDBPath(dbPath, in.Query, in.Top, "", in.Strategy)
	if err != nil {
		return fmt.Sprintf("search failed: %v", err), true
	}

	out := make([]searchResultJSON, 0, len(results))
	for _, r := range results {
		preview := r.Content
		if len(preview) > 500 {
			preview = preview[:500] + "..."
		}
		out = append(out, searchResultJSON{
			FilePath:    r.FilePath,
			FileName:    r.FileName,
			Distance:    r.Distance,
			Preview:     preview,
			PageNumbers: extractPageNumbers(r),
		})
	}

	data, _ := json.Marshal(out)
	return string(data), false
}

func extractPageNumbers(r engine.SearchResult) []int {
	if len(r.PageNumbers) > 0 {
		return r.PageNumbers
	}
	return nil
}

func execStatus(svc *service.Service, indexPath string) (string, bool) {
	status, err := svc.GetIndexStatus(indexPath)
	if err != nil {
		return fmt.Sprintf("failed to get status: %v", err), true
	}

	stats, _ := svc.GetIndexStats(indexPath)

	result := map[string]any{
		"total_files":    status.Total,
		"up_to_date":     status.UpToDate,
		"chunks":         stats,
		"pending_add":    len(status.ToAdd),
		"pending_update": len(status.ToUpdate),
		"pending_delete": len(status.ToDelete),
	}

	if len(status.ToAdd) > 0 {
		names := make([]string, 0, min(len(status.ToAdd), 10))
		for i, f := range status.ToAdd {
			if i >= 10 {
				names = append(names, fmt.Sprintf("...and %d more", len(status.ToAdd)-10))
				break
			}
			names = append(names, filepath.Base(f.Path))
		}
		result["new_files"] = names
	}
	if len(status.ToUpdate) > 0 {
		names := make([]string, 0, min(len(status.ToUpdate), 10))
		for i, f := range status.ToUpdate {
			if i >= 10 {
				names = append(names, fmt.Sprintf("...and %d more", len(status.ToUpdate)-10))
				break
			}
			names = append(names, filepath.Base(f.Path))
		}
		result["modified_files"] = names
	}
	if len(status.ToDelete) > 0 {
		names := make([]string, 0, min(len(status.ToDelete), 10))
		for i, f := range status.ToDelete {
			if i >= 10 {
				names = append(names, fmt.Sprintf("...and %d more", len(status.ToDelete)-10))
				break
			}
			names = append(names, filepath.Base(f.Path))
		}
		result["deleted_files"] = names
	}

	data, _ := json.Marshal(result)
	return string(data), false
}

type filePathInput struct {
	Path string `json:"path"`
}

func execReadFile(indexPath string, rawInput json.RawMessage) (string, bool) {
	var in filePathInput
	if err := json.Unmarshal(rawInput, &in); err != nil {
		return fmt.Sprintf("invalid input: %v", err), true
	}
	if in.Path == "" {
		return "path is required", true
	}

	// Security: ensure path is within the indexed folder
	absPath, err := filepath.Abs(in.Path)
	if err != nil {
		return fmt.Sprintf("invalid path: %v", err), true
	}
	absIndex, _ := filepath.Abs(indexPath)
	if !strings.HasPrefix(strings.ToLower(absPath), strings.ToLower(absIndex)+string(filepath.Separator)) &&
		strings.ToLower(absPath) != strings.ToLower(absIndex) {
		return fmt.Sprintf("access denied: path must be within %s", indexPath), true
	}

	// Reject binary/PDF files
	ext := strings.ToLower(filepath.Ext(absPath))
	switch ext {
	case ".pdf":
		return "cannot read PDF files directly — use the search tool to find content within PDFs", true
	case ".png", ".jpg", ".jpeg", ".gif", ".bmp", ".tiff", ".ico", ".webp":
		return "cannot read image files — use the search tool to find content about images", true
	case ".exe", ".dll", ".so", ".dylib", ".bin", ".zip", ".gz", ".tar", ".rar", ".7z":
		return "cannot read binary files", true
	}

	info, err := os.Stat(absPath)
	if err != nil {
		return fmt.Sprintf("file not found: %v", err), true
	}
	if info.IsDir() {
		return "path is a directory, use list_dir instead", true
	}

	data, err := os.ReadFile(absPath)
	if err != nil {
		return fmt.Sprintf("failed to read file: %v", err), true
	}

	content := string(data)
	if len(data) > maxReadFileSize {
		content = string(data[:maxReadFileSize]) + fmt.Sprintf("\n\n[truncated — file is %d bytes, showing first %d bytes]", len(data), maxReadFileSize)
	}

	return content, false
}

func execListDir(indexPath string, rawInput json.RawMessage) (string, bool) {
	var in filePathInput
	if err := json.Unmarshal(rawInput, &in); err != nil {
		return fmt.Sprintf("invalid input: %v", err), true
	}
	if in.Path == "" {
		in.Path = indexPath
	}

	absPath, err := filepath.Abs(in.Path)
	if err != nil {
		return fmt.Sprintf("invalid path: %v", err), true
	}
	absIndex, _ := filepath.Abs(indexPath)
	if !strings.HasPrefix(strings.ToLower(absPath), strings.ToLower(absIndex)) {
		return fmt.Sprintf("access denied: path must be within %s", indexPath), true
	}

	entries, err := os.ReadDir(absPath)
	if err != nil {
		return fmt.Sprintf("failed to read directory: %v", err), true
	}

	type dirEntry struct {
		Name  string `json:"name"`
		Size  int64  `json:"size"`
		IsDir bool   `json:"is_dir"`
	}

	out := make([]dirEntry, 0, len(entries))
	for _, e := range entries {
		info, err := e.Info()
		if err != nil {
			continue
		}
		out = append(out, dirEntry{
			Name:  e.Name(),
			Size:  info.Size(),
			IsDir: e.IsDir(),
		})
	}

	data, _ := json.Marshal(out)
	return string(data), false
}

func execIndexUpdate(ctx context.Context, svc *service.Service, indexPath string) (string, bool) {
	var processed, total int
	progressCb := func(current, count int, file string) {
		processed = current
		total = count
		fmt.Fprintf(os.Stderr, "\r  indexing: %d/%d %s", current, count, filepath.Base(file))
	}

	err := svc.UpdateIndex(ctx, indexPath, service.IndexOptions{Incremental: true}, progressCb)
	fmt.Fprintln(os.Stderr) // newline after progress
	if err != nil {
		return fmt.Sprintf("index update failed: %v", err), true
	}

	if total == 0 {
		return "index is already up to date — no changes detected", false
	}
	return fmt.Sprintf("index updated successfully — processed %d files", processed), false
}
