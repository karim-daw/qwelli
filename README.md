# Qwelli

Local semantic file search using vector embeddings. Index your folders and find files by meaning, not keywords. Comes with a built-in web UI and an AI chat agent for natural-language questions about your documents.

## Quick Start

### 1. Prerequisites

- Go 1.25+ with CGO enabled
- Node.js 18+ (for building the web UI)
- Voyage AI API key ([get one here](https://dash.voyageai.com/))

### 2. Build

```bash
# Full build: frontend + Go binary (Linux/Mac)
./scripts/build.sh

# Full build: frontend + Go binary (Windows)
scripts\build.bat

# Release build (stripped, no console window on Windows)
./scripts/build.sh --release
scripts\build.bat --release

# Build Go binary only (requires web/dist to exist)
go build -o qwelli ./cmd/qwelli
```

### 3. Run

```bash
# Start the web UI (opens browser automatically)
./qwelli

# Or explicitly
./qwelli serve
./qwelli serve --port 3000  # Custom port
```

On first launch, the browser opens to a setup screen where you enter your Voyage AI API key. After that, you're ready to index and search.

## Windows

Building natively on Windows produces a single `.exe` with DuckDB statically linked — no separate DLL needed.

1. Install MSYS2 from https://www.msys2.org/
2. Run `pacman -S mingw-w64-ucrt-x86_64-gcc` in MSYS2
3. Add `C:\msys64\ucrt64\bin` to PATH
4. Run `scripts\build.bat`

> **Note:** Cross-compiling from Linux requires `gcc-mingw-w64-x86-64` and produces a `.exe` + `duckdb.dll` pair instead of a single binary.

## CLI Usage

```bash
# Setup config (API key, model)
qwelli init

# Index a folder
qwelli index ~/Documents/project

# Search indexed files
qwelli search "machine learning" --index ~/Documents/project
qwelli search "api docs" -i ~/Documents/project -t 10

# List all indexes
qwelli list

# Show index status
qwelli status --index ~/Documents/project

# Delete an index
qwelli delete --index ~/Documents/project

# Interactive shell
qwelli shell

# Start web UI
qwelli serve

# AI chat about your documents (requires Foundry config)
qwelli chat --index ~/Documents/project
```

## AI Chat

`qwelli chat` is an interactive AI agent that answers natural-language questions about your indexed documents. It uses Claude on Azure AI Foundry and has access to your document index through a set of tools.

### Setup

Set the following environment variables (or add them to `~/.qwelli/config.yaml`):

```bash
export FOUNDRY_ENDPOINT="https://your-endpoint.azure.com"
export FOUNDRY_API_KEY="your-api-key"
export FOUNDRY_MODEL="claude-sonnet-4-6"  # default
```

### Usage

```bash
qwelli chat --index ~/Documents/project
```

This starts a REPL where you can ask questions like "What files discuss authentication?" or "Summarize the PDF reports from last month." Press Ctrl+C to cancel a response.

### Agent Tools

The agent has access to 8 tools that operate on your indexed collection:

| Tool | Description |
|------|-------------|
| `search` | Semantic, keyword, or hybrid search across indexed documents |
| `status` | Check index status — total files, pending changes |
| `read_file` | Read full text contents of a file (text files only) |
| `list_dir` | List directory contents within the indexed folder |
| `index_update` | Incremental re-index to pick up new/modified/deleted files |
| `get_file_chunks` | Get all indexed chunks for a file, including PDFs |
| `get_file_info` | File metadata — type, size, dates, chunk count |
| `find_files` | Query indexed files by type, name pattern, date range, subfolder |

## Web UI Features

- **Semantic, keyword, and hybrid search** with strategy toggle
- **Content type filtering** (text, images, or both)
- **PDF preview** with page navigation
- **Index management** -- create, sync, delete indexes from the sidebar
- **Live terminal** showing real-time server logs
- **Light/dark mode** toggle
- **Resizable panels** (sidebar and terminal)
- **Search caching** with recent search history
- **Quit button** for graceful server shutdown from the browser

## Search Strategies

| Strategy | Description |
|----------|-------------|
| `semantic` | Vector similarity search (default) |
| `keyword` | Full-text search with TF-IDF scoring |
| `hybrid` | Combines both with Reciprocal Rank Fusion |

## Supported File Types

**Text:** `.txt`, `.md`, `.go`, `.py`, `.js`, `.ts`, `.tsx`, `.jsx`, `.java`, `.c`, `.cpp`, `.h`, `.rs`, `.rb`, `.php`, `.cs`, `.swift`, `.html`, `.css`, `.scss`, `.yaml`, `.yml`, `.toml`, `.sh`, `.proto`, `.graphql`

**PDF:** Full text extraction with optional image extraction from pages

**Images:** `.jpg`, `.jpeg`, `.png`, `.gif` — standalone image files embedded via multimodal model

Files >500KB and hidden files/folders are skipped.

## Configuration

Config is stored at `~/.qwelli/config.yaml`. Set up via `qwelli init` (CLI) or the web UI setup screen on first launch.

Environment variables override config file values:

| Variable | Description |
|----------|-------------|
| `VOYAGE_API_KEY` | Voyage AI API key (required) |
| `VOYAGE_MODEL` | Embedding model (default: `voyage-multimodal-3`) |
| `VOYAGE_EMBEDDING_ENDPOINT` | API endpoint |
| `VOYAGE_RERANK_MODEL` | Reranker model |
| `VOYAGE_RERANK_ENDPOINT` | Reranker endpoint |
| `ENABLE_RERANKER` | Enable/disable reranking (`true`/`false`) |
| `FOUNDRY_ENDPOINT` | Azure AI Foundry base URL (required for `qwelli chat`) |
| `FOUNDRY_API_KEY` | Foundry API key (required for `qwelli chat`) |
| `FOUNDRY_MODEL` | Foundry deployment name (default: `claude-sonnet-4-6`) |

## Data Storage

All data is local:

- **Config:** `~/.qwelli/config.yaml`
- **Indexes:** `~/.qwelli/indexes/*.db`

Each indexed folder gets its own DuckDB database with HNSW vector index.

## Embedding Models

| Model | Dimension | Notes |
|-------|-----------|-------|
| `voyage-multimodal-3` | 1024 | Text + images (default) |
| `voyage-3` | 1024 | Text-only |

## Project Structure

```
qwelli/
├── cmd/qwelli/          # CLI entry point
├── internal/
│   ├── agent/           # AI chat agent (tool definitions, streaming loop)
│   ├── cli/             # Commands (init, index, search, serve, shell, chat, etc.)
│   ├── config/          # YAML config + env var loading
│   ├── db/              # DuckDB wrapper, HNSW index, FTS search
│   ├── engine/          # Core business logic
│   │   ├── chunker/     # Text chunking
│   │   ├── differ/      # File change detection
│   │   ├── embeddings/  # Embedding generation
│   │   ├── extraction/  # PDF text extraction
│   │   ├── fileprocessor/ # File type handling
│   │   └── search/      # Search strategies (semantic, keyword, hybrid)
│   ├── server/          # HTTP API + embedded React UI
│   ├── service/         # Service layer (owns DB lifecycle)
│   ├── voyage/          # Voyage AI client
│   └── textutil/        # Text utilities
├── web/                 # React frontend (Vite + TypeScript)
├── scripts/             # Build scripts
└── dist/                # Built binaries
```

## How It Works

1. **Index:** Scan folder -> chunk files -> generate embeddings via Voyage AI -> store in DuckDB with HNSW index
2. **Search:** Embed query -> approximate nearest neighbor search -> optional reranking -> return matches

One embedding model per database. Changing model requires re-indexing.

## Testing

```bash
# All tests
go test ./...

# Specific packages
go test ./internal/db/... -v
go test ./internal/engine/... -v

# With coverage
go test ./... -cover
```

## Cost

Using Voyage AI `voyage-multimodal-3`:

- ~$0.02 per 1,000 documents indexed
- ~$0.0001 per search query
