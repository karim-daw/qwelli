# Voyage Reranker Integration

This document describes the Voyage AI reranker integration in Qwelli, which improves search result relevance by reranking initial search results using Voyage's specialized reranking models.

## Overview

The reranker acts as a post-processing step after the initial search (semantic, keyword, or hybrid) is performed. It takes the query and the initial search results, and uses Voyage AI's reranking API to reorder them based on true relevance to the query.

## Benefits

- **Improved Relevance**: Reranking models are specifically trained to assess query-document relevance, often outperforming embedding similarity alone
- **Better User Experience**: Users get the most relevant results at the top, reducing time spent searching
- **Seamless Integration**: Works with all existing search strategies (semantic, keyword, hybrid)
- **Detailed Logging**: Comprehensive logging shows reranking performance and improvements

## Configuration

### Enable Reranker

Add the following to your `~/.qwelli/config.yaml`:

```yaml
# Reranker settings
enable_reranker: true
rerank_provider: voyage
rerank_model: rerank-2
```

### Environment Variables

You can also configure via environment variables:

```bash
export QWELLI_RERANK_MODEL=rerank-2
export QWELLI_RERANK_ENDPOINT=https://api.voyageai.com/v1/rerank  # Optional
```

### API Key

The reranker uses the same API key as the embedding provider:

```bash
export QWELLI_EMBEDDING_KEY=your_voyage_api_key
```

## Architecture

### Components

1. **Reranker Provider Interface** (`internal/engine/reranker/provider.go`)
   - Defines the `Provider` interface for reranking implementations
   - Includes `SearchResult` type to avoid circular dependencies

2. **Voyage Provider** (`internal/engine/reranker/voyage_provider.go`)
   - Implements Voyage AI's reranking API
   - Features:
     - Automatic retry with exponential backoff (max 3 retries)
     - Timeout handling (30 seconds default)
     - Detailed logging with emoji indicators
     - Performance metrics (tokens used, time elapsed)

3. **Server Integration** (`internal/server/server.go`)
   - Initializes reranker on server startup
   - Applies reranking between search and response
   - Falls back gracefully on errors

### Search Flow with Reranking

```
User Query
  ↓
Search Strategy (semantic/keyword/hybrid)
  ↓
Initial Results (e.g., top 20)
  ↓
Reranker (if enabled)
  ↓
Reranked Results (sorted by relevance score)
  ↓
Response to User
```

## Logging Examples

### Successful Reranking

```
🔍 Cache miss, performing search for: machine learning best practices
🔄 Reranking 10 results for query: machine learning best practices
  ✓ Reranking completed in 245ms (1,234 tokens used)
  📊 Top result relevance: 0.892 (was: 0.745)
```

### Initialization

```
🚀 Server starting on http://localhost:8080
📂 Index directory: /home/user/.qwelli/indexes
🤖 Model: voyage-multimodal-3
🔄 Reranker: enabled (voyage)
```

### Error Handling

```
⚠️  Reranking failed: request timeout (using original results)
```

## API Integration

The reranker is transparent to the API - no changes are needed to client code. Search requests work exactly as before:

```
GET /api/search?q=neural+networks&index=/path/to/docs&strategy=semantic&top=10
```

The reranking happens automatically server-side if enabled.

## Performance Considerations

### Latency

- Reranking adds ~200-500ms of latency per search
- This is acceptable for most use cases given the relevance improvement
- Results are cached, so subsequent identical queries are instant

### Token Usage

- Each rerank request consumes tokens based on:
  - Query length
  - Number of documents
  - Document lengths
- Typical usage: 1,000-5,000 tokens per rerank request

### Best Practices

1. **Cache Results**: Server caches search results for 5 minutes by default
2. **Limit Initial Results**: Don't send more than 50-100 results to reranker
3. **Monitor Logs**: Watch for timeout or error patterns
4. **Adjust topK**: Use appropriate `top` parameter in search requests

## Implementation Details

### Relevance Score Handling

The reranker returns relevance scores (higher is better), which are converted to distance metrics (lower is better) to maintain consistency with the embedding-based search:

```go
// Store negative relevance score as distance
result.Distance = -item.RelevanceScore

// Sort by distance ascending (which is relevance descending)
sort.Slice(rerankedResults, func(i, j int) bool {
    return rerankedResults[i].Distance < rerankedResults[j].Distance
})
```

### Error Handling

The implementation gracefully handles errors:

- **API Errors**: Logged and original results returned
- **Timeouts**: Retried up to 3 times with exponential backoff
- **Invalid Responses**: Logged and skipped
- **Network Issues**: Falls back to original results

### Request Format

The Voyage rerank API expects:

```json
{
  "query": "user search query",
  "documents": ["doc1 content", "doc2 content", ...],
  "model": "rerank-2"
}
```

### Response Format

```json
{
  "object": "list",
  "data": [
    {
      "index": 0,
      "relevance_score": 0.892
    },
    {
      "index": 2,
      "relevance_score": 0.745
    }
  ],
  "model": "rerank-2",
  "usage": {
    "total_tokens": 1234
  }
}
```

## Troubleshooting

### Reranker Not Working

1. Check that `enable_reranker: true` in config
2. Verify API key is set: `echo $QWELLI_EMBEDDING_KEY`
3. Check server logs for initialization messages
4. Look for error messages in search logs

### Timeouts

If you see frequent timeouts:

1. Reduce the number of results being reranked
2. Check network connectivity to Voyage API
3. Consider increasing timeout (modify `defaultTimeout` in code)

### Poor Results

If reranked results seem worse:

1. Verify you're using the correct model (`rerank-2`)
2. Check that initial search results are relevant
3. Ensure query and documents are in English (or supported language)
4. Try different search strategies (semantic vs. hybrid)

## Future Enhancements

Potential improvements for future versions:

- [ ] Configurable timeout per request
- [ ] Support for other reranking providers (Cohere, OpenAI)
- [ ] Selective reranking based on initial result quality
- [ ] A/B testing framework to measure improvement
- [ ] Reranking cache separate from search cache
- [ ] Batch reranking for multiple queries

## References

- [Voyage AI Rerank API Documentation](https://docs.voyageai.com/docs/reranker)
- [Qwelli Configuration Guide](./config.example.yaml)
- [Search Strategies Documentation](./internal/engine/search/README.md)
