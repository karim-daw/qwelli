# Qwelli - Claude Code Guidelines

## Project Overview

Qwelli is a semantic search engine for local files. It indexes folders using Voyage AI embeddings, stores them in DuckDB with HNSW vector search, and supports multimodal content (text, PDFs, images). It provides both a CLI and a web UI built with React.

**Go version:** 1.25.5
**Current version:** 0.1.0

## Quick Commands

```bash
# Build (CLI only)
go build -o qwelli ./cmd/qwelli

# Build with embedded web UI (builds React frontend first)
./scripts/build-with-ui.sh

# Run all tests
go test ./...

# Run specific package tests
go test ./internal/db/...
go test ./internal/engine/...
go test ./internal/engine/search/...

# Run with verbose output
go test -v ./internal/engine/embeddings/...

# Run benchmarks
go test -bench=. ./internal/db/...
go test -bench=. ./internal/engine/...
```

## Architecture

```
cmd/qwelli/              # CLI entry point (cobra). Defaults to `serve` if no args.
internal/
  cli/                   # CLI commands: index, search, list, delete, shell, serve, init, status
  config/                # YAML config + env var loading (~/.qwelli/config.yaml)
  db/                    # DuckDB wrapper, schema, HNSW index, FTS search
  engine/                # Core business logic (does NOT manage DB connections)
    chunker/             # Text and PDF chunking with configurable chunk sizes
    differ/              # File change detection for incremental indexing
    embeddings/          # Embedding generation and storage
    extraction/          # PDF text extraction, image extraction, SHA256 hashing
    fileprocessor/       # File type routing and supported types
    search/              # Search strategies (semantic, keyword, hybrid)
  integration/           # End-to-end tests
  server/                # HTTP API + embedded React UI (go:embed)
  service/               # Service layer (owns DB lifecycle, single entry point)
  testutil/              # Test helpers and MockVoyageClient
  textutil/              # Sentence splitting and token counting
  voyage/                # Voyage AI client (embeddings + reranking)
web/                     # React frontend (Vite + TypeScript + Tailwind CSS)
scripts/                 # Build and deployment scripts
testdata/                # Test data (PDF samples)
```

## Key Patterns

### Service Layer (`internal/service/`)

All business operations go through `service.Service`. It owns the DB lifecycle -- callers never open databases directly.

```go
// Standard usage pattern
svc, err := service.Load()  // Loads config, creates Voyage client, creates engine
// No Close() needed on Service itself -- it doesn't hold a persistent DB connection

// Use service methods (each opens/closes DB internally)
svc.CreateIndex(ctx, folderPath, opts, progressCb)
svc.UpdateIndex(ctx, folderPath, opts, progressCb)     // Incremental re-index
svc.Search(folderPath, query, topK, contentType, strategy)
svc.SearchByDBPath(dbPath, query, topK, contentType, strategy)
svc.ListIndexes()
svc.DeleteIndex(folderPath)
svc.GetIndexStatus(folderPath)
svc.GetIndexStats(folderPath)

// Constructor with dependency injection (used in tests)
svc := service.New(cfg, mockClient)
```

### Engine API (`internal/engine/`)

Engine receives `*db.ProjectDB` from callers -- it does NOT manage DB connections.

```go
eng := engine.NewEngine(voyageClient, enableMultimodal)
eng.SetParallelProcessing(true, numWorkers)    // Enable parallel file processing
eng.SetParallelPDFProcessing(true, numWorkers) // Enable parallel PDF page processing
eng.SetContentTypeMode(mode)                   // text, image, or both

eng.IndexFolder(ctx, projectDB, folderPath, incremental, progressCb, phaseCb)
eng.Search(projectDB, query, topK, contentType, strategy)
eng.GetIndexStatus(projectDB, folderPath)
eng.DetectDimension()  // Makes test API call to detect vector dimension
```

### DB Layer (`internal/db/`)

DuckDB with the VSS extension for HNSW vector search.

```go
db.OpenProjectDB(path, dimension) // Opens/creates database, loads VSS extension
db.GetDimensionFromDB(path)       // Reads dimension from existing DB (read-only)
projectDB.Close()                 // Must be called by the opener

// Schema tables: files, chunks, embeddings, metadata
// HNSW index must be rebuilt after embedding changes
projectDB.RebuildHNSWIndex()
projectDB.BuildHNSWIndexIfNeeded(prevCount)
```

**Database schema (v2):**
- `files` -- file metadata (path, type, hash, modified_at, size)
- `chunks` -- text/image chunks with content, page numbers, content_type
- `embeddings` -- float vectors dimensioned to the model output
- `metadata` -- key-value store (dimension, model, folder_path, schema_version)

### Search Strategies (`internal/engine/search/`)

All strategies implement `SearchStrategy` interface:

```go
type SearchStrategy interface {
    Search(query string, projectDB *db.ProjectDB, topK int, contentType string) ([]db.SearchResult, error)
    Name() string
}
```

- `semantic` -- Vector similarity via HNSW (default)
- `keyword` -- Full-text search with TF-IDF scoring
- `hybrid` -- Combines both with Reciprocal Rank Fusion (RRF)

### Voyage AI Client (`internal/voyage/`)

All Voyage client interactions go through `voyage.ClientInterface`:

```go
type ClientInterface interface {
    Embed(text string) ([]float32, error)
    EmbedBatch(ctx context.Context, texts []string, progressCallback func(current, total int)) ([][]float32, error)
    EmbedMultimodal(ctx context.Context, inputs []MultimodalInput, progressCallback func(current, total int)) ([][]float32, error)
    Rerank(ctx context.Context, query string, documents []string) ([]RerankResult, error)
    RerankWithMetadata(ctx context.Context, query string, inputs []RerankInput) ([]RerankResult, error)
    EmbeddingModel() string
    RerankModel() string
}
```

### HTTP Server (`internal/server/`)

The server embeds the React UI via `go:embed web/dist` and serves it alongside the API.

**API routes:**
- `GET /api/indexes` -- List all indexes
- `GET /api/search` -- Search an index
- `POST /api/index/create` -- Create a new index
- `GET /api/index/status` -- Get index status
- `POST /api/index/update` -- Incremental update
- `GET /api/index/progress` -- SSE progress stream
- `POST /api/index/cancel` -- Cancel indexing
- `POST /api/index/delete` -- Delete an index
- `GET /api/file` -- File access
- `POST /api/open-folder` -- Open folder in OS
- `POST /api/open-file-location` -- Open file location in OS
- `GET /api/terminal/stream` -- SSE terminal output stream
- `POST /api/shutdown` -- Graceful shutdown
- `GET /api/setup/status` -- Config/setup status

### CLI Entry Point (`cmd/qwelli/main.go`)

When run without arguments (e.g., double-click), defaults to `serve` mode. Otherwise uses Cobra for routing to 8 subcommands: `shell`, `serve`, `init`, `index`, `search`, `list`, `status`, `delete`.

## Environment Variables

```bash
VOYAGE_API_KEY=...              # Required - Voyage AI API key
VOYAGE_MODEL=voyage-multimodal-3  # Embedding model (default)
VOYAGE_EMBEDDING_ENDPOINT=...  # Custom API endpoint (optional)
VOYAGE_RERANK_MODEL=...        # Reranker model (optional)
VOYAGE_RERANK_ENDPOINT=...     # Reranker endpoint (optional)
ENABLE_RERANKER=true           # Enable/disable reranking (true/false)
```

Environment variables override config file values.

## Config

- Config file: `~/.qwelli/config.yaml`
- Index storage: `~/.qwelli/indexes/*.db`
- Run `qwelli init` to create initial config

**Config fields** (see `internal/config/config.go`):
- Embedding: `embedding_provider`, `api_key`, `model`, `endpoint`
- Multimodal: `enable_multimodal` (default: true), `image_quality`
- Reranker: `enable_reranker` (default: true), `rerank_provider`, `rerank_model`, `rerank_endpoint`
- Parallel processing: `enable_parallel` (default: true), `parallel_workers` (0 = auto ~90% CPUs), `enable_parallel_pdf` (default: true), `parallel_pdf_workers`
- Storage: `index_dir`

## Supported File Types

**Text:** txt, md, go, py, js, ts, tsx, jsx, java, c, cpp, h, rs, rb, php, cs, swift, html, css, scss, yaml, yml, toml, sh, proto, graphql

**PDF:** pdf (text extraction + image extraction from pages)

**Images:** jpg, jpeg, png, gif (multimodal embeddings)

See `internal/engine/fileprocessor/types.go` for the authoritative list.

Text files >500KB are skipped. Files >50MB use streaming hash instead of memory-loaded processing.

## Testing

### Running tests

```bash
go test ./...                                   # All tests
go test ./internal/db/...                       # Database tests
go test ./internal/engine/search/...            # Search strategy tests
go test -v -run TestSpecificName ./internal/... # Specific test
```

### Test conventions

- Tests requiring `VOYAGE_API_KEY` must use `t.Skip()` (not `t.Fatal()`) when the env var is missing
- Use `internal/testutil` for test helpers:
  - `testutil.NewMockVoyageClient(dimension)` -- deterministic mock embeddings
  - `testutil.CreateTestConfig(t)` -- config with mock settings
  - `testutil.CreateTestEngine(t, dimension)` -- engine with mock client
  - `testutil.CreateTestFiles(t, dir, files)` -- create test file fixtures
  - `testutil.CreateTestIndex(t, indexDir, folderPath, files, dimension)` -- full test index
  - `testutil.IndexTestData(t, eng, projectDB, folderPath, mockClient)` -- index test data
  - `testutil.AssertSearchResults(t, results, count, content)` -- verify search results
- PDF tests look for files in `testdata/pdf_samples/`
- E2E tests are in `internal/integration/e2e_test.go`
- MockVoyageClient generates deterministic embeddings from SHA256 hashes of input text

### Benchmark tests

Benchmark files exist in `internal/engine/` and `internal/db/`:
- `benchmark_test.go` -- general benchmarks
- `benchmark_pdf_parallel_test.go` -- parallel PDF processing
- `benchmark_realdata_test.go` -- real data benchmarks

## Build & Deployment

### Build scripts (`scripts/`)

- `build-with-ui.sh` -- Full build: npm install, React build, copy dist, Go build
- `build-release.sh` -- Cross-compile for Windows (`.exe` + DLL)
- `build-windows.bat` / `build-with-ui.bat` -- Windows-specific builds
- `download-duckdb.bat` -- Download DuckDB library for Windows
- `DEMO.sh` -- Demo script

### Web UI build process

The React frontend in `web/` is built with Vite and embedded into the Go binary:

1. `cd web && npm install && npm run build` -- Build React app to `web/dist/`
2. Copy `web/dist/` to `internal/server/web/dist/`
3. Go binary embeds it via `//go:embed web/dist` in `internal/server/server.go`

## Key Dependencies

| Package | Purpose |
|---------|---------|
| `github.com/duckdb/duckdb-go/v2` | DuckDB SQL database with VSS extension |
| `github.com/spf13/cobra` | CLI framework |
| `github.com/ledongthuc/pdf` | PDF text parsing |
| `github.com/pdfcpu/pdfcpu` | PDF manipulation and image extraction |
| `github.com/joho/godotenv` | .env file loading |
| `gopkg.in/yaml.v3` | YAML config parsing |

## Common Tasks

### Adding a new CLI command

1. Create `internal/cli/newcmd.go`
2. Add `NewNewcmdCmd()` function returning `*cobra.Command`
3. Register in `cmd/qwelli/main.go` via `rootCmd.AddCommand()`

### Adding a new API endpoint

1. Add handler in `internal/server/handler_*.go`
2. Register route in `server.go` `Start()` method
3. Add request/response types to `types.go` if needed

### Adding a new search strategy

1. Create `internal/engine/search/newstrategy.go`
2. Implement `SearchStrategy` interface (`Search()` and `Name()`)
3. Add case in `engine.go` `buildStrategy()` switch

### Adding a new file type

1. Add extension to the appropriate map in `internal/engine/fileprocessor/types.go`
2. If needed, add processing logic in `fileprocessor/service.go`

### Modifying search behavior

- Strategies are in `internal/engine/search/`
- Each implements `SearchStrategy` interface
- Hybrid strategy combines semantic + keyword using Reciprocal Rank Fusion

### Modifying the database schema

1. Update `internal/db/schema.go` `buildSchema()` function
2. Update `internal/db/models.go` if data structures change
3. Update `internal/db/queries.go` for new query methods
4. Increment `schema_version` in metadata

## Don't

- Don't open databases directly in CLI/server code -- always use the service layer
- Don't use `t.Fatal()` for missing API keys in tests -- use `t.Skip()`
- Don't forget to rebuild the HNSW index after modifying embeddings
- Don't commit `.env` files or API keys
- Don't access `projectDB` concurrently from multiple goroutines without synchronization (DuckDB Appender is not thread-safe)
- Don't skip calling `projectDB.Close()` -- it must be called by whichever code opened the DB
- Don't add new file types without updating `fileprocessor/types.go`
