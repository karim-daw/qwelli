# Qwelli

Local semantic file search using vector embeddings. Index your folders and find files by meaning, not keywords. Comes with a built-in web UI.

## Quick Start

### 1. Prerequisites

- Go 1.21+
- Node.js 18+ (for building the web UI)
- Voyage AI API key ([get one here](https://dash.voyageai.com/))

### 2. Build

```bash
# Build with embedded web UI (recommended)
./scripts/build-with-ui.sh

# Or build Go binary only (if web/dist already exists)
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

## Windows Distribution

Build a Windows `.exe` from Linux/WSL for sharing with colleagues:

```bash
./scripts/build-release.sh
```

This produces `qwelli-windows-amd64.zip` containing:
- `qwelli.exe` -- double-click to launch (no console window)
- `duckdb.dll` -- required runtime library (must stay next to the exe)

The colleague extracts the zip, double-clicks `qwelli.exe`, and the browser opens. First launch shows a setup screen for the API key.

> **Note:** Cross-compiling from Linux requires `gcc-mingw-w64-x86-64` (the script installs it automatically). Building natively on Windows produces a single `.exe` with DuckDB statically linked.

### Building natively on Windows

If building on Windows directly (single `.exe`, no DLL needed):

1. Install MSYS2 from https://www.msys2.org/
2. Run `pacman -S mingw-w64-ucrt-x86_64-gcc` in MSYS2
3. Add `C:\msys64\ucrt64\bin` to PATH
4. Run `scripts\build-with-ui.bat`

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
```

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

**Text:** `.txt`, `.md`, `.go`, `.py`, `.js`, `.ts`, `.java`, `.c`, `.cpp`, `.rs`, `.rb`, `.php`, `.html`, `.css`, `.yaml`, `.yml`, `.toml`, `.sh`, `.json`, `.proto`, `.graphql`

**PDF:** Full text extraction with optional image extraction from pages

**Images:** Image embeddings via multimodal model (from PDFs)

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
│   ├── cli/             # Commands (init, index, search, serve, shell, etc.)
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
