package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"sync"

	"github.com/anthropics/anthropic-sdk-go"
	"github.com/anthropics/anthropic-sdk-go/option"
	"github.com/karim-daw/qwelli/internal/service"
)

// ChatEvent represents a streaming event from the agent loop.
type ChatEvent struct {
	Type       string          `json:"type"`                   // "thinking", "tool_call", "tool_result", "text_delta", "done", "error"
	Text       string          `json:"text,omitempty"`         // for text_delta, thinking, error
	ToolName   string          `json:"tool_name,omitempty"`    // for tool_call, tool_result
	ToolCallID string          `json:"tool_call_id,omitempty"` // Anthropic tool_use block ID — links tool_call to its tool_result
	ToolInput  json.RawMessage `json:"tool_input,omitempty"`   // for tool_call
	ToolResult string          `json:"tool_result,omitempty"`  // for tool_result
	IsError    bool            `json:"is_error,omitempty"`     // for tool_result
}

// Agent orchestrates a conversational tool-use loop with Claude via Azure AI Foundry.
type Agent struct {
	client   anthropic.Client
	svc      *service.Service
	model    anthropic.Model
	index    string // Absolute path to the indexed folder
	dbPath   string // Resolved DB path for this index
	messages []anthropic.MessageParam
	tools    []anthropic.ToolUnionParam
	system   []anthropic.TextBlockParam
}

// New creates an Agent connected to Azure AI Foundry.
func New(endpoint, apiKey, model string, svc *service.Service, indexPath string) (*Agent, error) {
	dbPath, err := svc.GenerateDBPath(indexPath)
	if err != nil {
		return nil, fmt.Errorf("resolve db path: %w", err)
	}
	if _, err := os.Stat(dbPath); os.IsNotExist(err) {
		return nil, fmt.Errorf("index not found for %s — run 'qwelli index' first", indexPath)
	}

	// Get stats for the system prompt
	fileCount := 0
	chunkCount := 0
	if status, err := svc.GetIndexStatus(indexPath); err == nil {
		fileCount = status.Total
	}
	if n, err := svc.GetIndexStats(indexPath); err == nil {
		chunkCount = n
	}

	client := anthropic.NewClient(
		option.WithAPIKey(apiKey),
		option.WithBaseURL(endpoint),
	)

	systemPrompt := fmt.Sprintf(`You are a research assistant with access to a document collection indexed by Qwelli.
You can search the collection, find files, read their content, check index status, and manage the index.

When answering questions:
- Run multiple searches with different queries and strategies to be thorough
- Use find_files to discover documents by type, date, or name before searching content
- Cite specific files and page numbers in your answers
- For PDFs and scanned documents, use get_file_chunks to read the full indexed text
- For text files (txt, md, csv, etc.), use read_file for the raw content
- Use get_file_info to quickly check a file's metadata and chunk count
- Ask clarifying questions when the request is ambiguous
- Synthesize across multiple sources, don't just dump search results

Available index: %s (%d files, %d chunks)`, indexPath, fileCount, chunkCount)

	return &Agent{
		client:   client,
		svc:      svc,
		model:    anthropic.Model(model),
		index:    indexPath,
		dbPath:   dbPath,
		messages: nil,
		tools:    toolDefs(),
		system:   []anthropic.TextBlockParam{{Text: systemPrompt}},
	}, nil
}

// ChatStream runs the tool-use loop, emitting structured events via the callback.
// It processes the user message and loops until the agent produces a text-only response.
func (a *Agent) ChatStream(ctx context.Context, userMessage string, emit func(ChatEvent)) error {
	a.messages = append(a.messages, anthropic.NewUserMessage(anthropic.NewTextBlock(userMessage)))

	for {
		stream := a.client.Messages.NewStreaming(ctx, anthropic.MessageNewParams{
			Model:     a.model,
			MaxTokens: 16000,
			Messages:  a.messages,
			Tools:     a.tools,
			System:    a.system,
			Thinking:  anthropic.ThinkingConfigParamOfEnabled(8000),
		})

		message := anthropic.Message{}
		for stream.Next() {
			event := stream.Current()
			if err := message.Accumulate(event); err != nil {
				emit(ChatEvent{Type: "error", Text: fmt.Sprintf("accumulate stream event: %v", err)})
				return fmt.Errorf("accumulate stream event: %w", err)
			}

			switch ev := event.AsAny().(type) {
			case anthropic.ContentBlockDeltaEvent:
				if ev.Delta.Thinking != "" {
					emit(ChatEvent{Type: "thinking", Text: ev.Delta.Thinking})
				} else if ev.Delta.Text != "" {
					emit(ChatEvent{Type: "text_delta", Text: ev.Delta.Text})
				}
			}
		}
		if err := stream.Err(); err != nil {
			emit(ChatEvent{Type: "error", Text: fmt.Sprintf("stream error: %v", err)})
			return fmt.Errorf("stream error: %w", err)
		}

		// Append the assistant's full response to history
		a.messages = append(a.messages, message.ToParam())

		// Collect tool calls and emit tool_call events upfront
		type pendingTool struct {
			variant  anthropic.ToolUseBlock
			rawInput json.RawMessage
		}
		var pending []pendingTool
		for _, block := range message.Content {
			if variant, ok := block.AsAny().(anthropic.ToolUseBlock); ok {
				rawInput := json.RawMessage(variant.JSON.Input.Raw())
				emit(ChatEvent{Type: "tool_call", ToolName: variant.Name, ToolCallID: variant.ID, ToolInput: rawInput})
				pending = append(pending, pendingTool{variant: variant, rawInput: rawInput})
			}
		}

		// Execute all tools in parallel, preserving order
		type execResult struct {
			id     string
			name   string
			result string
			isErr  bool
		}
		execResults := make([]execResult, len(pending))
		var wg sync.WaitGroup
		for i, p := range pending {
			wg.Add(1)
			go func(i int, p pendingTool) {
				defer wg.Done()
				result, isErr := executeTool(ctx, a.svc, a.index, a.dbPath, p.variant.Name, p.rawInput)
				execResults[i] = execResult{id: p.variant.ID, name: p.variant.Name, result: result, isErr: isErr}
			}(i, p)
		}
		wg.Wait()

		// Build tool result blocks and emit tool_result events in order
		var toolResults []anthropic.ContentBlockParamUnion
		for _, r := range execResults {
			toolResults = append(toolResults, anthropic.NewToolResultBlock(r.id, r.result, r.isErr))
			emit(ChatEvent{Type: "tool_result", ToolName: r.name, ToolCallID: r.id, ToolResult: r.result, IsError: r.isErr})
		}

		// No tool calls — agent is done
		if len(toolResults) == 0 {
			emit(ChatEvent{Type: "done"})
			return nil
		}

		// Send tool results back
		a.messages = append(a.messages, anthropic.NewUserMessage(toolResults...))
	}
}

// Chat sends a user message and streams the agent's response to stdout.
// It runs the tool-use loop until the agent produces a text-only response.
func (a *Agent) Chat(ctx context.Context, userMessage string) error {
	return a.ChatStream(ctx, userMessage, func(ev ChatEvent) {
		switch ev.Type {
		case "text_delta":
			fmt.Print(ev.Text)
		case "tool_call":
			fmt.Fprintf(os.Stderr, "\n  [tool: %s]\n", ev.ToolName)
		case "done":
			fmt.Println()
		}
	})
}

// ClearHistory resets the conversation history.
func (a *Agent) ClearHistory() {
	a.messages = nil
}
