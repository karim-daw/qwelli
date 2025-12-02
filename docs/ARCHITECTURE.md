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
│   │   ├── embeddings.go      # Embedder wrapper
│   │   ├── embedder.go        # Provider interface and factory
│   │   ├── 
_provider.go # OpenAI embeddings
│   │   ├── cohere_provider.go # Cohere embeddings
│   │   └── watcher.go
│   │
│   ├── search/                # core search logic
│   │   ├── bm25.go
│   │   ├── vector.go
│   │   ├── hybrid.go
│   │   └── ranking.go
│   │
│   ├── db/                    # duckdb / data storage
│   │   ├── duckdb.go          # open/close DB, schema init
│   │   ├── schema.go          # CREATE TABLE statements
│   │   ├── models.go          # Document + Embedding structs
│   │   ├── embeddings.go      # insert/load embeddings
│   │   ├── search.go          # HNSW index creation + ANN search
│   │   └── util.go            # small helper functions
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
├── examples/
│   └── demo/                  # End-to-end demo application
│       └── main.go            # Full demo: scan folder, index files, search
│
├── tests/
│   ├── test_folder/           # Test files for demo
│   └── db_test/               # Database tests
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
- **MVP**: Basic full-text search, single background worker, minimal proto files
- **Production**: Multiple indexes, desktop client, update hooks, ONNX embeddings, hot-reloading watchers, HNSW vector indexes

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
- **embeddings.go**: Simple wrapper around embedding provider
- **embedder.go**: Provider interface definition and factory
- **openai_provider.go**: OpenAI embeddings API client
- **cohere_provider.go**: Cohere embeddings API client
- **watcher.go**: File system watching for real-time updates

### internal/search/

Core search algorithms:
- **bm25.go**: BM25 text search implementation
- **vector.go**: Vector similarity search
- **hybrid.go**: Hybrid scoring combining BM25 and vector search
- **ranking.go**: Result ranking and re-ranking pipeline

### internal/db/

Database layer:
- **duckdb.go**: Database connection management, initialization, VSS extension loading, and lifecycle
- **schema.go**: Table definitions and schema creation (documents, embeddings with FLOAT arrays)
- **models.go**: Go struct definitions for Document and Embedding types
- **embeddings.go**: Embedding insertion, batch loading, and retrieval operations with proper type handling
- **search.go**: HNSW index creation using DuckDB VSS extension and approximate nearest neighbor (ANN) search
- **util.go**: Helper functions for float32/float64 conversions

DuckDB provides:
- Native vector data types (FLOAT arrays) and operations
- Built-in HNSW index support via VSS extension for efficient ANN search
- Full-text search capabilities
- High-performance analytical queries
- Experimental persistence for HNSW indexes on file-based databases

**Current Implementation Status:**
- ✅ DuckDB connection with VSS extension auto-loading
- ✅ Schema with documents and embeddings tables
- ✅ Document and embedding insertion
- ✅ HNSW index creation with cosine similarity
- ✅ ANN search with distance computation
- ✅ Embedding loading with proper type conversion

### pkg/client/

Optional public SDK providing:
- Convenience wrappers around gRPC client
- Simplified API for external consumers
- Used by desktop apps, CLI tools, plugins

### examples/demo/

End-to-end demonstration application that showcases the complete indexing and search pipeline:
- **main.go**: Full working demo that:
  - Scans a test folder recursively
  - Reads file contents and extracts metadata
  - Generates content-aware embeddings (simulated)
  - Indexes documents and embeddings into DuckDB
  - Builds HNSW index for fast similarity search
  - Performs semantic searches and displays results

This demo serves as both a working example and an integration test for the database layer.

## Current Status

### ✅ Completed (Phase 2 - Database Layer)

The database layer is fully functional and tested:

**Database & Storage**
- ✅ DuckDB connection setup with VSS extension loading
- ✅ Database schema design and creation (documents, embeddings tables)
- ✅ Document and Embedding model definitions
- ✅ Vector storage with DuckDB native FLOAT array types
- ✅ HNSW index creation for embeddings using VSS extension
- ✅ Embedding insertion and batch loading with proper type handling
- ✅ Database utility functions (float conversions)

**Search Implementation**
- ✅ HNSW-based approximate nearest neighbor (ANN) search
- ✅ Vector similarity search using DuckDB vector operations with cosine distance
- ✅ Distance computation and result ranking

**Testing & Examples**
- ✅ End-to-end demo application (`examples/demo/main.go`)
- ✅ Working integration with real file scanning and indexing

### Key Implementation Details

**DuckDB VSS Extension:**
- VSS extension must be installed and loaded before creating HNSW indexes
- HNSW indexes require experimental persistence flag for file-based databases
- Index syntax: `CREATE INDEX ... USING vss(vector) WITH (metric='cosine', ...)`
- Vector operations use the `<->` operator for distance computation

**Type Handling:**
- DuckDB expects `float32` arrays for FLOAT array columns
- Arrays are returned as `[]interface{}` and require type conversion
- Proper handling of JSON metadata (must be serialized to string)

**Index Configuration:**
- HNSW index parameters: `hnsw_m=16`, `hnsw_efc=200` (defaults)
- Cosine similarity metric for semantic search
- Index supports incremental updates but may need compaction after deletes

## TODO

### Phase 1: Foundation ✅ (Current)

**Services & Infrastructure**
- [x] Basic gRPC server setup
- [x] SearchService with basic file search (filename and content matching)
- [x] Protobuf definitions and code generation
- [x] gRPC reflection enabled

### Phase 2: Database & Indexing ✅ (COMPLETED)

**Database & Storage**
- [x] DuckDB connection setup (duckdb.go) - with VSS extension auto-loading
- [x] Database schema design and creation (schema.go) - documents and embeddings tables
- [x] Document and Embedding model definitions (models.go)
- [ ] Full-text search integration using DuckDB
- [x] Vector storage with DuckDB native FLOAT array types
- [x] HNSW index creation for embeddings (search.go) - using VSS extension
- [x] Embedding insertion and batch loading (embeddings.go) - with proper type handling
- [x] Database utility functions (util.go) - float32/float64 conversions

**Indexing System**
- [ ] File system scanner (recursive directory traversal)
- [ ] Text extractor (support multiple file types)
- [ ] Indexing pipeline (scan → extract → embed → store)

**Services & Infrastructure**
- [ ] Indexer service (gRPC API for indexing operations)

### Phase 3: Search Implementation

**Search Algorithms**
- [ ] BM25 implementation for text search
- [ ] DuckDB full-text search integration
- [x] HNSW-based approximate nearest neighbor (ANN) search (search.go)
- [x] Vector similarity search using DuckDB vector operations 
- [ ] Hybrid search algorithm (combining BM25 and vector scores)
- [ ] Result ranking and re-ranking - basic distance-based ranking 
- [ ] Search result pagination

**Embeddings & Models**
- [x] Cloud embedding API integration (OpenAI, Cohere)
- [x] Embedding generation pipeline (via API providers)
- [x] Provider interface for swappable backends
- [x] Batch embedding processing with true parallelization
- [x] Embedding storage and retrieval

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
- [x] End-to-end tests - ✅ Demo application serves as E2E test

**Documentation & SDK**
- [ ] API documentation
- [ ] Public SDK (pkg/client)
- [x] Usage examples - ✅ End-to-end demo in examples/demo/
- [ ] Desktop client integration guide

**Services & Infrastructure**
- [ ] Performance optimization

## Technology Stack

- **Language**: Go 1.25+
- **API**: gRPC with Protocol Buffers
- **Database**: DuckDB with native vector support
- **Vector Search**: DuckDB HNSW indexes for approximate nearest neighbor search
- **Embeddings**: OpenAI and Cohere APIs (text-embedding-3-small default)
- **Code Generation**: protoc or Buf

## Future Considerations

- Desktop application integration (Avalonia, Electron, etc.)
- VSCode plugin support
- Multiple search indexes
- Image embeddings
- Hot-reloading configuration
- Distributed indexing for large file systems

