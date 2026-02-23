# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

Qwelli is a local semantic file search engine. It indexes folders using Voyage AI embeddings (default model: `voyage-multimodal-3.5`) and stores them in DuckDB with HNSW vector search. It ships as a single binary with an embedded React web UI. It also includes a standalone AI agent (`qwelli chat`) that lets users ask natural-language questions about their indexed document collections using Claude on Azure AI Foundry.

## Build & Test Commands

```bash
# Full build: frontend + Go binary (Windows)
scripts\build.bat

# Full build: frontend + Go binary (Linux/Mac)
./scripts/build.sh

# Release build (stripped binary, no console window on Windows)
scripts\build.bat --release
./scripts/build.sh --release

# Build Go binary only (requires web/dist to exist for embedded UI)
go build -o qwelli ./cmd/qwelli

# Build web frontend separately
cd web && npm install && npm run build
# Then copy web/dist/ to internal/server/web/dist/

# Run all tests
go test ./...

# Run specific package tests
go test ./internal/db/...
go test -v ./internal/engine/chunker/...

# Run a single test
go test -v -run TestFunctionName ./internal/engine/...
```

**Build requirements:** Go 1.25+, CGO enabled (DuckDB requires it). On Windows, needs MSYS2 with `mingw-w64-ucrt-x86_64-gcc` and `C:\msys64\ucrt64\bin` in PATH. DuckDB is statically linked via platform-specific Go bindings (`duckdb-go-bindings/windows-amd64`, etc.) — no separate DLL or library download needed.

## Architecture

### Layered Design

```
CLI/Server → Service → Engine → DB
                ↘ Voyage Client (embeddings API)
```

- **Service layer** (`internal/service/`) owns DB lifecycle. CLI and server code never open databases directly — they call `service.Load()` or `service.New()`.
- **Engine** (`internal/engine/engine.go`) receives `*db.ProjectDB` from callers. It coordinates file processing, embedding, and search but doesn't manage connections.
- **DB layer** (`internal/db/`) wraps DuckDB with vector search. Schema is in `schema.go`. Chunks table denormalizes `file_path` and `file_type` from files table for JOIN-free search queries.

### Key Interfaces

- `voyage.ClientInterface` — abstraction over Voyage AI HTTP API (embeddings + reranking). Used for dependency injection in tests.
- `search.SearchStrategy` — interface for search implementations (`semantic`, `keyword`, `hybrid` via RRF fusion).
- `fileprocessor.FileProcessingService` — unified file processor that dispatches by type (text, PDF, image).

### Web UI Architecture

The React frontend (`web/`) is built to `web/dist/`, copied to `internal/server/web/dist/`, and embedded in the Go binary via `//go:embed`. The server serves it as static files at the root route.

Uses **shadcn/ui** (Radix primitives + Tailwind CSS), **sonner** for toasts, and **react-pdf** for PDF preview.

```
web/src/
  api/          # Typed fetch wrapper (client.ts) + per-feature modules (search, indexes, chat)
  types/        # TypeScript interfaces mirroring server/types.go + chat event/message types
  contexts/     # AppContext (indexes, viewMode) + SearchContext (query, results, recent searches)
  hooks/        # useTheme, useSSE, useResizable, useSearch, useIndexProgress, useChat, etc.
  components/
    ui/         # shadcn/ui generated components (Button, Dialog, Card, etc.)
    layout/     # AppLayout, Sidebar, TopBar, MainContent
    search/     # SearchForm, SearchResults, ResultCard, RecentSearches
    status/     # StatusView, StatusSummaryGrid, FileChangeList
    modals/     # NewIndexDialog, IndexProgressModal, FullTextModal, PDFPreviewModal
    chat/       # ChatView, ChatMessage, ChatInput, ToolCallBlock
    screens/    # SetupScreen, QuitScreen
    terminal/   # TerminalPanel
  lib/          # cn() utility, format helpers
  App.tsx       # Slim provider shell (~40 lines)
```

**Key patterns:**
- **Contexts** use plain `useState` with setter functions exposed via context — no `useReducer`. Context values are wrapped in `useMemo` to prevent unnecessary re-renders.
- **Theme** uses Tailwind `dark:` variants and CSS variable classes (`bg-background`, `text-foreground`, `bg-muted`, etc.) instead of runtime `isDark` conditionals. The `.dark` class is toggled on `<html>` by `useTheme`.
- **API client** (`client.ts`) — typed `get`/`post`/`rawPost` wrappers accept an optional `AbortSignal` for request cancellation.
- **Search** — in-flight requests are cancelled via `AbortController` when a new search starts or the component unmounts. Recent searches strip large fields (`imageData`, `content`) to keep localStorage lightweight. Recent searches are stored in `SearchContext` (backed by localStorage, keyed by index path).
- **Chat** — `ChatView` pins the input at the bottom with a gradient fade. Auto-scroll only triggers when the user is near the bottom; a floating scroll-to-bottom button appears when scrolled up. `ToolCallBlock` uses Lucide icons with colored icon pills and CSS grid animated expand/collapse. SSE stream readers use `try/finally` to release locks.
- **SSE** — `useSSE` hook for terminal streaming; `useIndexProgress` manages its own EventSource for index/update progress with cancel support. EventSources are closed on unmount.
- **Terminal** — logs are capped at 500 entries to prevent memory growth.
- **Modals** — `FullTextModal` and `NewIndexDialog` use shadcn `Dialog`. `PDFPreviewModal` uses a plain overlay (shadcn Dialog's base classes conflict with the full-height flex layout needed for the PDF viewer).

### Server Patterns

- **Port resolution** (`listener.go`): `ResolveListener(port, portExplicit)` binds the socket before the server starts. Default port (8080) auto-falls back to an OS-assigned port if busy; explicit `--port` fails with a clear error. Both `Server` and `SetupServer` expose a `Listen(portExplicit)` method — call it before `Start()`. The listener-first approach also eliminates the need for `time.Sleep` before opening the browser.
- SSE (Server-Sent Events) for real-time indexing progress (`/api/index/progress`) and terminal output (`/api/terminal/stream`).
- Search results cached in-memory with 5-minute TTL.
- Background indexing with cancellation support via context.
- Setup server (`setup_server.go`) runs on first launch when no config exists, then restarts as the main server.

### Indexing Pipeline

`resolving` → `processing` → `embedding` → `storing` → `hnsw` → `complete`

Files are scanned, text/images extracted (with parallel workers), embeddings generated via Voyage API in batches, stored in DuckDB, then the HNSW index is rebuilt if embeddings changed.

### Agent Architecture

```
User ↔ CLI (chat.go) ↔ Agent Loop (internal/agent/) ↔ Azure AI Foundry
                                    ↕                    (Claude via Anthropic SDK)
                              Tool Executor
                         ┌──────┼──────┐
                      Service  Filesystem  DB
```

- **Agent loop** (`internal/agent/agent.go`) — streaming tool-use loop using `anthropic-sdk-go`. Calls `Messages.NewStreaming()`, accumulates events, dispatches tool calls, sends results back, repeats until text-only response.
- **Tools** (`internal/agent/tools.go`) — 8 tools that map to existing service/DB/filesystem operations:
  - `search` — semantic/keyword/hybrid search via `svc.SearchByDBPath()`
  - `status` — index status via `svc.GetIndexStatus()`
  - `read_file` — text file contents (rejects PDFs/binaries, security-bounded to index folder)
  - `list_dir` — directory listing within index folder
  - `index_update` — incremental re-index via `svc.UpdateIndex()`
  - `get_file_chunks` — full indexed content of any file (including PDFs) via `projectDB.GetChunksForFile()`
  - `get_file_info` — single-file metadata + chunk count from the index DB
  - `find_files` — query indexed files by type, name pattern, date range, subfolder
- **CLI** (`internal/cli/chat.go`) — `qwelli chat --index <path>` REPL with Ctrl+C per-turn cancellation.
- **Config** — `FOUNDRY_ENDPOINT`, `FOUNDRY_API_KEY`, `FOUNDRY_MODEL` env vars or `config.yaml` fields.

## Environment Variables

```bash
VOYAGE_API_KEY=...              # Required for indexing/search
VOYAGE_MODEL=voyage-multimodal-3.5  # Embedding model (default)
VOYAGE_EMBEDDING_ENDPOINT=...  # API endpoint
VOYAGE_RERANK_MODEL=...        # Optional reranker
VOYAGE_RERANK_ENDPOINT=...     # Optional reranker endpoint
ENABLE_RERANKER=true           # Enable/disable reranking

FOUNDRY_ENDPOINT=...           # Required for `qwelli chat` (Azure AI Foundry base URL)
FOUNDRY_API_KEY=...            # Required for `qwelli chat`
FOUNDRY_MODEL=claude-sonnet-4-6  # Foundry deployment name (default)
```

Environment variables override `~/.qwelli/config.yaml` values.

## Data Storage

- Config: `~/.qwelli/config.yaml`
- Indexes: `~/.qwelli/indexes/*.db` (one DuckDB file per indexed folder)

## Adding a New CLI Command

1. Create `internal/cli/newcmd.go` with `NewNewcmdCmd()` returning `*cobra.Command`
2. Register in `cmd/qwelli/main.go` via `rootCmd.AddCommand()`

## Adding a New Frontend Component

1. Create the component in the appropriate `web/src/components/` subdirectory
2. Use shadcn/ui primitives (`Button`, `Card`, `Dialog`, etc.) — don't hand-roll HTML equivalents
3. Use CSS variable classes for theming (`bg-background`, `text-muted-foreground`, `dark:text-red-400`) — never `isDark` ternaries
4. For API calls, add a function in the relevant `web/src/api/` module, not inline `fetch()`
5. For shared state, use `useAppContext()` or `useSearchContext()` — not prop drilling
6. Run `cd web && npx tsc --noEmit && npm run build` to verify

## Adding a New API Endpoint

1. Add handler in `internal/server/handler_*.go`
2. Register route in `server.go` `setupRoutes()`
3. Add request/response types to `types.go` if needed

## Adding an Agent Tool

1. Add tool definition in `internal/agent/tools.go` `toolDefs()` — use `anthropic.ToolParam` with `Name`, `Description`, and `InputSchema` (JSON Schema)
2. Add dispatch case in `executeTool()` switch
3. Implement `exec<ToolName>()` function — takes raw JSON input, validates, calls service/DB/filesystem, returns `(string, bool)` (result, isError)
4. Security: all file/directory tools must validate paths are within the indexed folder
5. Update system prompt in `internal/agent/agent.go` to guide the agent on when to use the new tool

## Testing Notes

- Tests requiring `VOYAGE_API_KEY` use `t.Skip()` when the key is missing — never `t.Fatal()`.
- PDF test fixtures go in `testdata/pdf_samples/`.
- The `voyage.ClientInterface` can be mocked for unit tests.

## Rules

- Never open databases directly outside the service layer.
- HNSW index must be rebuilt after embedding changes (`BuildHNSWIndexIfNeeded` or `RebuildHNSWIndex`).
- Changing the embedding model requires re-creating the database (dimension is fixed at creation time). The service layer detects this automatically via `handleModelChange()`.
- Files >500KB are skipped. OneDrive placeholder files are detected and skipped.
- Never use `alert()` or `confirm()` in the frontend — use `toast` (sonner) and `AlertDialog` (shadcn).
- Use Lucide icons in the frontend — never Unicode symbol characters for icons.
- Never add `isDark` ternaries — use Tailwind `dark:` variants and CSS variable classes.
- API calls go in `web/src/api/` modules, not inline in components.
- Shared state goes in contexts, not prop-drilled through intermediate components.
- Agent tools must security-bound all file/directory access to the indexed folder path.
- Agent tool results are strings — use JSON for structured data, plain text for file contents.
- The Anthropic SDK `NewClient()` returns by value (not pointer) — `anthropic.Client`, not `*anthropic.Client`.
- Always use `--admin` flag when merging PRs with `gh pr merge`.
