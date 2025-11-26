# Qwelli Architecture & Design

## Overview

Qwelli is a local file search engine built in Go using gRPC. It provides hybrid search capabilities combining vector and semantic search techniques for efficient local file discovery.

## Why gRPC?

| Feature | gRPC | REST |
|---------|------|------|
| Speed | ⭐⭐⭐⭐⭐ | ⭐⭐⭐ |
| Streaming | Bidirectional | Hard |
| Strong types | Yes | No |
| Binary protocol | Yes | No |
| Local IPC | Excellent | Acceptable |
| Cross-language clients | Auto-generated | Manual |
| Desktop app integration | Trivial (TS/Go/C#) | Verbose |

gRPC is ideal for Qwelli due to its performance, type safety, and ease of integration with desktop applications.

## Project Structure

```
qwelli/
├── cmd/
│   └── qwellid/               # The main server binary
│       └── main.go
│
├── api/
│   ├── proto/                 # .proto definitions
│   │   ├── indexer.proto
│   │   ├── search.proto
│   │   └── common.proto
│   │
│   ├── gen/                   # Generated code (Don't edit!)
│   │   ├── go/
│   │   │   └── qwelli/       # Generated Go gRPC stubs
│   │   └── ts/               # optional TS client for desktop app
│   │
│   └── buf.yaml               # if using Buf (recommended)
│
├── internal/
│   ├── server/                # gRPC server implementations
│   │   ├── grpc_server.go
│   │   ├── indexer_service.go
│   │   ├── search_service.go
│   │   └── health_service.go
│   │
│   ├── indexer/               # core indexing logic
│   │   ├── scanner.go
│   │   ├── extractor.go
│   │   ├── embeddings.go
│   │   └── watcher.go
│   │
│   ├── search/                # core search logic
│   │   ├── bm25.go
│   │   ├── vector.go
│   │   ├── hybrid.go
│   │   └── ranking.go
│   │
│   ├── db/                    # sqlite / data storage
│   │   ├── sqlite.go
│   │   └── migrations/
│   │
│   ├── models/                # internal struct models
│   │   ├── document.go
│   │   └── embedding.go
│   │
│   ├── config/                # config loading + CLI flags
│   │   └── config.go
│   │
│   └── utils/                 # logger, file helpers
│       ├── logger.go
│       └── files.go
│
├── pkg/                       # public SDK for external clients (optional)
│   └── client/
│       ├── client.go          # wraps gRPC client
│       └── models.go
│
├── tools/
│   ├── protoc/
│   │   └── generate.go        # go:generate script
│
├── go.mod
└── README.md
```

## Design Principles

### 1. Separation of Concerns

- **api/proto/**: Interface contracts only (search, indexer, common types)
- **api/gen/**: All generated stubs (Go, TypeScript)
- **internal/server/**: Thin gRPC adapters that delegate to core logic
- **internal/indexer/ & internal/search/**: Core business logic (pure Go, testable)

### 2. gRPC as Transport Layer

The gRPC layer is just an adapter. The core search engine is pure Go, making it:
- Replaceable
- Testable
- Independent of transport mechanism

### 3. Scalability

This structure works for both MVP and production:
- **MVP**: Basic FTS5, single background worker, minimal proto files
- **Production**: Multiple indexes, desktop client, update hooks, ONNX embeddings, hot-reloading watchers

## Component Details

### api/proto/

Protocol buffer definitions separated by domain:
- `search.proto`: Search API definitions
- `indexer.proto`: Indexing API definitions
- `common.proto`: Shared types and messages

### internal/server/

gRPC service implementations that:
- Implement generated service interfaces
- Inject dependencies (search engine, indexer, database)
- Handle request/response conversion
- Manage context and errors

Example structure:
```go
type SearchService struct {
    qwellipb.UnimplementedSearchServiceServer
    searchEngine *search.Engine
}
```

### internal/indexer/

Core indexing functionality:
- **scanner.go**: Filesystem crawling
- **extractor.go**: Text extraction from various file types
- **embeddings.go**: Embedding generation using local models
- **watcher.go**: File system watching for real-time updates

### internal/search/

Core search algorithms:
- **bm25.go**: BM25 text search implementation
- **vector.go**: Vector similarity search
- **hybrid.go**: Hybrid scoring combining BM25 and vector search
- **ranking.go**: Result ranking and re-ranking pipeline

### internal/db/

Database layer:
- SQLite with FTS5 for full-text search
- Optional vector extension for vector storage
- Migration system for schema management

### pkg/client/

Optional public SDK providing:
- Convenience wrappers around gRPC client
- Simplified API for external consumers
- Used by desktop apps, CLI tools, plugins

## TODO

### Phase 1: Foundation ✅ (Current)

**Services & Infrastructure**
- [x] Basic gRPC server setup
- [x] SearchService with basic file search (filename and content matching)
- [x] Protobuf definitions and code generation
- [x] gRPC reflection enabled

### Phase 2: Database & Indexing

**Database & Storage**
- [ ] SQLite connection setup
- [ ] Database schema design (documents, embeddings, metadata)
- [ ] Migration system
- [ ] FTS5 integration for text search
- [ ] Vector storage solution (SQLite extension or separate index)

**Indexing System**
- [ ] File system scanner (recursive directory traversal)
- [ ] Text extractor (support multiple file types)
- [ ] Indexing pipeline (scan → extract → embed → store)

**Services & Infrastructure**
- [ ] Indexer service (gRPC API for indexing operations)

### Phase 3: Search Implementation

**Search Algorithms**
- [ ] BM25 implementation for text search
- [ ] SQLite FTS5 text search integration
- [ ] Vector similarity search (cosine similarity)
- [ ] Hybrid search algorithm (combining BM25 and vector scores)
- [ ] Result ranking and re-ranking
- [ ] Search result pagination

**Embeddings & Models**
- [ ] Local embedding model integration (ONNX or similar)
- [ ] Embedding generation pipeline
- [ ] Model loading and caching
- [ ] Batch embedding processing
- [ ] Embedding storage and retrieval

### Phase 4: Advanced Features

**Indexing System**
- [ ] File watcher for real-time updates

**Services & Infrastructure**
- [ ] Health service (gRPC)
- [ ] Configuration management (CLI flags, config file)
- [ ] Structured logging system
- [ ] Error handling and recovery
- [ ] Graceful shutdown

### Phase 5: Polish & Production

**Testing & Quality**
- [ ] Unit tests for core search logic
- [ ] Integration tests for gRPC services
- [ ] Database migration tests
- [ ] Performance benchmarks
- [ ] End-to-end tests

**Documentation & SDK**
- [ ] API documentation
- [ ] Public SDK (pkg/client)
- [ ] Usage examples
- [ ] Desktop client integration guide

**Services & Infrastructure**
- [ ] Performance optimization

## Technology Stack

- **Language**: Go 1.21+
- **API**: gRPC with Protocol Buffers
- **Database**: SQLite with FTS5
- **Vector Search**: SQLite vector extension or custom implementation
- **Embeddings**: Local ONNX model (or similar)
- **Code Generation**: protoc or Buf

## Future Considerations

- Desktop application integration (Avalonia, Electron, etc.)
- VSCode plugin support
- Multiple search indexes
- Image embeddings
- Hot-reloading configuration
- Distributed indexing for large file systems

