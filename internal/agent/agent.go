package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	"github.com/anthropics/anthropic-sdk-go"
	"github.com/anthropics/anthropic-sdk-go/option"
	"github.com/karim-daw/qwelli/internal/service"
)

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

// Chat sends a user message and streams the agent's response to stdout.
// It runs the tool-use loop until the agent produces a text-only response.
func (a *Agent) Chat(ctx context.Context, userMessage string) error {
	a.messages = append(a.messages, anthropic.NewUserMessage(anthropic.NewTextBlock(userMessage)))

	for {
		stream := a.client.Messages.NewStreaming(ctx, anthropic.MessageNewParams{
			Model:     a.model,
			MaxTokens: 8192,
			Messages:  a.messages,
			Tools:     a.tools,
			System:    a.system,
		})

		message := anthropic.Message{}
		for stream.Next() {
			event := stream.Current()
			if err := message.Accumulate(event); err != nil {
				return fmt.Errorf("accumulate stream event: %w", err)
			}

			// Stream text deltas to stdout in real-time
			switch ev := event.AsAny().(type) {
			case anthropic.ContentBlockDeltaEvent:
				if ev.Delta.Text != "" {
					fmt.Print(ev.Delta.Text)
				}
			}
		}
		if err := stream.Err(); err != nil {
			return fmt.Errorf("stream error: %w", err)
		}

		// Append the assistant's full response to history
		a.messages = append(a.messages, message.ToParam())

		// Collect tool calls
		var toolResults []anthropic.ContentBlockParamUnion
		for _, block := range message.Content {
			if variant, ok := block.AsAny().(anthropic.ToolUseBlock); ok {
				fmt.Fprintf(os.Stderr, "\n  [tool: %s]\n", variant.Name)

				result, isErr := executeTool(ctx, a.svc, a.index, a.dbPath, variant.Name, json.RawMessage(variant.JSON.Input.Raw()))
				toolResults = append(toolResults, anthropic.NewToolResultBlock(variant.ID, result, isErr))
			}
		}

		// No tool calls — agent is done
		if len(toolResults) == 0 {
			fmt.Println() // final newline
			return nil
		}

		// Send tool results back
		a.messages = append(a.messages, anthropic.NewUserMessage(toolResults...))
	}
}

// ClearHistory resets the conversation history.
func (a *Agent) ClearHistory() {
	a.messages = nil
}
