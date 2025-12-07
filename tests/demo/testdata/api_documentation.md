# Qwelli Search API Documentation

## Overview
The Qwelli API provides semantic search capabilities over indexed documents.

## Endpoints

### POST /search
Performs a semantic search query.

**Request Body:**
```json
{
  "query": "machine learning algorithms",
  "top_k": 10,
  "threshold": 0.7
}
```

**Response:**
```json
{
  "results": [
    {
      "doc_id": "abc123",
      "path": "/documents/tech.md",
      "score": 0.95,
      "snippet": "Machine learning involves..."
    }
  ],
  "total": 1
}
```

### GET /health
Health check endpoint.

**Response:**
```json
{
  "status": "healthy",
  "indexed_documents": 1250
}
```

## Authentication
All endpoints require an API key in the `X-API-Key` header.

## Rate Limits
- 100 requests per minute per API key
- 1000 requests per hour per API key

