---
name: Multimodal Image Support
overview: Add multimodal embedding support to Qwelli by integrating Voyage AI for image embeddings, extracting images from PDFs, and enabling semantic search across both text and images.
todos:
  - id: db-schema
    content: "Update database schema: add content_type, source_chunk_id, image_data columns to chunks table with migration support"
    status: completed
  - id: db-models
    content: Update Chunk and SearchResult models to include new multimodal fields
    status: completed
    dependencies:
      - db-schema
  - id: db-queries
    content: Update InsertChunk and GetChunk queries to handle new fields, add GetChunksByType method
    status: completed
    dependencies:
      - db-models
  - id: image-extractor
    content: Create image_extractor.go to extract images from PDFs using pdfcpu, with compression support
    status: completed
  - id: multimodal-chunker
    content: Create multimodal_chunker.go to orchestrate text + image chunking with proper sequencing
    status: completed
    dependencies:
      - image-extractor
  - id: voyage-provider
    content: Create voyage_provider.go implementing MultimodalEmbeddingProvider interface with Voyage AI API integration
    status: completed
  - id: provider-interface
    content: Extend embedding_provider.go with MultimodalEmbeddingProvider interface and update NewEmbedder for provider selection
    status: completed
    dependencies:
      - voyage-provider
  - id: engine-integration
    content: Update engine.go processPDFFileNew to detect multimodal support and route to MultimodalChunker when enabled
    status: completed
    dependencies:
      - multimodal-chunker
      - provider-interface
  - id: search-filtering
    content: Add content_type filtering to SearchANN in search.go with optional contentType parameter
    status: completed
    dependencies:
      - db-queries
  - id: cli-index
    content: Add --multimodal flag to index command and update progress display
    status: completed
    dependencies:
      - engine-integration
  - id: cli-search
    content: Add --text-only and --images-only flags to search command with image preview display
    status: completed
    dependencies:
      - search-filtering
  - id: cli-init
    content: Update init command to prompt for provider selection (OpenAI vs Voyage) and multimodal enablement
    status: completed
    dependencies:
      - provider-interface
  - id: config-updates
    content: Add EnableMultimodal and ImageQuality fields to config.go
    status: completed
  - id: migration-logic
    content: Implement migration logic in migrations.go to auto-upgrade existing databases to schema v2
    status: completed
    dependencies:
      - db-schema
---

# Qwelli Multimodal Embeddings Implem

entation Plan

## Overview

Add multimodal embedding support to Qwelli's PDF processing pipeline, enabling semantic search across both text chunks and images extracted from PDFs using Voyage AI's multimodal embeddings API.

## Current Architecture Analysis

The codebase uses:

- **Database**: DuckDB with `files`, `chunks`, `embeddings`, and `metadata` tables
- **Chunk Model**: Text chunks stored in `chunks` table (not a unified `documents` table)
- **Embedding Provider**: `HTTPEmbeddingProvider` (OpenAI-compatible, text-only)
- **PDF Processing**: `PDFProcessor` extracts text only; images are not extracted
- **Engine**: `processPDFFileNew` handles PDF text chunking

## Key Design Decisions

### 1. Data Model Changes

- **Extend `chunks` table**: Add `content_type` (TEXT, default 'text'), `source_chunk_id` (TEXT, nullable), and `image_data` (BLOB, nullable) columns
- **Image storage**: Store base64-encoded images in `image_data` column for self-contained database
- **Parent linking**: Images link to parent PDF file via `file_id` (already exists)
- **Metadata**: Store image dimensions, format, page number in existing chunk structure

### 2. Provider Architecture

- **Extend interface**: Create `MultimodalEmbeddingProvider` interface extending `EmbeddingProvider`
- **Voyage provider**: New `VoyageEmbeddingProvider` implementing multimodal interface
- **Factory pattern**: Update `NewEmbedder` to support provider selection
- **Backward compatibility**: Existing `HTTPEmbeddingProvider` remains unchanged for text-only

### 3. Search Strategy

- **Default**: Search both text and images (no filtering)
- **CLI flags**: Add `--text-only` and `--images-only` to search command
- **Results display**: Show content type indicator, image preview path for image results

## Database Schema Changes

### Modified `chunks` table

```sql
ALTER TABLE chunks ADD COLUMN content_type TEXT DEFAULT 'text';
ALTER TABLE chunks ADD COLUMN source_chunk_id TEXT;
ALTER TABLE chunks ADD COLUMN image_data BLOB;
CREATE INDEX idx_chunks_content_type ON chunks(content_type);
CREATE INDEX idx_chunks_source_chunk ON chunks(source_chunk_id);
```



### Chunk Model Updates

Update `internal/db/models.go`:

```go
type Chunk struct {
    // Existing fields...
    ContentType   string  // "text" or "image"
    SourceChunkID *string // Parent chunk ID for images (nullable)
    ImageData     []byte  // Base64-encoded image data (nullable)
}
```



## Implementation Files

### New Files to Create

1. **`internal/processor/image_extractor.go`**

- Extract images from PDFs using `pdfcpu/pkg/api`
- Return `PDFImage` structs with data, format, dimensions, page number
- Clean up temp files automatically
- Handle image compression (max 1024x1024) before storage

2. **`internal/processor/multimodal_chunker.go`**

- Orchestrate text chunking + image wrapping
- Return `MultimodalChunk` array with both text and image chunks
- Sequence chunks by page number, maintain indices
- Integrate with existing `PDFChunker`

3. **`internal/indexer/voyage_provider.go`**

- Implement `MultimodalEmbeddingProvider` interface
- API: `POST https://api.voyageai.com/v1/multimodalembeddings`
- Auth: `Authorization: Bearer <API_KEY>`
- Model: `voyage-multimodal-3` (1024 dims)
- Handle text strings and base64-encoded images
- Smart batching: max 1000 inputs, 320K tokens per request

4. **`internal/db/migrations.go`**

- Auto-detect schema version from metadata table
- `MigrateToV2()`: Add new columns to existing databases
- Set defaults: `content_type='text'` for existing rows
- Transaction-based with rollback on failure

### Files to Modify

1. **`internal/db/schema.go`** (lines 16-27)

- Add new columns to CREATE TABLE statement for `chunks`
- Add new indexes
- Update schema version to 2 in metadata

2. **`internal/db/models.go`** (Chunk struct, line ~15)

- Add `ContentType`, `SourceChunkID`, `ImageData` fields
- Update `SearchResult` to include content type

3. **`internal/db/queries.go`** (InsertChunk, GetChunk methods)

- Handle new fields in INSERT/SELECT statements
- Add `GetChunksByType(contentType string, fileID string)` method

4. **`internal/db/search.go`** (SearchANN method, line ~63)

- Add optional `contentType` parameter for filtering
- Update WHERE clause: `WHERE c.content_type = ?` when filtering

5. **`internal/engine/engine.go`** (processPDFFileNew method, line ~502)

- Detect if provider supports multimodal (type assertion)
- If multimodal enabled: use MultimodalChunker
- Create chunks with appropriate content_type
- Prepare mixed inputs for embedding provider (text strings + image base64)

6. **`internal/indexer/embedding_provider.go`**

- Add `MultimodalEmbeddingProvider` interface
- Add `EmbedImage(imageBase64 string)` method
- Add `EmbedMultimodal(inputs []MultimodalInput)` method

7. **`internal/indexer/embeddings.go`** (NewEmbedder method, line ~20)

- Add provider type parameter
- Switch between HTTP (OpenAI) and Voyage providers
- Return appropriate provider implementation

8. **`internal/config/config.go`** (Config struct, line ~12)

- Add `EnableMultimodal bool` (default: false)
- Add `ImageQuality string` (default: "medium")

9. **`internal/cli/index.go`** (index command handler, line ~33)

- Add `--multimodal` flag
- Pass to engine
- Update progress: show text chunks + images count

10. **`internal/cli/search.go`** (search command handler, line ~36)

    - Add `--text-only` and `--images-only` flags
    - Display content type in results
    - For images: decode base64, save preview to temp file, show path

11. **`internal/cli/init.go`** (init command, line ~22)

    - Prompt for provider: "OpenAI" or "Voyage AI"
    - If Voyage: default multimodal to true
    - Save to config.yaml

## Implementation Sequence

### Phase 1: Database Foundation

**Files**: `db/schema.go`, `db/models.go`, `db/queries.go`, `db/migrations.go`

1. Add new columns to schema with defaults
2. Update Chunk struct with new fields
3. Implement migration logic to auto-upgrade existing databases
4. Test migration on sample database

**Validation**: Existing databases upgrade without data loss

### Phase 2: Image Extraction

**Files**: `processor/image_extractor.go`, `processor/multimodal_chunker.go`

1. Create ImageExtractor using pdfcpu API

- Extract to temp dir, read into memory, clean up
- Parse dimensions using `image.DecodeConfig()`
- Compress images to max 1024x1024

2. Create MultimodalChunker

- Combine text chunks from PDFChunker
- Wrap images as individual chunks
- Sequence by page number

3. Write tests with sample PDFs

**Validation**: Extract images from user's POC PDF, verify count and metadata

### Phase 3: Voyage Provider

**Files**: `indexer/voyage_provider.go`, `indexer/embedding_provider.go`, `indexer/embeddings.go`, `config/config.go`

1. Define `MultimodalEmbeddingProvider` interface
2. Implement VoyageProvider

- Text embedding: similar to HTTPEmbeddingProvider pattern
- Image embedding: base64 encode, send to API
- Multimodal batch: mix text strings and image objects

3. Update NewEmbedder to support provider selection
4. Add config fields for multimodal enablement

**Validation**: Generate embeddings for sample text + images, verify dimensions (1024)

### Phase 4: Engine Integration

**Files**: `engine/engine.go`

1. Detect provider capabilities via type assertion
2. Route PDFs to MultimodalChunker if multimodal enabled
3. Batch text and image chunks separately or together
4. Insert chunks with correct content_type and image_data

**Validation**: Index PDF with images, verify both text and image chunks in DB

### Phase 5: CLI & Search

**Files**: `cli/index.go`, `cli/search.go`, `cli/init.go`, `db/search.go`

1. Add multimodal flag to index command
2. Add content-type filters to search command
3. Update init to configure provider
4. Implement filtered search in database layer
5. Display image previews in search results

**Validation**: End-to-end workflow via CLI

### Phase 6: Testing & Polish

1. Write unit tests for all new files
2. Integration tests for full pipeline
3. Migration tests (v1 → v2)
4. Error handling improvements
5. Documentation updates

## Migration Strategy

### Backward Compatibility

- **Existing indexes**: Auto-migrate schema on open, add columns with defaults
- **OpenAI users**: No changes required, multimodal disabled by default
- **Config**: Default provider = "openai" for existing users

### User Migration Path

1. Run `qwelli init` to switch to Voyage provider
2. Set VOYAGE_API_KEY environment variable (or via config)
3. Re-index folders with `--multimodal` flag to add images

## API Integration Details

### Voyage API Request Format

```json
{
  "model": "voyage-multimodal-3",
  "inputs": [
    "text content here",
    {"image_base64": "base64_encoded_image_data"}
  ],
  "input_type": "document"
}
```



### Response Format

```json
{
  "embeddings": [[0.1, 0.2, ...], [0.3, 0.4, ...]],
  "text_tokens": 15,
  "image_pixels": 1024000,
  "total_tokens": 1847
}
```



### Batching Strategy

- Max 1,000 inputs per request
- Max 320,000 total tokens
- Batch text chunks together (up to 8000 tokens like OpenAI)
- Include images in same batch if under limits
- Split into multiple requests if needed

## Risk Mitigation

1. **Database bloat**: Compress images to max 1024x1024 before storage
2. **Rate limits**: Exponential backoff, client-side rate limiting
3. **Migration failures**: Transaction-based, backup recommendation
4. **Image extraction errors**: Graceful degradation, log warnings, continue with text
5. **Breaking changes**: Feature flags, extensive regression testing

## Success Criteria

- Extract and index both text and images from PDFs
- Generate multimodal embeddings via Voyage AI
- Search across both content types with optional filtering
- Existing OpenAI indexes continue to work
- Zero data loss during migration
- Image extraction < 100ms per image
- Search latency < 200ms

## References

- Voyage AI Docs: https://docs.voyageai.com/docs/multimodal-embeddings
- API Reference: https://docs.voyageai.com/reference/multimodal-embeddings-api