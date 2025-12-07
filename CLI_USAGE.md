# Qwelli CLI - Desktop App Usage Guide

Qwelli is a desktop application that provides semantic search for your local files using vector embeddings.

## Architecture

**No GCP Server Required!** Qwelli runs entirely on the customer's machine:

```
┌─────────────────────────────────────┐
│     Customer's Desktop              │
│                                     │
│  ┌──────────────┐                  │
│  │  qwelli CLI  │                  │
│  └──────┬───────┘                  │
│         │                           │
│  ┌──────▼────────┐                 │
│  │  Local DuckDB │                 │
│  │  (~/.qwelli/) │                 │
│  └───────────────┘                 │
└──────────┬──────────────────────────┘
           │
           │ HTTPS (embeddings only)
           │
    ┌──────▼──────┐
    │  OpenAI API │
    └─────────────┘
```

## Installation

```bash
# Build from source
cd /home/karim-daw/dev/qwelli
go build -o qwelli ./cmd/qwelli

# Move to PATH (optional)
sudo mv qwelli /usr/local/bin/
```

## Quick Start

### 1. Initialize Configuration

```bash
qwelli init
```

This will prompt you for:
- OpenAI API key (required)
- Embedding model (default: text-embedding-3-small)
- API endpoint (default: https://api.openai.com/v1/embeddings)

Configuration is saved to `~/.qwelli/config.yaml`

### 2. Index a Folder

```bash
qwelli index ~/Documents/my-project
```

This will:
- Scan all files in the folder recursively
- Generate embeddings for each file
- Store everything in `~/.qwelli/indexes/my-project.db`
- Build an HNSW index for fast similarity search

### 3. Search

```bash
qwelli search "machine learning algorithms" --index ~/Documents/my-project
```

Returns the top 5 most semantically similar files.

Options:
- `--index` or `-i`: Path to the indexed folder (required)
- `--top` or `-t`: Number of results to return (default: 5)

### 4. List All Indexes

```bash
qwelli list
```

Shows all folders you've indexed.

### 5. Check Index Status

```bash
qwelli status --index ~/Documents/my-project
```

Shows statistics about a specific index:
- Number of indexed documents
- Database size
- Last indexed time

## Example Workflow

```bash
# First time setup
qwelli init

# Index your documentation folder
qwelli index ~/Documents/company-docs

# Index your code repository
qwelli index ~/code/my-app

# List what you've indexed
qwelli list

# Search for something
qwelli search "API authentication" --index ~/Documents/company-docs

# Search code
qwelli search "database connection pooling" --index ~/code/my-app --top 10
```

## File Storage

All data is stored locally:
- **Config**: `~/.qwelli/config.yaml`
- **Indexes**: `~/.qwelli/indexes/*.db`

Each indexed folder gets its own DuckDB database file.

## Privacy & Security

- All your documents and embeddings are stored **locally** on your machine
- Only the **text content** is sent to OpenAI for embedding generation
- No data is stored on any server
- You control your own API key and usage

## Customer Distribution Options

### Option 1: BYOK (Current Implementation)
- Customer provides their own OpenAI API key
- No server infrastructure needed
- Customer controls costs

### Option 2: Managed Service (Future)
- You provide a small GCP service to proxy embedding requests
- Customer pays you a subscription
- You handle OpenAI costs + margin
- Better UX (no API key needed)

## Next Steps for Production

1. **Binary Distribution**: Build for multiple platforms (Linux, macOS, Windows)
   ```bash
   GOOS=darwin GOARCH=amd64 go build -o qwelli-macos ./cmd/qwelli
   GOOS=windows GOARCH=amd64 go build -o qwelli.exe ./cmd/qwelli
   GOOS=linux GOARCH=amd64 go build -o qwelli-linux ./cmd/qwelli
   ```

2. **Installer**: Create installers for each platform
   - macOS: `.dmg` or homebrew
   - Windows: `.msi` installer
   - Linux: `.deb`/`.rpm` or snap

3. **Auto-updates**: Add update checking mechanism

4. **GUI (Optional)**: Build a desktop GUI using:
   - [Wails](https://wails.io/) - Go + Web UI
   - [Fyne](https://fyne.io/) - Pure Go GUI

5. **Additional Features**:
   - Re-index modified files only
   - Watch folders for changes
   - Export search results
   - Advanced filtering (by file type, date, etc.)

## Cost Estimation for Customers

Using OpenAI `text-embedding-3-small`:
- $0.02 per 1M tokens
- ~1,000 tokens per page of text
- **~$0.02 to index 1,000 documents**

Very affordable for individual use!
