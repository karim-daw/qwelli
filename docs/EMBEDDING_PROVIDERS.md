# Embedding Providers

Qwelli supports multiple cloud-based embedding providers through a clean interface-based design. You can easily swap between OpenAI and Cohere without changing your code.

## Quick Start

### Default (OpenAI)

```go
// Uses OpenAI by default
embedder, err := indexer.NewEmbedder("text-embedding-3-small", "", 0)
if err != nil {
    log.Fatal(err)
}
defer embedder.Close()
```

### Using Environment Variables

```bash
# OpenAI (default)
export QWELLI_EMBEDDING_PROVIDER=openai
export OPENAI_API_KEY=sk-...

# Or Cohere
export QWELLI_EMBEDDING_PROVIDER=cohere
export COHERE_API_KEY=...

# Optional: Override model
export QWELLI_EMBEDDING_MODEL=text-embedding-3-large
```

Then use the same code:
```go
embedder, err := indexer.NewEmbedder("", "", 0) // Reads from env vars
```

### Explicit Configuration

```go
import "github.com/karim-daw/qwelli/internal/indexer"

// OpenAI
cfg := indexer.Config{
    Provider: "openai",
    Model:    "text-embedding-3-small",
    APIKey:   "sk-...",
}
embedder, err := indexer.NewEmbedderWithProvider(cfg)

// Cohere
cfg := indexer.Config{
    Provider: "cohere",
    Model:    "embed-english-v3.0",
    APIKey:   "...",
}
embedder, err := indexer.NewEmbedderWithProvider(cfg)
```

## Supported Providers

### 1. OpenAI ⭐ Default

**Pros:**
- ✅ Very fast (~100-200ms per embedding)
- ✅ True parallel batch processing
- ✅ High quality embeddings
- ✅ Large batch support (up to 2048 texts)
- ✅ Multiple model sizes available

**Cons:**
- 💰 Costs money ($0.02 per 1M tokens for small model)
- ⚠️ Requires internet
- ⚠️ Data sent to OpenAI servers

**Configuration:**
```bash
# Environment variables
QWELLI_EMBEDDING_PROVIDER=openai
OPENAI_API_KEY=sk-...
QWELLI_EMBEDDING_MODEL=text-embedding-3-small  # optional
```

**Models:**
| Model | Dimension | Cost per 1M tokens |
|-------|-----------|-------------------|
| `text-embedding-3-small` | 1536 | $0.02 |
| `text-embedding-3-large` | 3072 | $0.13 |
| `text-embedding-ada-002` | 1536 | $0.10 |

### 2. Cohere

**Pros:**
- ✅ Fast (~100-200ms per embedding)
- ✅ True parallel batch processing
- ✅ Multi-lingual support
- ✅ Flexible input types
- ✅ Specialized models for different tasks

**Cons:**
- 💰 Costs money ($0.10 per 1M tokens)
- ⚠️ Requires internet
- ⚠️ Data sent to Cohere servers

**Configuration:**
```bash
# Environment variables
QWELLI_EMBEDDING_PROVIDER=cohere
COHERE_API_KEY=...
QWELLI_EMBEDDING_MODEL=embed-english-v3.0  # optional
```

**Models:**
| Model | Dimension | Languages |
|-------|-----------|-----------|
| `embed-english-v3.0` | 1024 | English |
| `embed-multilingual-v3.0` | 1024 | 100+ languages |
| `embed-english-light-v3.0` | 384 | English (faster) |

## Performance Comparison

Based on 13 documents + 10 queries (23 embeddings total):

| Provider | Time | Cost | Notes |
|----------|------|------|-------|
| **OpenAI** | ~2-5s | $0.001 | Batch processing |
| **Cohere** | ~2-5s | $0.005 | Batch processing |

## Environment Variables Reference

```bash
# Provider selection (default: openai)
QWELLI_EMBEDDING_PROVIDER=openai|cohere

# Model override (optional, provider-specific defaults used if not set)
QWELLI_EMBEDDING_MODEL=model-name

# API Keys (required)
OPENAI_API_KEY=sk-...
COHERE_API_KEY=...

# Custom endpoints (optional)
OPENAI_API_ENDPOINT=https://api.openai.com/v1/embeddings
COHERE_API_ENDPOINT=https://api.cohere.ai/v1/embed
```

## Code Examples

### Production: Use OpenAI

```go
import "github.com/karim-daw/qwelli/internal/indexer"

cfg := indexer.Config{
    Provider: "openai",
    Model:    "text-embedding-3-small",
    APIKey:   os.Getenv("OPENAI_API_KEY"),
}

embedder, err := indexer.NewEmbedderWithProvider(cfg)
if err != nil {
    log.Fatal(err)
}
defer embedder.Close()

// Batch processing - very fast!
texts := []string{"doc1", "doc2", "doc3", ...}
embeddings, err := embedder.EmbedBatch(texts)
// Takes ~200ms for multiple documents!
```

### Using Environment Variables

```go
// Reads from QWELLI_EMBEDDING_PROVIDER env var
embedder, err := indexer.NewEmbedder("", "", 0)
if err != nil {
    log.Fatal(err)
}
defer embedder.Close()

// Works with any provider!
embedding, err := embedder.Embed("Hello, world!")
```

Then just change the environment variable:
```bash
# OpenAI (default)
OPENAI_API_KEY=sk-... go run main.go

# Cohere
QWELLI_EMBEDDING_PROVIDER=cohere COHERE_API_KEY=... go run main.go
```

## Adding New Providers

To add a new provider, implement the `EmbeddingProvider` interface:

```go
type EmbeddingProvider interface {
    Embed(text string) ([]float32, error)
    EmbedBatch(texts []string) ([][]float32, error)
    Dimension() int
    Close() error
}
```

See `openai_provider.go` or `cohere_provider.go` for examples.

## Best Practices

### For Cost-Conscious Projects
- Use **OpenAI** with `text-embedding-3-small` (lowest cost)
- Batch embeddings when possible
- Cache embeddings to avoid re-computing

### For Speed-Critical Projects
- Use **OpenAI** or **Cohere** with batch processing
- Enable concurrent requests when appropriate
- Consider smaller models for faster processing

### For Multi-lingual Projects
- Use **Cohere** `embed-multilingual-v3.0`
- Supports 100+ languages out of the box

### For Prototyping
- Start with **OpenAI** (simple, fast, good quality)
- Use environment variables for easy provider switching
- Same code works for both providers!

## Troubleshooting

### "Provider not found" error
Make sure `QWELLI_EMBEDDING_PROVIDER` is one of: `openai`, `cohere`

### "API key required" error
Set the appropriate environment variable:
- OpenAI: `OPENAI_API_KEY`
- Cohere: `COHERE_API_KEY`

### Slow performance
- Check your internet connection
- Ensure you're using `EmbedBatch()` for multiple texts
- Both providers are typically very fast (100-200ms)

### Rate limiting
- OpenAI: Check your account tier limits
- Cohere: Monitor your API usage
- Both providers have generous rate limits for typical use cases
