# Qwelli - Claude Code Guidelines

## Project Overview

Qwelli is a semantic search engine for local files. It indexes folders using Voyage AI embeddings and stores them in DuckDB with HNSW vector search.

## Quick Commands

```bash
# Build
go build -o qwelli ./cmd/qwelli

# Build with web UI
./scripts/build-full.sh

# Run tests
go test ./...

# Run specific package tests
go test ./internal/db/...
go test ./internal/engine/...

# Run with verbose output
go test -v ./internal/engine/embeddings/...
```

## Architecture

```
cmd/qwelli/          # CLI entry point (cobra)
internal/
  cli/               # CLI commands (index, search, list, delete, shell, serve)
  config/            # YAML config + env var loading (~/.qwelli/config.yaml)
  db/                # DuckDB wrapper, HNSW index, FTS search
  engine/            # Core business logic
    chunker/         # Text chunking
    differ/          # File change detection
    embeddings/      # Embedding generation
    extraction/      # PDF text extraction
    fileprocessor/   # File type handling
    search/          # Search strategies (semantic, keyword, hybrid)
  server/            # HTTP API + embedded React UI
  service/           # Service layer (owns DB lifecycle)
  voyage/            # Voyage AI client
  textutil/          # Text utilities
web/                 # React frontend (Vite + TypeScript)
```

## Key Patterns

### Service Layer
All business operations go through `service.Service`. It owns DB lifecycle - callers never open databases directly.

```go
svc, err := service.Load()  // Loads config + creates voyage client + engine
defer svc.Close()           // If applicable

// Use service methods
svc.CreateIndex(ctx, folderPath, opts, progressCb)
svc.Search(folderPath, query, topK, contentType, strategy)
svc.ListIndexes()
```

### Engine API
Engine receives `*db.ProjectDB` from callers - it doesn't manage DB connections.

```go
eng.IndexFolder(ctx, projectDB, folderPath, incremental, progressCb, phaseCb)
eng.Search(projectDB, query, topK, contentType, strategy)
eng.GetIndexStatus(projectDB, folderPath)
```

### DB Layer
- `db.OpenProjectDB(path, dimension)` - opens/creates database
- `db.GetDimensionFromDB(path)` - reads dimension from existing DB
- HNSW index must be rebuilt after embedding changes

## Environment Variables

```bash
VOYAGE_API_KEY=...              # Required
VOYAGE_MODEL=voyage-3           # Embedding model
VOYAGE_EMBEDDING_ENDPOINT=...   # API endpoint
VOYAGE_RERANK_MODEL=...         # Optional reranker
VOYAGE_RERANK_ENDPOINT=...      # Optional reranker endpoint
```

## Config Location

- Config: `~/.qwelli/config.yaml`
- Indexes: `~/.qwelli/indexes/*.db`

## File Types Supported

PDF, TXT, MD, GO, PY, JS, TS, JSON, YAML, and more (see `fileprocessor.IsSupported`)

## Search Strategies

- `semantic` - Vector similarity (default)
- `keyword` - Full-text search with TF-IDF scoring
- `hybrid` - Combines both with RRF fusion

## Testing Notes

- API tests in `indexer_test.go` require `VOYAGE_*` env vars (skipped if missing)
- PDF tests look for files in `testdata/pdf_samples/`
- Use `t.Skip()` not `t.Fatal()` for missing external dependencies

## Common Tasks

### Adding a new CLI command
1. Create `internal/cli/newcmd.go`
2. Add `NewNewcmdCmd()` function returning `*cobra.Command`
3. Register in `cmd/qwelli/main.go`

### Adding a new API endpoint
1. Add handler in `internal/server/handler_*.go`
2. Register route in `server.go` setupRoutes()
3. Add types to `types.go` if needed

### Modifying search behavior
- Strategies are in `internal/engine/search/`
- Each implements `SearchStrategy` interface
- Hybrid strategy combines results using RRF

## Don't

- Don't open databases directly in CLI/server code - use service layer
- Don't use `t.Fatal()` for missing API keys in tests - use `t.Skip()`
- Don't forget to rebuild HNSW index after modifying embeddings
- Don't commit `.env` files or API keys
