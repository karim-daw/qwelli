# Qwelli Web UI Quick Start

This guide will help you build and run the Qwelli web interface.

## Prerequisites

- Go 1.21+ (with CGO support)
- Node.js 18+ and npm
- Voyage AI API key

## Step 1: Configure Environment

Create a `.env` file in the project root:

```bash
QWELLI_EMBEDDING_KEY=your-voyage-api-key
QWELLI_EMBEDDING_MODEL=voyage-multimodal-3
QWELLI_EMBEDDING_ENDPOINT=https://api.voyageai.com/v1/multimodalembeddings
```

## Step 2: Initialize Qwelli

```bash
# Build the CLI first (without UI)
go build -o qwelli ./cmd/qwelli

# Initialize configuration
./qwelli init

# Index a folder
./qwelli index ./path/to/your/folder
```

## Step 3: Build the Web UI

### Linux/Mac

```bash
./build-with-ui.sh
```

### Windows

```cmd
build-with-ui.bat
```

The build script will:
1. Install frontend dependencies (npm install)
2. Build the React app
3. Copy assets to the Go embed location
4. Build the Go binary with embedded assets

## Step 4: Start the Web Server

```bash
# Default port (8080)
./qwelli serve

# Custom port
./qwelli serve --port 3000
```

## Step 5: Open in Browser

Navigate to: **http://localhost:8080**

You should see the Qwelli search interface with:
- Dropdown to select an indexed folder
- Search box with natural language queries
- Options for search strategy (semantic/keyword/hybrid)
- Content type filtering (text/images/all)
- Results display with similarity scores

## Development Mode

To develop the frontend with hot reload:

```bash
# Terminal 1: Start backend server (Go must be built first)
go build -o qwelli ./cmd/qwelli
./qwelli serve

# Terminal 2: Start frontend dev server
cd web
npm install
npm run dev
```

The Vite dev server will proxy API requests to `http://localhost:8080`.

## Features

### Search Strategies

- **Semantic**: Vector similarity search using embeddings
- **Keyword**: Full-text search with relevance scoring
- **Hybrid**: Combines both semantic and keyword search

### Content Filtering

- **All**: Search both text and image content
- **Text Only**: Only search text chunks
- **Images Only**: Only search image embeddings (if multimodal enabled)

### Results Display

Each result shows:
- Similarity score (higher % = better match)
- File path
- Content preview
- Page number (for PDFs)
- Content type (text/image)

## Troubleshooting

### "No indexes available"

You need to index a folder first using the CLI:

```bash
./qwelli index ./path/to/folder
```

### "Index not found" error

The folder path might not match. Use the exact path you indexed with the CLI, or select from the dropdown.

### Build fails with "npm not found"

Install Node.js from https://nodejs.org/ (version 18 or higher recommended).

### Frontend build fails

Delete `web/node_modules` and `web/package-lock.json`, then rebuild:

```bash
cd web
rm -rf node_modules package-lock.json
cd ..
./build-with-ui.sh
```

## API Endpoints

The server exposes these REST endpoints:

### List Indexes
```
GET /api/indexes
```

Returns all indexed folders with document counts.

### Search
```
GET /api/search?q=<query>&index=<path>&strategy=<semantic|keyword|hybrid>&top=<N>&content_type=<text|image>
```

Performs search on the specified index.

## Architecture

```
┌─────────────┐
│   Browser   │
└──────┬──────┘
       │ HTTP
       ▼
┌─────────────────┐
│  Go HTTP Server │
│  (port 8080)    │
├─────────────────┤
│  Static Files   │◄── Embedded React build (embed.FS)
│  /api/indexes   │
│  /api/search    │
└────────┬────────┘
         │
         ▼
┌─────────────────┐
│  Engine Layer   │
│  (search logic) │
└────────┬────────┘
         │
         ▼
┌─────────────────┐
│   DuckDB + VSS  │
│   (local .db)   │
└─────────────────┘
```

The entire React app is embedded in the Go binary, making it a single standalone executable that serves both the UI and the API.
