# Qwelli

Local semantic file search using vector embeddings. Index your folders and find files by meaning, not keywords.

## Prerequisites

- Go 1.21+
- Voyage AI API key
- **Windows only:** GCC compiler (for CGO/DuckDB)

## Windows Setup (GCC for CGO)

DuckDB requires CGO which needs a C compiler. Install MSYS2:

1. Download from https://www.msys2.org/
2. Run installer, use default path `C:\msys64`
3. Open MSYS2 terminal, run:
   ```bash
   pacman -S mingw-w64-ucrt-x86_64-gcc
   ```
4. Add to PATH permanently:
   - Press `Win+R`, type `sysdm.cpl`, press Enter
   - Advanced → Environment Variables
   - Under "User variables", edit `Path`
   - Add: `C:\msys64\ucrt64\bin`
   - Click OK, restart terminal
5. Verify:
   ```powershell
   gcc --version
   ```

## Quick Start

### 1. Get Voyage AI API Key

1. Go to https://www.voyageai.com/
2. Sign up and create a new API key
3. Copy it

### 2. Configure `.env`

Create a `.env` file in the project root:

```
QWELLI_EMBEDDING_KEY=your-voyage-api-key
QWELLI_EMBEDDING_MODEL=voyage-multimodal-3
QWELLI_EMBEDDING_ENDPOINT=https://api.voyageai.com/v1/multimodalembeddings
```

### 3. Build and Run

#### CLI Mode

```bash
# Build the binary
go build -o qwelli ./cmd/qwelli

# Or use individual commands
./qwelli init
./qwelli index ./my-folder
./qwelli search "query" --index ./my-folder

# Run interactive shell
./qwelli shell
```

#### Web UI Mode

```bash
# Build with embedded web UI
./build-with-ui.sh  # Linux/Mac
build-with-ui.bat   # Windows

# Start the web server
./qwelli serve

# Open browser to http://localhost:8080
```

**Important:** The `shell` command requires an interactive terminal. Always use the built binary (`./qwelli`) instead of `go run` for reliable operation, especially for the interactive shell.

## Usage

### Commands

#### `init`

Setup configuration (API key, model).

```bash
qwelli init
```

#### `index <folder>`

Index all files in a folder.

```bash
qwelli index ~/Documents/project
```

#### `search <query>`

Search indexed files.

```bash
qwelli search "machine learning" --index ~/Documents/project
qwelli search "api docs" -i ~/Documents/project -t 10
```

Options:

- `--index, -i` - Path to indexed folder
- `--top, -t` - Number of results (default: 5)

#### `list`

Show all indexed folders.

```bash
qwelli list
```

#### `status`

Show index statistics.

```bash
qwelli status --index ~/Documents/project
```

#### `shell`

Interactive mode.

```bash
qwelli shell
```

#### `serve`

Start web UI server.

```bash
qwelli serve
qwelli serve --port 3000  # Custom port
```

Options:

- `--port, -p` - Port number (default: 8080)

### Interactive Shell

```
qwelli> init                    # Setup config
qwelli> index ./my-folder       # Index and set as current
qwelli> search "query"          # Search current index
qwelli> use ./other-folder      # Switch current index
qwelli> list                    # Show all indexes
qwelli> status                  # Show index stats
qwelli> model                   # Show current model
qwelli> model gpt-4             # Change model (requires re-index)
qwelli> clear                   # Clear screen
qwelli> help                    # Show all commands
qwelli> exit                    # Exit
```

### Supported File Types

Text files only: `.txt`, `.md`, `.go`, `.py`, `.js`, `.ts`, `.java`, `.c`, `.cpp`, `.rs`, `.rb`, `.php`, `.html`, `.css`, `.yaml`, `.yml`, `.toml`, `.sh`, `.proto`, `.graphql`

**Note:** SQL files (`.sql`) are excluded as they can be very large (database dumps) and aren't suitable for semantic search.

Files >500KB and hidden files/folders are skipped.

## Build

```bash
# Current platform
go build -o qwelli ./cmd/qwelli

# Windows
GOOS=windows GOARCH=amd64 go build -o qwelli.exe ./cmd/qwelli

# Linux
GOOS=linux GOARCH=amd64 go build -o qwelli-linux ./cmd/qwelli

# macOS
GOOS=darwin GOARCH=amd64 go build -o qwelli-macos ./cmd/qwelli
GOOS=darwin GOARCH=arm64 go build -o qwelli-macos-arm ./cmd/qwelli
```

For Windows with bundled DuckDB (no GCC required at runtime):

```bash
./build-windows.bat
```

## Testing

### Unit Tests

```bash
# All tests
go test ./...

# Database tests only
go test ./internal/db/... -v

# With coverage
go test ./... -cover
```

### Integration Tests (requires API key)

```bash
# Set .env or export variables
go test ./internal/engine/indexer/... -v
```

### Demo

```bash
# Run end-to-end demo
go run tests/demo/main.go
```

### Manual Testing

```bash
# Build
go build -o qwelli ./cmd/qwelli

# Initialize
./qwelli init

# Index test data
./qwelli index tests/demo/testdata

# Search
./qwelli search "hello" --index tests/demo/testdata
./qwelli search "machine learning" --index tests/demo/testdata

# List and status
./qwelli list
./qwelli status --index tests/demo/testdata
```

## Embedding Providers

Currently supported: **Voyage AI**

### Voyage AI Models

| Model                 | Dimension | Notes                      |
| --------------------- | --------- | -------------------------- |
| `voyage-multimodal-3` | 1024      | Multimodal (text + images) |
| `voyage-3`            | 1024      | Text-only                  |

### Custom Endpoints

Default endpoint:

```
QWELLI_EMBEDDING_ENDPOINT=https://api.voyageai.com/v1/multimodalembeddings
```

## Project Structure

```
qwelli/
├── cmd/qwelli/          # CLI entry point
├── internal/
│   ├── cli/             # Commands (init, index, search, shell)
│   ├── config/          # Config file handling
│   ├── db/              # DuckDB + HNSW index
│   └── engine/          # Index & search orchestration
│       ├── chunker/     # Content chunking strategies
│       ├── indexer/     # Embedding providers
│       └── processor/   # File processing (PDF, images, etc.)
├── tests/demo/          # Demo application
└── dist/                # Built binaries
```

## Data Storage

All data is local:

- **Config:** `~/.qwelli/config.yaml`
- **Indexes:** `~/.qwelli/indexes/*.db`

Each indexed folder gets its own DuckDB database with HNSW vector index.

## How It Works

1. **Index:** Scan folder → Generate embeddings via Voyage AI → Store in DuckDB
2. **Search:** Embed query → HNSW approximate nearest neighbor search → Return matches

One embedding model per database. Change model = re-index.

## Cost

Using Voyage AI `voyage-multimodal-3`:

- ~$0.02 per 1,000 documents indexed
- ~$0.0001 per search query

## Architecture

See [ARCHITECTURE.md](./ARCHITECTURE.md) for detailed architecture documentation.
