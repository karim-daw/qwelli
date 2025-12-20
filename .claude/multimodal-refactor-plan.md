# Qwelli Multimodal Embeddings Refactor Plan

## Overview
Refactor Qwelli's PDF processing to support Voyage AI multimodal embeddings for both text chunks and images extracted from PDFs, enabling semantic search across both content types.

## Key Design Decisions

### 1. Data Model
- **Unified document model**: Images stored as separate documents in same `documents` table
- **Content type field**: `content_type` = "text" or "image"
- **Parent linking**: Images link to parent PDF via `source_doc_id`
- **Image storage**: Base64-encoded in database (self-contained, single .db file)

### 2. Provider Architecture
- **New interface**: `MultimodalEmbeddingProvider` extends `EmbeddingProvider`
- **Factory pattern**: Create providers via `provider_factory.go`
- **Backward compatible**: OpenAI provider unchanged (text-only)
- **Voyage provider**: Implements full multimodal interface

### 3. Search Strategy
- **Default**: Search both text and images
- **Filtering**: `--text-only` or `--images-only` CLI flags
- **Results**: Show content type, image previews for image results

## Database Schema Changes

### Modified `documents` table
```sql
ALTER TABLE documents ADD COLUMN content_type TEXT DEFAULT 'text';
ALTER TABLE documents ADD COLUMN source_doc_id TEXT;
ALTER TABLE documents ADD COLUMN image_metadata JSON;
CREATE INDEX idx_content_type ON documents(content_type);
CREATE INDEX idx_source_doc ON documents(source_doc_id);
```

### Document Model Updates
```go
type Document struct {
    // Existing fields...
    ContentType   string  // "text" or "image"
    SourceDocID   string  // Parent document ID for images
    ImageMetadata any     // JSON: format, dimensions, page
}
```

## Critical Files

### New Files to Create

1. **`internal/processor/image_extractor.go`**
   - Extract images from PDFs using `pdfcpu/pkg/api`
   - Return `PDFImage` structs with data, format, dimensions, page number
   - Clean up temp files automatically

2. **`internal/processor/multimodal_chunker.go`**
   - Orchestrate text chunking + image wrapping
   - Return `MultimodalChunk` array with both text and image chunks
   - Sequence chunks by page number, set indices

3. **`internal/indexer/voyage_provider.go`**
   - Implement `MultimodalEmbeddingProvider` interface
   - API: `POST https://api.voyageai.com/v1/multimodalembeddings`
   - Auth: `Authorization: Bearer <API_KEY>`
   - Model: `voyage-multimodal-3` (1024 dims)
   - Handle text strings and base64-encoded images
   - Smart batching: max 1000 inputs, 320K tokens per request

4. **`internal/indexer/provider_factory.go`**
   - Factory method: `NewProvider(providerType, apiKey, model, endpoint)`
   - Switch between "openai" and "voyage"
   - Return appropriate provider implementation

5. **`internal/db/migrations.go`**
   - Auto-detect schema version from metadata table
   - `MigrateToV2()`: Add new columns to existing databases
   - Set defaults: `content_type='text'` for existing rows
   - Transaction-based with rollback on failure

### Files to Modify

1. **`internal/db/schema.go`** (lines 20-40)
   - Add new columns to CREATE TABLE statement
   - Add new indexes
   - Update schema version to 2 in metadata

2. **`internal/db/models.go`** (Document struct, line ~15)
   - Add ContentType, SourceDocID, ImageMetadata fields

3. **`internal/db/queries.go`** (InsertDocument, GetDocument methods)
   - Handle new fields in INSERT/SELECT statements
   - Add `GetDocumentsByType(contentType string)` method

4. **`internal/db/search.go`** (SearchANN method, line ~50)
   - Add optional `contentType` parameter for filtering
   - Update WHERE clause: `WHERE d.content_type = ?` when filtering

5. **`internal/engine/engine.go`** (processPDFFile method, line ~150)
   - Detect if provider supports multimodal (type assertion)
   - If multimodal enabled: use MultimodalChunker
   - Create documents with appropriate content_type
   - Prepare mixed inputs for embedding provider

6. **`internal/indexer/embeddings.go`** (NewEmbedder method, line ~30)
   - Update to use provider factory instead of direct OpenAI creation

7. **`internal/config/config.go`** (Config struct, line ~10)
   - Add `EmbeddingProvider string` (default: "openai")
   - Add `EnableMultimodal bool` (default: false)
   - Add `ImageQuality string` (default: "medium")

8. **`internal/cli/index.go`** (index command handler, line ~40)
   - Add `--multimodal` flag
   - Pass to engine
   - Update progress: show text chunks + images count

9. **`internal/cli/search.go`** (search command handler, line ~50)
   - Add `--text-only` and `--images-only` flags
   - Display content type in results
   - For images: save preview to temp file, show path

10. **`internal/cli/init.go`** (init command, line ~30)
    - Prompt for provider: "OpenAI" or "Voyage AI"
    - If Voyage: default multimodal to true
    - Save to config.yaml

## Implementation Sequence

### Phase 1: Database Foundation (Priority 1)
**Files**: `db/schema.go`, `db/models.go`, `db/queries.go`, `db/migrations.go`

1. Add new columns to schema with defaults
2. Update Document struct with new fields
3. Implement migration logic to auto-upgrade existing databases
4. Test migration on sample database

**Validation**: Existing databases upgrade without data loss

### Phase 2: Image Extraction (Priority 1)
**Files**: `processor/image_extractor.go`, `processor/multimodal_chunker.go`

1. Create ImageExtractor using pdfcpu API
   - Extract to temp dir, read into memory, clean up
   - Parse dimensions using `image.DecodeConfig()`
2. Create MultimodalChunker
   - Combine text chunks from PDFChunker
   - Wrap images as individual chunks
   - Sequence by page number
3. Write tests with sample PDFs

**Validation**: Extract images from user's POC PDF, verify count and metadata

### Phase 3: Voyage Provider (Priority 1)
**Files**: `indexer/voyage_provider.go`, `indexer/provider_factory.go`, `config/config.go`

1. Implement VoyageProvider
   - Text embedding: similar to OpenAI pattern
   - Image embedding: base64 encode, send to API
   - Multimodal batch: mix text strings and image objects
2. Create provider factory
3. Add config fields for provider selection

**Validation**: Generate embeddings for sample text + images, verify dimensions (1024)

### Phase 4: Engine Integration (Priority 2)
**Files**: `engine/engine.go`, `indexer/embeddings.go`

1. Detect provider capabilities via type assertion
2. Route PDFs to MultimodalChunker if multimodal enabled
3. Batch text and image chunks separately or together
4. Insert documents with correct content_type

**Validation**: Index PDF with images, verify both text and image documents in DB

### Phase 5: CLI & Search (Priority 2)
**Files**: `cli/index.go`, `cli/search.go`, `cli/init.go`, `db/search.go`

1. Add multimodal flag to index command
2. Add content-type filters to search command
3. Update init to configure provider
4. Implement filtered search in database layer

**Validation**: End-to-end workflow via CLI

### Phase 6: Testing & Polish (Priority 3)
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
2. Set VOYAGE_API_KEY environment variable
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

- ✅ Extract and index both text and images from PDFs
- ✅ Generate multimodal embeddings via Voyage AI
- ✅ Search across both content types with optional filtering
- ✅ Existing OpenAI indexes continue to work
- ✅ Zero data loss during migration
- ✅ Image extraction < 100ms per image
- ✅ Search latency < 200ms

## References

- Voyage AI Docs: https://docs.voyageai.com/docs/multimodal-embeddings
- API Reference: https://docs.voyageai.com/reference/multimodal-embeddings-api
- Authentication: https://docs.voyageai.com/docs/api-key-and-installation
- Model: voyage-multimodal-3 (1024 dimensions, 32K context)
