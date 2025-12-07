---
name: Go Unit Test Suite
overview: Implement a modern Go test suite with unit tests for the db and indexer packages, plus an integration test for the E2E flow, using table-driven tests, temp directories, and HTTP mocking.
todos:
  - id: db-tests
    content: Create internal/db/db_test.go with unit tests for OpenProjectDB, InsertDocument, GetDocument, InsertEmbedding, LoadAllEmbeddings
    status: pending
  - id: search-tests
    content: Add tests for BuildHNSWIndex and SearchANN to verify vector search functionality
    status: pending
  - id: indexer-tests
    content: Create internal/indexer/indexer_test.go with mock HTTP server tests for OpenAIProvider
    status: pending
  - id: integration-test
    content: Create internal/db/integration_test.go for E2E flow test with mock embeddings
    status: pending
  - id: run-verify
    content: Run all tests with go test ./... -v and iterate to fix any failures
    status: pending
---

# Go Unit Test Implementation Plan

## Components to Test

Based on `examples/demo/main.go`, the system has two core packages:

1. **`internal/db/`** - DuckDB operations with vector search (HNSW)
2. **`internal/indexer/`** - Embedding generation via OpenAI API

## Enhancement: Vector Dimension Handling

Currently, `schema.go` uses `FLOAT[]` (variable-length). Different embedding models produce different dimensions (e.g., OpenAI text-embedding-3-small = 1536, ada-002 = 1536, text-embedding-3-large = 3072).

**Solution:** Add a `config` table and dimension validation:

1. **Add `config` table** in `schema.go`:
```go
CREATE TABLE IF NOT EXISTS config (
    key TEXT PRIMARY KEY,
    value TEXT
);
```

2. **Add dimension methods** to `ProjectDB`:

   - `SetEmbeddingDimension(dim int)` - stores dimension on first use
   - `GetEmbeddingDimension() (int, error)` - retrieves stored dimension
   - `ValidateEmbeddingDimension(vec []float32) error` - validates before insert

3. **Add `Dimension()` method** to `Embedder`:

   - Returns the dimension by making a test embed call (cached after first call)
   - Or infer from first `Embed()` result

4. **Update `InsertEmbedding`** to validate dimension matches stored config

This ensures all vectors in a database have consistent dimensions.

## Test Files to Create

### 1. `internal/db/db_test.go` - Database Unit Tests

Test the `ProjectDB` struct and its methods using temporary databases (`t.TempDir()`).

**Test cases:**

- `TestOpenProjectDB` - successful opening, empty path error
- `TestInsertAndGetDocument` - insert document, retrieve by ID
- `TestInsertEmbedding` - insert and load embeddings
- `TestBuildHNSWIndexAndSearchANN` - build index, perform vector search

Key pattern from schema:

```go
// Documents table: doc_id, path, file_type, modified_at, size, metadata, content
// Embeddings table: doc_id, vector (FLOAT[])
```

### 2. `internal/indexer/indexer_test.go` - Embedder Tests (Real API)

Uses real OpenAI API calls. Requires env vars: `QWELLI_EMBEDDING_KEY`, `QWELLI_EMBEDDING_MODEL`, `OPENAI_ENDPOINT`.

**Test cases:**

- `TestNewOpenAIProvider` - validation (missing key/model/endpoint)
- `TestEmbed` - single text embedding via real API
- `TestEmbedBatch` - batch embeddings via real API
- `TestEmbedderWrapper` - test the `Embedder` wrapper struct

Tests will skip if env vars are not set using `t.Skip()`.

### 3. `integration_test.go` (root or tests/) - Full E2E Integration Test

Real end-to-end test mirroring `examples/demo/main.go`:

1. Load env vars (godotenv)
2. Initialize real embedder
3. Create temp database
4. Index sample test files from `tests/test_folder/`
5. Generate real embeddings
6. Build HNSW index
7. Perform similarity searches and verify relevant results are returned

## Testing Conventions

- Use Go's standard `testing` package (no external dependencies)
- Table-driven tests with `t.Run()` for subtests
- `t.TempDir()` for isolated database files
- `t.Skip()` for tests requiring API keys when env vars not set
- Clear test names following `Test<Function>_<Scenario>` pattern

## Required Environment Variables

```bash
QWELLI_EMBEDDING_KEY=<your-api-key>
QWELLI_EMBEDDING_MODEL=<model-name>
OPENAI_ENDPOINT=<api-endpoint>
```