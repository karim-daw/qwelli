# Quick Start Guide

Get started with Qwelli embeddings in under 5 minutes!

## Prerequisites

- Go 1.25 or later
- API key from OpenAI or Cohere

## Setup

### 1. Get an API Key

**OpenAI (Recommended):**
1. Go to [platform.openai.com](https://platform.openai.com)
2. Create an account or sign in
3. Navigate to API Keys section
4. Create a new API key

**Cohere:**
1. Go to [cohere.ai](https://cohere.ai)
2. Create an account or sign in
3. Navigate to API Keys
4. Copy your API key

### 2. Configure Environment Variables

**Option A: Using .env file (Recommended)**

1. Copy the example file:
   ```bash
   cp .env.example .env
   ```

2. Edit `.env` and add your API key:
   ```bash
   QWELLI_EMBEDDING_PROVIDER=openai
   OPENAI_API_KEY=sk-your-api-key-here
   ```

**Option B: Export directly in shell**

```bash
export OPENAI_API_KEY=sk-...
# or
export COHERE_API_KEY=...
```

### 3. Install Dependencies

```bash
cd your-project
go get github.com/karim-daw/qwelli
```

## Basic Usage

```go
package main

import (
    "log"
    "github.com/joho/godotenv"
    "github.com/karim-daw/qwelli/internal/indexer"
)

func main() {
    // Load .env file (if it exists)
    _ = godotenv.Load()
    
    // Create embedder (uses OpenAI by default, reads from env vars)
    embedder, err := indexer.NewEmbedder("text-embedding-3-small", "", 0)
    if err != nil {
        log.Fatal(err)
    }
    defer embedder.Close()
    
    // Generate an embedding
    embedding, err := embedder.Embed("Hello, world!")
    if err != nil {
        log.Fatal(err)
    }
    
    log.Printf("Embedding dimension: %d", len(embedding))
    log.Printf("First 5 values: %v", embedding[:5])
}
```

## Using .env File

The simplest way to run your application is with a `.env` file:

**1. Create `.env`:**
```bash
QWELLI_EMBEDDING_PROVIDER=openai
OPENAI_API_KEY=sk-your-key-here
```

**2. Run your application:**
```bash
go run main.go
```

The `.env` file is automatically loaded by `godotenv.Load()` in your main function.

**Switching Providers:**

Just edit your `.env` file:
```bash
# Switch to Cohere
QWELLI_EMBEDDING_PROVIDER=cohere
COHERE_API_KEY=your-cohere-key
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

### Switch Providers

```go
// OpenAI
cfg := indexer.Config{
    Provider: "openai",
    Model:    "text-embedding-3-small",
    APIKey:   os.Getenv("OPENAI_API_KEY"),
}

// Cohere
cfg := indexer.Config{
    Provider: "cohere",
    Model:    "embed-english-v3.0",
    APIKey:   os.Getenv("COHERE_API_KEY"),
}

embedder, err := indexer.NewEmbedderWithProvider(cfg)
```

## Available Models

### OpenAI

| Model | Dimension | Cost per 1M tokens | Best For |
|-------|-----------|-------------------|----------|
| `text-embedding-3-small` | 1536 | $0.02 | General-purpose (recommended) |
| `text-embedding-3-large` | 3072 | $0.13 | Higher quality |
| `text-embedding-ada-002` | 1536 | $0.10 | Legacy model |

### Cohere

| Model | Dimension | Languages | Best For |
|-------|-----------|-----------|----------|
| `embed-english-v3.0` | 1024 | English | General-purpose |
| `embed-multilingual-v3.0` | 1024 | 100+ | Multi-lingual |
| `embed-english-light-v3.0` | 384 | English | Faster processing |

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
cp ../../.env.example .env
# Edit .env and add your OPENAI_API_KEY
go run main.go
```

## Configuration

### Environment Variables

```bash
# Provider (default: openai)
export QWELLI_EMBEDDING_PROVIDER=openai

# API Keys (required)
export OPENAI_API_KEY=sk-...
export COHERE_API_KEY=...

# Model override (optional)
export QWELLI_EMBEDDING_MODEL=text-embedding-3-large
```

### Programmatic Configuration

```go
cfg := indexer.Config{
    Provider:  "openai",
    Model:     "text-embedding-3-small",
    APIKey:    os.Getenv("OPENAI_API_KEY"),
    Endpoint:  "", // optional custom endpoint
    Dimension: 0,  // auto-detected
}

embedder, err := indexer.NewEmbedderWithProvider(cfg)
```

## Troubleshooting

### "API key required" error

Make sure you've set the appropriate environment variable:
```bash
export OPENAI_API_KEY=sk-your-key-here
# or
export COHERE_API_KEY=your-key-here
```

### "Provider not found" error

Make sure `QWELLI_EMBEDDING_PROVIDER` is either `openai` or `cohere`.

### Rate limiting

Both OpenAI and Cohere have rate limits. If you hit them:
- Use batch processing (`EmbedBatch()`) to reduce API calls
- Add exponential backoff retry logic
- Check your account tier limits

### Slow performance

- Check your internet connection
- Use `EmbedBatch()` for multiple texts (much faster than individual calls)
- Both providers typically respond in 100-200ms

## Next Steps

- Read [EMBEDDING_PROVIDERS.md](./EMBEDDING_PROVIDERS.md) for detailed provider comparison
- Read [ARCHITECTURE.md](./ARCHITECTURE.md) for system architecture
- Check out the [demo application](../examples/demo/main.go)

## Learn More

- [OpenAI Embeddings API](https://platform.openai.com/docs/guides/embeddings)
- [Cohere Embed API](https://docs.cohere.com/reference/embed)
- [DuckDB VSS Extension](https://duckdb.org/docs/extensions/vss)
