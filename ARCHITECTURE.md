# Qwelli Architecture - Desktop App (BYOK Model)

## Executive Summary

**GCP Server: NOT REQUIRED** ✅

Qwelli is a desktop application that runs entirely on the customer's machine. Each customer:
- Installs the CLI binary
- Provides their own OpenAI API key (BYOK)
- Indexes their local folders
- All data stays local

## Architecture Diagram

```
┌─────────────────────────────────────────────────────────────┐
│                    CUSTOMER'S MACHINE                       │
├─────────────────────────────────────────────────────────────┤
│                                                             │
│  ┌──────────────────────────────────────┐                  │
│  │         qwelli CLI Binary            │                  │
│  │  - init (setup)                      │                  │
│  │  - index (scan & embed)              │                  │
│  │  - search (query)                    │                  │
│  │  - list (show indexes)               │                  │
│  │  - status (stats)                    │                  │
│  └──────────────┬───────────────────────┘                  │
│                 │                                           │
│  ┌──────────────▼───────────────────────┐                  │
│  │      Configuration & Storage         │                  │
│  │  ~/.qwelli/                          │                  │
│  │    ├── config.yaml (API key)         │                  │
│  │    └── indexes/                      │                  │
│  │          ├── project1.db             │                  │
│  │          ├── project2.db             │                  │
│  │          └── docs.db                 │                  │
│  └──────────────────────────────────────┘                  │
│                                                             │
└─────────────────┼───────────────────────────────────────────┘
                  │
                  │ HTTPS API Calls
                  │ (Only for embedding generation)
                  │
           ┌──────▼──────────┐
           │   OpenAI API    │
           │  Customer's Key │
           └─────────────────┘
```

## Components

### 1. CLI Binary (`cmd/qwelli/main.go`)
- Single Go binary
- Uses Cobra for CLI framework
- Cross-platform (Linux, macOS, Windows)

### 2. Configuration (`internal/config/`)
- Stores API credentials in `~/.qwelli/config.yaml`
- Customer provides their own OpenAI API key
- Protected with 0600 permissions

### 3. Indexing Engine (`internal/engine/`)
- Reuses your existing demo code
- Scans folders recursively
- Generates embeddings via OpenAI API
- Stores documents + vectors locally

### 4. Database (`internal/db/`)
- DuckDB with VSS extension
- One `.db` file per indexed folder
- Stored in `~/.qwelli/indexes/`
- Contains:
  - Documents (paths, content, metadata)
  - Embeddings (vectors)
  - HNSW index (for fast search)

### 5. Embedder (`internal/indexer/`)
- Your existing OpenAI provider
- Batch processing for efficiency
- Customer's API key → OpenAI → vectors

## Data Flow

### Indexing Flow
```
1. Customer runs: qwelli index ~/my-folder
2. CLI scans all files in ~/my-folder
3. For each file:
   - Read content
   - Send to OpenAI API (customer's key)
   - Receive embedding vector
   - Store in local DuckDB
4. Build HNSW index for fast search
5. Save to ~/.qwelli/indexes/my-folder.db
```

### Search Flow
```
1. Customer runs: qwelli search "query" --index ~/my-folder
2. CLI loads ~/.qwelli/indexes/my-folder.db
3. Generate query embedding via OpenAI API
4. Search HNSW index locally (no network)
5. Return top K results instantly
```

## Why No GCP Server?

### Individual Use Case
Each customer:
- Has their own data
- Indexes their own folders
- Searches their own documents
- **No sharing between customers**

Therefore:
- ❌ No need for centralized storage
- ❌ No need for shared search index
- ❌ No need for multi-tenancy
- ✅ Desktop app is perfect!

### When You WOULD Need a Server

Only if you wanted to offer:
1. **Team/Shared Search** - Multiple users searching same index
2. **Managed Embeddings** - You provide API key (subscription model)
3. **Web Access** - Browser-based instead of desktop app
4. **Centralized Analytics** - Track usage across customers

For individual customers with BYOK → **No server needed!**

## Infrastructure Costs

### Your Costs (BYOK Model)
- **Development**: One-time engineering
- **Distribution**: Hosting binaries (S3/GitHub Releases)
- **Support**: Customer support
- **Server costs**: $0 ✅

### Customer Costs
- OpenAI API usage:
  - ~$0.02 per 1,000 documents indexed
  - ~$0.0001 per search query
- Very affordable for individual use

## Deployment & Distribution

### Build for Multiple Platforms
```bash
# macOS
GOOS=darwin GOARCH=amd64 go build -o qwelli-macos ./cmd/qwelli
GOOS=darwin GOARCH=arm64 go build -o qwelli-macos-arm ./cmd/qwelli

# Windows
GOOS=windows GOARCH=amd64 go build -o qwelli.exe ./cmd/qwelli

# Linux
GOOS=linux GOARCH=amd64 go build -o qwelli-linux ./cmd/qwelli
```

### Distribution Options
1. **GitHub Releases** - Host binaries for download
2. **Homebrew** (macOS) - `brew install qwelli`
3. **Chocolatey** (Windows) - `choco install qwelli`
4. **Snap/AppImage** (Linux)
5. **Direct download** from your website

### Update Mechanism
- Auto-update checker in CLI
- `qwelli update` command
- Download latest from GitHub releases

## Future Enhancements

### Phase 1 (Current)
- ✅ CLI with BYOK
- ✅ Local indexing
- ✅ Semantic search
- ✅ Multi-folder support

### Phase 2 (Optional)
- 🔄 GUI desktop app (Wails/Fyne)
- 🔄 Auto-reindex on file changes
- 🔄 Incremental indexing
- 🔄 Multiple embedding providers

### Phase 3 (If Needed)
- 🔄 Managed embedding service (GCP)
- 🔄 Subscription model
- 🔄 Web interface option
- 🔄 Team features

## Comparison: BYOK vs Managed

| Feature | BYOK (Current) | Managed Service |
|---------|---------------|-----------------|
| Customer provides API key | ✅ Yes | ❌ No |
| Your infrastructure costs | $0 | $$$ GCP costs |
| Customer setup complexity | Medium | Low |
| Your revenue model | One-time purchase | Subscription |
| Privacy | 100% local | Data via your server |
| Maintenance | Low | Medium/High |

## Conclusion

**For individual customers using their own data:**
- Desktop app with BYOK is the perfect architecture
- No GCP server needed
- Simple, private, cost-effective
- Easy to distribute and maintain

**You only need a server if:**
- Customers demand "no API key" experience
- You want recurring subscription revenue
- You need team/sharing features

Start with BYOK, add managed service later if customers request it!
