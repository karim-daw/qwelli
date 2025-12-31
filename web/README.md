# Qwelli Web UI

This is the React-based web interface for Qwelli.

## Development

```bash
# Install dependencies
npm install

# Run development server (proxies API to localhost:8080)
npm run dev

# Build for production
npm run build
```

## Building into Binary

The web UI is embedded into the Go binary during the build process. To build the complete application with the web UI:

```bash
# Linux/Mac
./build-with-ui.sh

# Windows
build-with-ui.bat
```

The build script will:
1. Install frontend dependencies (if needed)
2. Build the React app into `web/dist`
3. Copy the dist folder to `internal/server/web/dist` for embedding
4. Build the Go binary with embedded assets

## Architecture

- **Frontend**: React + TypeScript + Vite
- **API**: REST endpoints served by Go backend
- **Embedding**: Go's `embed` package bundles the built assets into the binary

## API Endpoints

- `GET /api/indexes` - List all available indexes
- `GET /api/search?q=<query>&index=<path>&strategy=<semantic|keyword|hybrid>&top=<N>&content_type=<text|image>` - Search an index
