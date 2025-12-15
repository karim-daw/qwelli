# Qwelli Architecture

## Overview

Desktop app for local semantic search. No server required.

```
┌─────────────────────────────────────┐
│     Your Machine                    │
│                                     │
│  ┌──────────────┐                  │
│  │  qwelli CLI  │                  │
│  └──────┬───────┘                  │
│         │                           │
│  ┌──────▼────────┐                 │
│  │  DuckDB + VSS │                 │
│  │  (~/.qwelli/) │                 │
│  └───────────────┘                 │
└──────────┬──────────────────────────┘
           │ HTTPS (embeddings only)
    ┌──────▼──────┐
    │  OpenAI API │
    └─────────────┘
```

## Components

```
cmd/qwelli/          # CLI entry point
internal/
├── cli/             # Cobra commands
├── config/          # YAML config
├── db/              # DuckDB wrapper
│   ├── duckdb.go    # Connection, schema
│   ├── queries.go   # Insert/load
│   └── search.go    # HNSW index, ANN search
├── engine/          # Orchestration
└── indexer/         # OpenAI embeddings
```

## Data Flow

### Indexing

```
Folder → Scan files → OpenAI API → Embeddings → DuckDB → HNSW Index
```

### Search

```
Query → OpenAI API → Embedding → HNSW ANN Search → Results
```

## Database Schema

```sql
-- Documents
CREATE TABLE documents (
    doc_id TEXT PRIMARY KEY,
    path TEXT,
    file_type TEXT,
    modified_at TIMESTAMP,
    size BIGINT,
    metadata JSON,
    content TEXT
);

-- Embeddings (one per document)
CREATE TABLE embeddings (
    doc_id TEXT PRIMARY KEY REFERENCES documents(doc_id),
    vector FLOAT[1536]  -- dimension from model
);

-- HNSW index for fast search
CREATE INDEX hnsw_idx ON embeddings USING HNSW (vector)
WITH (metric = 'cosine');

-- Key-value metadata
CREATE TABLE metadata (
    key TEXT PRIMARY KEY,
    value TEXT
);
```

## Key Design Decisions

1. **One model per database** - Simplifies schema, no model_id needed
2. **HNSW index** - O(log n) approximate nearest neighbor search
3. **Cosine distance** - Standard for text embeddings
4. **Local only** - All data on user's machine
5. **BYOK** - User provides their own OpenAI key
