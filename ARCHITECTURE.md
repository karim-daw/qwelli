# Qwelli Architecture

## Overview

Desktop app for local semantic search. No server required.

```
┌─────────────────────────────────────────────────────────┐
│                    Your Machine                          │
│                                                          │
│  ┌──────────────┐                                       │
│  │ qwelli CLI   │                                       │
│  └──────┬───────┘                                       │
│         │                                               │
│  ┌──────▼──────────┐                                    │
│  │ Engine Layer    │                                    │
│  └───┬────────┬────┘                                    │
│      │        │                                         │
│  ┌───▼───┐ ┌─▼────────┐  ┌──────────────┐            │
│  │ File  │ │ Embedding│  │ DuckDB + HNSW │            │
│  │ Proc. │ │ Provider │  │ ~/.qwelli/    │            │
│  └───────┘ └────┬─────┘  └───────────────┘            │
│                 │                                      │
└─────────────────┼──────────────────────────────────────┘
                  │ HTTPS (embeddings only)
          ┌───────▼───────┐
          │  OpenAI API   │
          └───────────────┘
```

## Components

### Component Architecture

```
┌─────────────────────────────────────────────────────────────┐
│ CLI Layer                                                    │
│  ┌──────────────┐                                           │
│  │ cmd/qwelli   │  Main Entry Point                         │
│  └──────┬───────┘                                           │
│         │                                                    │
│  ┌──────▼──────────┐                                        │
│  │ internal/cli/   │  Cobra Commands                        │
│  └──────┬──────────┘                                        │
└─────────┼────────────────────────────────────────────────────┘
          │
┌─────────▼────────────────────────────────────────────────────┐
│ Business Logic Layer                                         │
│  ┌──────────────────┐  ┌──────────────┐                     │
│  │ internal/engine │  │ internal/    │                     │
│  │ Orchestration    │  │ config/       │                     │
│  └──────┬───────────┘  │ YAML Config │                     │
│         │                └──────────────┘                    │
└─────────┼────────────────────────────────────────────────────┘
          │
    ┌─────┴─────┬──────────────┬──────────────┐
    │           │              │               │
┌───▼───┐ ┌─────▼────┐ ┌───────▼────┐ ┌───────▼────┐
│ Data  │ │ Process  │ │  Indexer   │ │   Config   │
│ Layer │ │  Layer   │ │   Layer    │ │   Layer    │
└───┬───┘ └─────┬────┘ └───────┬────┘ └────────────┘
    │           │              │
┌───▼───┐ ┌─────▼────┐ ┌───────▼────┐
│ DuckDB│ │ PDF/Text │ │  OpenAI    │
│ +HNSW │ │ Chunker  │ │    API    │
└───────┘ └──────────┘ └────────────┘
```

### Component Details

```
cmd/qwelli/          # CLI entry point
internal/
├── cli/             # Cobra commands
│   ├── init.go      # Configuration setup
│   ├── index.go     # Index folder command
│   ├── search.go    # Search command
│   ├── list.go      # List indexes
│   ├── status.go    # Index statistics
│   └── shell.go     # Interactive shell
├── config/          # YAML config management
│   └── config.go     # Load/save config
├── db/              # DuckDB wrapper
│   ├── duckdb.go    # Connection, schema creation
│   ├── models.go    # Data models (File, Chunk, Embedding)
│   ├── queries.go   # CRUD operations
│   └── search.go    # HNSW index, ANN search
├── engine/          # Orchestration layer
│   └── engine.go    # IndexFolder, Search, GetIndexStatus
├── processor/       # File processing
│   ├── pdf_processor.go    # PDF text extraction
│   ├── pdf_chunker.go      # PDF chunking
│   ├── chunker.go          # Text chunking
│   └── hash.go             # SHA256 computation
└── indexer/         # Embedding providers
    ├── embedding_provider.go  # Interface
    └── embeddings.go          # OpenAI implementation
```

## Data Flow

### Indexing Flow

```
User
 │
 │ index <folder>
 ▼
CLI
 │
 │ IndexFolder(folderPath)
 ▼
Engine
 │
 │ Scan folder for files
 ▼
┌─────────────────────────────────────┐
│ For each file:                      │
│                                     │
│  Engine ──► Processor               │
│              │                       │
│              ├─► Process (PDF/Text) │
│              ├─► Chunk content     │
│              │                       │
│              └─► Return chunks      │
│                                     │
│  Engine ──► DB                     │
│              ├─► Insert File        │
│              └─► Insert Chunks      │
└─────────────────────────────────────┘
 │
 │ EmbedBatch(all chunks)
 ▼
Indexer
 │
 │ POST /embeddings
 ▼
OpenAI API
 │
 │ Return vectors
 ▼
Indexer ──► Engine
              │
              │ For each chunk:
              │   Insert Embedding ──► DB
              │
              │ Build/Rebuild HNSW Index ──► DB
              │
              └─► Success ──► CLI ──► User
```

### Search Flow

```
User
 │
 │ search "query"
 ▼
CLI
 │
 │ Search(query, topK)
 ▼
Engine
 │
 │ Embed(query)
 ▼
Indexer
 │
 │ POST /embeddings
 ▼
OpenAI API
 │
 │ Return vector
 ▼
Indexer ──► Engine
              │
              │ SearchANN(queryVec, topK)
              ▼
            DB
              │
              │ HNSW ANN Search
              │
              └─► Return SearchResults
              │
              ▼
            Engine
              │
              │ Format results
              │
              └─► Return SearchResults ──► CLI ──► User
```

### File Processing Pipeline

```
File from Filesystem
        │
        ▼
   ┌─────────┐
   │File Type│
   └────┬────┘
        │
   ┌────┴────┐
   │         │
  PDF      Text
   │         │
   ▼         ▼
Extract   Read File
Text      Content
   │         │
   ▼         ▼
Has Text? Size > 500KB?
   │         │
   │      Yes│No
   │      │  │
   │      │  ▼
   │      │Estimate
   │      │Tokens
   │      │  │
   │      │  ▼
   │      │> 1000?
   │      │  │
   │      │  │
   │      │  ├─Yes─► Chunk Text (300 tokens, 10 overlap)
   │      │  │
   │      │  └─No──► Single Chunk (entire file)
   │      │
   │      └─► Skip Large File
   │
   ├─No─► Skip Image-only PDF
   │
   └─Yes─► Chunk PDF Pages (300 tokens, 10 overlap)
           │
           │
           ▼
   ┌───────────────────┐
   │ Create Chunk      │
   │ Records           │
   │ (with page_nums)  │
   └─────────┬─────────┘
             │
             ▼
   ┌───────────────────┐
   │ Store File Record │
   │ (with SHA256 hash)│
   └─────────┬─────────┘
             │
             ▼
   ┌───────────────────┐
   │ Store Chunk       │
   │ Records           │
   │ (denormalized)    │
   └─────────┬─────────┘
             │
             ▼
   ┌───────────────────┐
   │ Batch Embed       │
   │ All Chunks        │
   │ (via OpenAI API)  │
   └─────────┬─────────┘
             │
             ▼
   ┌───────────────────┐
   │ Store Embeddings │
   └─────────┬─────────┘
             │
             ▼
   ┌───────────────────┐
   │ Build/Rebuild     │
   │ HNSW Index        │
   └─────────┬─────────┘
             │
             ▼
      Indexing Complete
```

## Database Schema

### Entity Relationship Diagram

```
┌─────────────────────────────────────────────────────────────┐
│                         FILES                                 │
│  ┌───────────────────────────────────────────────────────┐ │
│  │ file_id (PK)                                           │ │
│  │ path (UNIQUE)                                          │ │
│  │ file_type                                              │ │
│  │ file_hash (SHA256)                                     │ │
│  │ modified_at                                            │ │
│  │ size                                                    │ │
│  │ indexed_at                                              │ │
│  └───────────────────────────────────────────────────────┘ │
└───────────────────────┬─────────────────────────────────────┘
                        │
                        │ 1:N (has)
                        │
┌───────────────────────▼─────────────────────────────────────┐
│                         CHUNKS                               │
│  ┌───────────────────────────────────────────────────────┐ │
│  │ chunk_id (PK)                                          │ │
│  │ file_id (FK) ────────────┐                            │ │
│  │ file_path (denormalized) │                            │ │
│  │ file_type (denormalized) │                            │ │
│  │ chunk_index              │                            │ │
│  │ total_chunks             │                            │ │
│  │ content                  │                            │ │
│  │ start_token              │                            │ │
│  │ end_token                │                            │ │
│  │ page_numbers[]           │                            │ │
│  └──────────────────────────┼────────────────────────────┘ │
└─────────────────────────────┼──────────────────────────────┘
                               │
                               │ 1:1 (has)
                               │
┌──────────────────────────────▼──────────────────────────────┐
│                      EMBEDDINGS                              │
│  ┌───────────────────────────────────────────────────────┐ │
│  │ chunk_id (PK) ────────────┐                           │ │
│  │ vector FLOAT[dimension]   │                           │ │
│  └───────────────────────────┼───────────────────────────┘ │
└──────────────────────────────┼──────────────────────────────┘
                                │
                                │ HNSW Index on vector
                                │ (cosine distance)
                                ▼
                         ┌──────────────┐
                         │  HNSW Index  │
                         │  (for search)│
                         └──────────────┘

┌─────────────────────────────────────────────────────────────┐
│                        METADATA                              │
│  ┌───────────────────────────────────────────────────────┐ │
│  │ key (PK)                                               │ │
│  │ value                                                  │ │
│  └───────────────────────────────────────────────────────┘ │
└─────────────────────────────────────────────────────────────┘
```

### Schema Details

```sql
-- Files table: One record per indexed file
CREATE TABLE files (
    file_id TEXT PRIMARY KEY,
    path TEXT NOT NULL UNIQUE,
    file_type TEXT NOT NULL,
    file_hash TEXT NOT NULL,        -- SHA256 for change detection
    modified_at TIMESTAMP NOT NULL,
    size BIGINT NOT NULL,
    indexed_at TIMESTAMP NOT NULL
);

-- Chunks table: Multiple chunks per file (for large files)
CREATE TABLE chunks (
    chunk_id TEXT PRIMARY KEY,
    file_id TEXT NOT NULL,           -- FK to files
    file_path TEXT NOT NULL,         -- Denormalized for fast search results
    file_type TEXT NOT NULL,         -- Denormalized for fast search results
    chunk_index INT NOT NULL,        -- Order within file
    total_chunks INT NOT NULL,       -- Total chunks in file
    content TEXT NOT NULL,
    start_token INT,                 -- Optional: token position
    end_token INT,                   -- Optional: token position
    page_numbers INT[]               -- For PDFs: which pages this chunk spans
);

CREATE INDEX idx_chunks_file_id ON chunks(file_id);

-- Embeddings table: One embedding per chunk
CREATE TABLE embeddings (
    chunk_id TEXT PRIMARY KEY,       -- FK to chunks
    vector FLOAT[dimension]          -- Dimension from embedding model
);

-- HNSW index for fast approximate nearest neighbor search
CREATE INDEX hnsw_idx ON embeddings USING HNSW (vector)
WITH (metric = 'cosine');

-- Metadata table: Key-value store for project settings
CREATE TABLE metadata (
    key TEXT PRIMARY KEY,
    value TEXT
);
```

### Design Rationale

1. **Files → Chunks → Embeddings**: Three-tier structure allows:

   - Large files to be split into multiple searchable chunks
   - Efficient updates (delete file cascades to chunks/embeddings)
   - Denormalized fields in chunks for fast search results (no JOIN needed)

2. **Denormalization**: `file_path` and `file_type` are duplicated in chunks table to avoid JOINs during search, improving query performance.

3. **HNSW Index**: Hierarchical Navigable Small World graph provides O(log n) approximate nearest neighbor search with high recall.

4. **Change Detection**: File hash (SHA256) enables efficient change detection without re-reading entire file content.

## Key Design Decisions

1. **One model per database** - Simplifies schema, no model_id needed. Changing models requires re-indexing.

2. **Three-tier data model (Files → Chunks → Embeddings)**:

   - Enables chunking of large files for better search granularity
   - Allows efficient updates (delete file cascades to chunks/embeddings)
   - Denormalized fields in chunks avoid JOINs during search

3. **HNSW index** - Hierarchical Navigable Small World graph provides O(log n) approximate nearest neighbor search with high recall.

4. **Cosine distance** - Standard metric for text embeddings, measures semantic similarity.

5. **Local only** - All data stored on user's machine in `~/.qwelli/`. No cloud storage, no data leaves the machine except API calls for embeddings.

6. **BYOK (Bring Your Own Key)** - User provides their own OpenAI API key. No keys stored in the application.

7. **Incremental indexing** - Change detection using file hash (SHA256) and modification time enables efficient updates without full re-indexing.

8. **Batch embedding** - Multiple chunks embedded in a single API call for efficiency and cost savings.

9. **Chunking strategy**:
   - PDFs: Chunked by pages with 300-token chunks and 10-token overlap
   - Text files: Chunked if >1000 tokens, otherwise stored as single chunk
   - Large files (>500KB) are skipped to prevent memory issues
