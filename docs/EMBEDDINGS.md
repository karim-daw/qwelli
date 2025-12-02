# Embeddings with Ollama

Qwelli uses [Ollama](https://ollama.ai) for fast, lightweight, and local embedding generation. This document explains how the embedding system works and how to use it.

## Overview

The embedding system is designed to be:
- **Zero-Config**: Automatically installs Ollama if not present
- **Super Fast**: Uses Ollama's optimized inference engine
- **Simple**: Minimal API with automatic server management
- **Lightweight**: No heavy dependencies or ONNX runtimes needed
- **Cross-Platform**: Works on Linux, macOS, Windows, and WSL
- **Extensible**: Easy to swap models or add custom configurations

## Architecture

### Components

1. **OllamaManager** (`internal/indexer/ollama.go`)
   - Manages Ollama server lifecycle
   - Handles model downloading and initialization
   - Provides embedding generation APIs
   - Automatically detects embedding dimensions

2. **Embedder** (`internal/indexer/embeddings.go`)
   - Thin wrapper around OllamaManager
   - Maintains backward compatibility with existing code
   - Provides simple `Embed()` and `EmbedBatch()` methods

### How It Works

1. **OS Detection**: Detects your operating system (Linux, macOS, Windows, or WSL)
2. **Auto-Install**: If Ollama is not installed, automatically installs it for your platform
3. **Server Start**: Starts an Ollama server process automatically
4. **Model Check**: Checks if the specified model is available locally
5. **Auto-Download**: Downloads the model if it's not found
6. **Dimension Detection**: Generates a test embedding to determine the vector dimension
7. **Ready**: The embedder is ready to generate embeddings

## Usage

### Basic Usage

```go
import "github.com/karim-daw/qwelli/internal/indexer"

// Create an embedder with default model (nomic-embed-text)
embedder, err := indexer.NewEmbedder("nomic-embed-text", "", 0)
if err != nil {
    log.Fatal(err)
}
defer embedder.Close() // Important: stops the Ollama server

// Get the embedding dimension
dim := embedder.Dimension()
fmt.Printf("Embedding dimension: %d\n", dim)

// Generate a single embedding
embedding, err := embedder.Embed("Hello, world!")
if err != nil {
    log.Fatal(err)
}

// Generate batch embeddings
texts := []string{"First text", "Second text", "Third text"}
embeddings, err := embedder.EmbedBatch(texts)
if err != nil {
    log.Fatal(err)
}
```

### Advanced Usage

```go
// Use a different model
embedder, err := indexer.NewEmbedder("mxbai-embed-large", "", 0)

// Or use the OllamaManager directly for more control
manager, err := indexer.NewOllamaManager(
    "http://localhost:11434", // Ollama host
    "nomic-embed-text",       // Model name
    true,                     // Managed server (start/stop automatically)
)
if err != nil {
    log.Fatal(err)
}
defer manager.Close()
```

## Supported Models

The default model is `nomic-embed-text`, but you can use any Ollama embedding model:

| Model | Dimension | Description |
|-------|-----------|-------------|
| `nomic-embed-text` | 768 | Fast and accurate general-purpose embeddings |
| `mxbai-embed-large` | 1024 | Larger, more accurate embeddings |
| `all-minilm` | 384 | Smaller, faster embeddings |

To use a different model, just pass the model name to `NewEmbedder()`:

```go
embedder, err := indexer.NewEmbedder("mxbai-embed-large", "", 0)
```

## Configuration

### Server Settings

Default configuration:
- **Host**: `http://localhost:11434`
- **Model**: `nomic-embed-text`
- **Managed Server**: `true` (starts/stops automatically)
- **Server Start Timeout**: 30 seconds
- **Health Check Interval**: 500ms

These can be customized by using `NewOllamaManager()` directly:

```go
manager, err := indexer.NewOllamaManager(
    "http://custom-host:11434",
    "custom-model",
    false, // Use existing Ollama server (don't manage it)
)
```

### Environment Variables

Ollama respects standard environment variables:
- `OLLAMA_HOST`: Override the default host
- `OLLAMA_MODELS`: Custom models directory (see Storage Locations below)
- `OLLAMA_NUM_PARALLEL`: Number of parallel requests

See [Ollama documentation](https://github.com/ollama/ollama/blob/main/docs/faq.md) for more details.

### Storage Locations

**Where is Ollama installed?**

- **Linux/WSL**: `/usr/local/bin/ollama` or `/usr/bin/ollama`
- **macOS**: `/opt/homebrew/bin/ollama` (Homebrew) or `/Applications/Ollama.app`
- **Windows**: `C:\Program Files\Ollama\ollama.exe`

**Where are models stored?**

- **Linux/WSL**: `~/.ollama/models` (user) or `/usr/share/ollama/.ollama/models` (system service)
- **macOS**: `~/.ollama/models`
- **Windows**: `C:\Users\%username%\.ollama\models`

Example model for `nomic-embed-text` (default):
```
~/.ollama/models/
├── blobs/
│   ├── sha256-970aa74c0a90...  # Model weights (~274 MB)
│   └── sha256-c71d239df917...  # Model config
└── manifests/
    └── registry.ollama.ai/
        └── library/
            └── nomic-embed-text/
                └── latest
```

To change the storage location, set `OLLAMA_MODELS`:
```bash
export OLLAMA_MODELS=/path/to/your/models
```

## API Reference

### Embedder

```go
type Embedder struct {
    manager *OllamaManager
}

// Create a new embedder
func NewEmbedder(modelName string, modelDir string, expectedDim int) (*Embedder, error)

// Generate embedding for a single text
func (e *Embedder) Embed(text string) ([]float32, error)

// Generate embeddings for multiple texts
func (e *Embedder) EmbedBatch(texts []string) ([][]float32, error)

// Get embedding dimension
func (e *Embedder) Dimension() int

// Clean up and stop server
func (e *Embedder) Close() error
```

### OllamaManager

```go
type OllamaManager struct {
    client    *api.Client
    cmd       *exec.Cmd
    host      string
    model     string
    dimension int
    managed   bool
}

// Create a new Ollama manager
func NewOllamaManager(host, model string, managedServer bool) (*OllamaManager, error)

// Generate embedding for a single text
func (om *OllamaManager) Embed(text string) ([]float32, error)

// Generate embeddings for multiple texts
func (om *OllamaManager) EmbedBatch(texts []string) ([][]float32, error)

// Get embedding dimension
func (om *OllamaManager) Dimension() int

// Clean up and stop server if managed
func (om *OllamaManager) Close() error
```

## Installation

### Automatic Installation

Qwelli automatically detects your operating system and installs Ollama if it's not found:

- **Linux/WSL**: Downloads and runs the official install script
- **macOS**: Uses Homebrew if available, otherwise provides manual instructions
- **Windows**: Downloads and runs the installer (requires user interaction)

When you first run your application, you'll see output like:

```
🖥️  System: WSL (Windows Subsystem for Linux) (linux/arm64)
⚠️  Ollama is not installed
🔧 Attempting automatic installation...
📥 Downloading Ollama installation script...
🚀 Running Ollama installer...
✅ Ollama installation complete!
```

### Manual Installation (Optional)

If you prefer to install manually or if automatic installation fails:

**macOS:**
```bash
brew install ollama
```

**Linux:**
```bash
curl -fsSL https://ollama.ai/install.sh | sh
```

**Windows:**
Download from [ollama.ai](https://ollama.ai/download)

### Verify Installation

```bash
ollama --version
```

## Troubleshooting

### "ollama binary not found in PATH"

Make sure Ollama is installed and in your system PATH:
```bash
which ollama
```

### "timeout waiting for Ollama server to start"

This can happen if:
1. Another Ollama instance is already running
2. Port 11434 is blocked or in use
3. Ollama binary has insufficient permissions

Try running Ollama manually first:
```bash
ollama serve
```

### "failed to pull model"

Check your internet connection and try pulling the model manually:
```bash
ollama pull nomic-embed-text
```

### Model not downloading

If auto-download fails, you can manually pull the model:
```bash
ollama pull nomic-embed-text
```

Then restart your application.

## Performance Tips

1. **Batch Processing**: Use `EmbedBatch()` for multiple texts to improve throughput
2. **Reuse Embedder**: Create one embedder and reuse it across your application
3. **Model Selection**: Choose smaller models for speed, larger for accuracy
4. **External Server**: For production, run Ollama as a separate service and set `managedServer=false`

## Migration from ONNX

If you're upgrading from the old ONNX-based implementation:

1. The API is **backward compatible** - no code changes needed
2. The `modelDir` and `expectedDim` parameters are now ignored (kept for compatibility)
3. Embedding dimension is auto-detected from the model
4. Old ONNX model files can be safely deleted

Example old code:
```go
embedder, err := indexer.NewEmbedder(
    "KnightsAnalytics/all-MiniLM-L6-v2",
    "~/.qwelli/models",
    384,
)
```

Example new code (same API!):
```go
embedder, err := indexer.NewEmbedder(
    "nomic-embed-text",
    "", // ignored
    0,  // ignored
)
```

## Examples

See working examples in:
- `examples/demo/main.go` - Full end-to-end demo
- `tests/test_embedding.go` - Standalone embedding test
