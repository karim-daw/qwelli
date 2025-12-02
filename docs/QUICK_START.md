# Quick Start Guide

Get started with Qwelli embeddings in under 5 minutes!

## Prerequisites

**None!** Qwelli automatically installs Ollama if it's not present on your system.

## Basic Usage

```go
package main

import (
    "log"
    "github.com/karim-daw/qwelli/internal/indexer"
)

func main() {
    // Create embedder - this will:
    // 1. Detect your OS
    // 2. Install Ollama if needed
    // 3. Start the server
    // 4. Download the model if needed
    embedder, err := indexer.NewEmbedder("nomic-embed-text", "", 0)
    if err != nil {
        log.Fatal(err)
    }
    defer embedder.Close() // Important: stops the server
    
    // Generate an embedding
    embedding, err := embedder.Embed("Hello, world!")
    if err != nil {
        log.Fatal(err)
    }
    
    log.Printf("Embedding dimension: %d", len(embedding))
    log.Printf("First 5 values: %v", embedding[:5])
}
```

## First Run Output

When you run your application for the first time, you'll see:

```
🖥️  System: WSL (Windows Subsystem for Linux) (linux/arm64)
⚠️  Ollama is not installed
🔧 Attempting automatic installation...
📥 Downloading Ollama installation script...
🚀 Running Ollama installer...
✅ Ollama installed successfully!
✅ Ollama is installed
🚀 Starting Ollama server...
✅ Ollama server started (PID: 12345)
⏳ Waiting for Ollama server to be ready...
✅ Ollama server is ready
🔍 Checking for model 'nomic-embed-text'...
📥 Model 'nomic-embed-text' not found, downloading...
   📊 pulling manifest
   📊 pulling 970aa74c0a90... 100%
✅ Model 'nomic-embed-text' downloaded successfully
🔢 Detecting embedding dimension...
✅ Ollama manager initialized
   Model: nomic-embed-text
   Dimension: 768
   Host: http://localhost:11434
```

## Subsequent Runs

After the first run, startup is much faster:

```
🖥️  System: WSL (Windows Subsystem for Linux) (linux/arm64)
✅ Ollama is installed
🚀 Starting Ollama server...
✅ Ollama server started (PID: 67890)
⏳ Waiting for Ollama server to be ready...
✅ Ollama server is ready
🔍 Checking for model 'nomic-embed-text'...
✅ Model 'nomic-embed-text' is already available
🔢 Detecting embedding dimension...
✅ Ollama manager initialized
   Model: nomic-embed-text
   Dimension: 768
   Host: http://localhost:11434
```

## Common Tasks

### Generate Single Embedding

```go
text := "This is a test document"
embedding, err := embedder.Embed(text)
if err != nil {
    log.Fatal(err)
}
log.Printf("Generated %d-dimensional embedding", len(embedding))
```

### Generate Batch Embeddings

```go
texts := []string{
    "First document",
    "Second document",
    "Third document",
}
embeddings, err := embedder.EmbedBatch(texts)
if err != nil {
    log.Fatal(err)
}
log.Printf("Generated %d embeddings", len(embeddings))
```

### Get Embedding Dimension

```go
dim := embedder.Dimension()
log.Printf("This model produces %d-dimensional embeddings", dim)
```

### Use a Different Model

```go
// Use a larger model for better quality
embedder, err := indexer.NewEmbedder("mxbai-embed-large", "", 0)

// Or a smaller model for speed
embedder, err := indexer.NewEmbedder("all-minilm", "", 0)
```

## Installation Locations

### Ollama Binary

- **Linux/WSL**: `/usr/local/bin/ollama`
- **macOS**: `/opt/homebrew/bin/ollama` (Homebrew) or `/Applications/Ollama.app`
- **Windows**: `C:\Program Files\Ollama\ollama.exe`

### Models

- **Linux/WSL**: `~/.ollama/models`
- **macOS**: `~/.ollama/models`
- **Windows**: `C:\Users\%username%\.ollama\models`

## Available Models

| Model | Dimension | Size | Best For |
|-------|-----------|------|----------|
| `nomic-embed-text` | 768 | ~274 MB | General-purpose (default) |
| `mxbai-embed-large` | 1024 | ~670 MB | Higher quality, slower |
| `all-minilm` | 384 | ~120 MB | Faster, lower quality |

## Configuration

### Custom Model Storage

```bash
# Set before running your app
export OLLAMA_MODELS=/path/to/models
```

### Use External Ollama Server

```go
// Don't manage the server yourself
manager, err := indexer.NewOllamaManager(
    "http://remote-server:11434",
    "nomic-embed-text",
    false, // Don't start/stop server
)
```

## Examples

### Full Working Example

See `examples/demo/main.go` for a complete end-to-end example that:
- Scans a folder for files
- Generates embeddings
- Stores them in DuckDB
- Performs semantic search

Run it:
```bash
cd examples/demo
go run main.go
```

### Embedding Test

See `tests/test_embedding.go` for a simple embedding test:
```bash
go run tests/test_embedding.go
```

## Troubleshooting

### Installation Fails

If automatic installation fails, install manually:

**Linux/WSL:**
```bash
curl -fsSL https://ollama.ai/install.sh | sh
```

**macOS:**
```bash
brew install ollama
```

**Windows:**
Download from [ollama.ai/download](https://ollama.ai/download)

### Server Won't Start

Check if Ollama is already running:
```bash
ps aux | grep ollama
```

Kill existing process if needed:
```bash
pkill ollama
```

### Model Download Stuck

Check your internet connection and try pulling manually:
```bash
ollama pull nomic-embed-text
```

## Next Steps

- Read [EMBEDDINGS.md](./EMBEDDINGS.md) for detailed documentation
- Read [AUTO_INSTALL.md](./AUTO_INSTALL.md) for installation details
- Read [ARCHITECTURE.md](./ARCHITECTURE.md) for system architecture
- Check out the [demo application](../examples/demo/main.go)

## Learn More

- [Ollama Documentation](https://github.com/ollama/ollama)
- [Nomic Embed Text Model](https://ollama.ai/library/nomic-embed-text)
- [DuckDB VSS Extension](https://duckdb.org/docs/extensions/vss)
