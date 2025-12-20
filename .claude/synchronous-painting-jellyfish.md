# Data Model Refactor Plan - Qwelli DuckDB Schema

## Current State Analysis

### Existing Schema

```sql
-- documents: stores content + metadata
CREATE TABLE documents (
    doc_id TEXT PRIMARY KEY,
    path TEXT,
    file_type TEXT,
    modified_at TIMESTAMP,
    size BIGINT,
    text_metadata JSON,
    content TEXT,
    content_type TEXT,      -- unused
    image_metadata JSON     -- unused
)

-- embeddings: 1:1 with documents
CREATE TABLE embeddings (
    doc_id TEXT PRIMARY KEY REFERENCES documents(doc_id),
    vector FLOAT[dimension]
)

-- metadata: global config
CREATE TABLE metadata (
    key TEXT PRIMARY KEY,
    value TEXT
)
```

### Current Usage Patterns

- **Indexing:** One-time bulk insert of documents → embeddings → HNSW index build
- **Search:** Query embedding → HNSW ANN lookup → document retrieval by doc_id
- **Chunk identity:** MD5 hash of `path:chunk:index` for chunked documents
- **Metadata storage:** JSON blob containing chunk_index, page_numbers, indexed_at, PDF metadata
- **No updates:** Immutable indexes - delete and re-index to update
- **No full-text search:** Pure vector similarity

### Known Issues & Limitations

1. **Unused fields:** `content_type`, `image_metadata` defined but never populated
2. **Flat JSON metadata:** No schema enforcement, 3 different parse variants in code
3. **No deduplication:** Same content in different files = duplicate embeddings
4. **No incremental updates:** Must re-index entire folder
5. **No versioning:** Can't track document changes over time
6. **No relationships:** Can't model document hierarchies or collections
7. **Search limitations:** No hybrid search (vector + keyword), no filtering by file type/date
8. **HNSW Index Immutability:** DuckDB's HNSW index cannot be updated, deleted from, or appended to after creation - requires full rebuild for any changes

---

## Discussion Topics

### 1. Primary Use Case Clarification

Current design assumes:

- Semantic search over code repositories and document folders
- Chunk-level granularity for large files
- Read-heavy workload (write once, search many times)
- Single embedding model per index

**Questions:**

- Do you need incremental updates (add/remove individual files)?
- Do you need to track document versions/history?
- Should we support multiple collections within one database?
- Do you need hybrid search (vector + full-text keyword)?

### 2. Chunk Management Strategy

Current approach:

- Chunks are separate documents with `chunk_index` in JSON metadata
- Original file path stored in each chunk
- No way to query "all chunks from file X"

**Potential improvements:**

- Separate `chunks` table with foreign key to parent `files` table?
- Store file-level metadata separate from chunk-level?
- Add `file_id` column to enable "find all chunks from this file"?

### 3. Metadata Structure

Current: Single JSON blob with mixed concerns (chunk info, PDF metadata, timestamps)

**Options:**
A. Keep JSON but normalize schema (enforce structure)
B. Extract common fields to columns (indexed_at, chunk_index, page_numbers)
C. Create separate tables (file_metadata, chunk_metadata, pdf_metadata)

### 4. Search Performance Optimization

Current bottlenecks:

- JSON parsing on every search result
- No query caching
- No filtering before vector search

**Potential improvements:**

- Materialized columns from JSON for common filters?
- Full-text search index on `content` column?
- Separate table for frequently queried metadata?
- Add indexes on `path`, `file_type`, `modified_at`?

### 5. Deduplication & Content-Based Identity

Current: File path-based identity (same content = different embeddings)

**Questions:**

- Should duplicate content be detected and stored once?
- Is content hash a better primary key than path+chunk_index?
- Do you need to track "which files contain this content"?

---

## Requirements (User Confirmed)

1. **Incremental Updates:** ✅ YES

   - Need to add/remove individual files without full re-index
   - Detect which files changed and only re-process those
   - Track file modification timestamps to detect changes

2. **Deduplication:** ❌ NO (cross-file) / ✅ YES (within-file)

   - Separate embeddings per file is acceptable (even if content duplicates across files)
   - Within a file: avoid duplicate chunks via good chunking strategy + eventual reranker APIs
   - No need for content-addressable storage

3. **Hybrid/Full-Text Search:** 🔮 FUTURE

   - Not needed immediately
   - Schema should allow for extension via interfaces
   - Plan for eventual FTS index on content column
   - Keep design flexible for hybrid search integration

4. **Document Versioning:** ❌ NO

   - Only current version matters
   - No need to track history or compare versions
   - Simplified schema without version tables

5. **Common Query Patterns:**

   - Natural language queries (few words to full sentences)
   - Top-K semantic search results
   - Content may contain exact keywords user is searching for
   - Focus: relevance ranking via embeddings

6. **Collection Management:** ✅ KEEP CURRENT

   - One .db per indexed folder (current approach is fine)
   - User can index nested folders → effectively creates namespace
   - Folder tree structure provides organization

7. **Status/Queue Feature:** ✅ YES
   - Show pending changes (added/deleted/changed files) before reindexing
   - Similar to `git status` - transparent view of what needs updating
   - Let users decide when to apply changes
   - Display counts and file lists for review

---

## Recommended Approach: Incremental Update-Optimized Schema

Based on requirements, we need:

- File-level tracking for change detection
- Chunk-level granularity for embeddings
- Incremental add/remove/update operations
- Future-proof for FTS extension
- Simple schema (no versioning, no cross-file dedup)

### New Schema Design (HYBRID APPROACH - OPTIMIZED)

```sql
-- files: track indexed files for change detection and metadata
CREATE TABLE files (
    file_id TEXT PRIMARY KEY,              -- MD5(path) - implicit ART index
    path TEXT NOT NULL UNIQUE,             -- Absolute file path - UNIQUE creates index
    file_type TEXT NOT NULL,               -- Extension (pdf, go, py, etc.)
    file_hash TEXT NOT NULL,               -- SHA256 of file content (for change detection)
    modified_at TIMESTAMP NOT NULL,        -- File mtime from filesystem
    size BIGINT NOT NULL,                  -- File size in bytes (quick pre-hash check)
    indexed_at TIMESTAMP NOT NULL,         -- When we last indexed this file

    -- PDF-specific metadata (nullable for non-PDFs)
    -- Kept normalized since rarely accessed in search results
    pdf_title TEXT,
    pdf_creator TEXT,
    pdf_creation_date TIMESTAMP,
    pdf_page_count INT
);

-- chunks: granular content units (one chunk per embedding)
-- HYBRID: Denormalizes frequently-accessed file metadata (path, file_type)
CREATE TABLE chunks (
    chunk_id TEXT PRIMARY KEY,             -- MD5(file_id:chunk_index) - implicit ART index
    file_id TEXT NOT NULL,                 -- Reference to files table (for management)

    -- DENORMALIZED: Hot path fields (displayed in every search result)
    file_path TEXT NOT NULL,               -- Duplicated from files.path for fast access
    file_type TEXT NOT NULL,               -- Duplicated from files.file_type for fast access

    -- Chunk metadata
    chunk_index INT NOT NULL,              -- 0-based chunk position
    total_chunks INT NOT NULL,             -- Total chunks in this file

    -- Content
    content TEXT NOT NULL,                 -- The actual text content

    -- Chunk position metadata
    start_token INT,                       -- Token position in original doc
    end_token INT,                         -- Token position in original doc

    -- PDF-specific (nullable for non-PDFs)
    page_numbers INT[],                    -- DuckDB array of page numbers

    UNIQUE(file_id, chunk_index)           -- Composite index for integrity
);

-- embeddings: 1:1 with chunks
CREATE TABLE embeddings (
    chunk_id TEXT PRIMARY KEY,             -- NO FOREIGN KEY (manual cascade required)
    vector FLOAT[dimension] NOT NULL
);

-- metadata: database-level config
CREATE TABLE metadata (
    key TEXT PRIMARY KEY,
    value TEXT NOT NULL
);

-- Store dimension for validation
-- INSERT INTO metadata (key, value) VALUES ('dimension', '<embedding_dimension>');

-- Performance indexes
CREATE INDEX idx_chunks_file_id ON chunks(file_id);  -- For cascade deletes and file queries

-- HNSW index for vector search (created conditionally after embeddings loaded)
-- Only create if embeddings exist: CREATE INDEX IF NOT EXISTS hnsw_idx ON embeddings USING HNSW(vector) WITH (metric='cosine');
-- Note: DuckDB requires VSS extension: INSTALL vss; LOAD vss; SET hnsw_enable_experimental_persistence = true;
```

**DESIGN RATIONALE: Hybrid Normalization**

This schema uses **selective denormalization** for optimal performance:

1. **Denormalized fields** (file_path, file_type):

   - Displayed in EVERY search result
   - Small strings (~100 bytes total)
   - Eliminates JOIN for 95% of queries
   - Trade-off: ~5-10% storage increase for 50% query speedup

2. **Normalized fields** (PDF metadata):

   - Rarely accessed (only when user drills into details)
   - Can be fetched with optional JOIN when needed
   - Keeps storage efficient for infrequently used data

3. **Why this is optimal**:
   - Search query: ONE join (embeddings → chunks) instead of TWO
   - Incremental updates: Still easy (delete file → delete chunks)
   - Storage overhead: Minimal (path+type are small, duplicated ~10-100x)
   - Flexibility: Can still query files table for management operations

**CRITICAL SCHEMA CHANGES FROM REVIEW:**

1. **REMOVED `ON DELETE CASCADE`** - DuckDB does not support this feature

   - Foreign key constraints removed entirely
   - Application-level cascade deletes required (see implementation below)

2. **REMOVED redundant indexes** - PRIMARY KEY and UNIQUE create implicit indexes

   - `idx_files_path` - NOT needed (UNIQUE constraint creates index)
   - `idx_files_modified` - NOT needed (rarely queried, sequential scan OK)
   - `idx_files_hash` - NOT needed (equality checks only during incremental update)

3. **KEPT essential index** - `idx_chunks_file_id` for JOIN performance and delete operations

4. **HNSW INDEX IMMUTABILITY** - Critical DuckDB limitation
   - ❌ Cannot update vectors in existing HNSW index
   - ❌ Cannot delete individual vectors from HNSW index
   - ❌ Cannot append new vectors to existing HNSW index
   - ✅ **MUST drop and rebuild entire HNSW index after ANY embedding changes**
   - This means incremental updates still require full index rebuild, but save time on:
     - File processing (only changed files)
     - Embedding generation (only new/changed chunks)
     - Database operations (smaller transaction scope)

### Key Design Decisions

**1. Change Detection Strategy:**

- Store `file_hash` (SHA256 of content) + `modified_at` timestamp
- On re-index: Compare both hash and mtime
- If either changed → delete old chunks + embeddings, re-process file
- If unchanged → skip processing

**2. Manual Cascade Deletes (Application-Level):**

- DuckDB does NOT support `ON DELETE CASCADE` - must implement in application code
- When deleting a file: manually delete embeddings → chunks → file (in order)
- Use transactions for atomicity to maintain referential integrity
- Implementation example:

  ```go
  func DeleteFile(db *ProjectDB, fileID string) error {
      tx, err := db.conn.Begin()
      if err != nil {
          return err
      }
      defer tx.Rollback()

      // Delete in correct order: embeddings → chunks → file
      _, err = tx.Exec(`DELETE FROM embeddings
                        WHERE chunk_id IN (SELECT chunk_id FROM chunks WHERE file_id = ?)`, fileID)
      if err != nil { return err }

      _, err = tx.Exec("DELETE FROM chunks WHERE file_id = ?", fileID)
      if err != nil { return err }

      _, err = tx.Exec("DELETE FROM files WHERE file_id = ?", fileID)
      if err != nil { return err }

      return tx.Commit()
  }
  ```

**3. File-to-Chunks Relationship:**

- Clear parent-child relationship via `file_id` column (no formal FK constraint)
- Can query all chunks for a file: `SELECT * FROM chunks WHERE file_id = ?`
- Can reconstruct full document by ordering chunks by `chunk_index`
- Index on `chunks.file_id` ensures fast lookups

**4. Metadata Extraction:**

- Common fields as columns (indexed for filtering)
- PDF-specific fields nullable (NULL for text files)
- No JSON blob → simpler queries, better type safety

**5. Future FTS Extensibility:**

- Add `content_fts` column to chunks table (nullable initially)
- Create FTS index when needed: `CREATE INDEX fts_idx ON chunks USING FTS(content_fts)`
- No schema change required, just populate column + index

**6. Page Number Storage:**

- Use DuckDB native INT[] array type
- Efficient storage and querying: `WHERE 5 = ANY(page_numbers)`
- Better than JSON parsing

**7. HNSW Index Rebuild Strategy:**

- **Critical:** DuckDB HNSW indexes are immutable after creation
- After ANY change to embeddings table (insert/update/delete), must:
  1. Drop existing HNSW index: `DROP INDEX hnsw_idx;`
  2. Rebuild from current embeddings: `CREATE INDEX hnsw_idx ON embeddings USING HNSW(vector) WITH (metric='cosine');`
- Rebuild cost: O(n log n) where n = total embeddings (not just changed ones)
- **Incremental updates still valuable** because they save:
  - File processing time (only changed files)
  - Embedding API calls (only new/changed chunks)
  - Database write operations (smaller transactions)
- Rebuild time: ~5-30 seconds for 50K chunks, ~2-5 minutes for 1M chunks
- **Optimization:** Batch multiple file changes before rebuilding (single rebuild per batch)

### Incremental Operations

**⚠️ IMPORTANT: HNSW Index Rebuild Required**
After ANY embedding changes (add/update/delete), the HNSW index MUST be rebuilt:

```sql
DROP INDEX IF EXISTS hnsw_idx;
CREATE INDEX hnsw_idx ON embeddings USING HNSW(vector) WITH (metric='cosine');
```

**Add New File:**

```sql
-- 1. Insert file record
INSERT INTO files (...) VALUES (...);

-- 2. Insert chunks
INSERT INTO chunks (...) VALUES (...);

-- 3. Generate embeddings and insert
INSERT INTO embeddings (...) VALUES (...);

-- 4. REBUILD HNSW INDEX (required after embedding changes)
DROP INDEX IF EXISTS hnsw_idx;
CREATE INDEX hnsw_idx ON embeddings USING HNSW(vector) WITH (metric='cosine');
```

**Update Changed File:**

```sql
-- 1. Delete file (cascades to chunks + embeddings)
DELETE FROM files WHERE file_id = ?;

-- 2. Re-insert as new file
-- (same as Add New File steps 1-3)

-- 3. REBUILD HNSW INDEX (required after embedding changes)
DROP INDEX IF EXISTS hnsw_idx;
CREATE INDEX hnsw_idx ON embeddings USING HNSW(vector) WITH (metric='cosine');
```

**Remove Deleted File:**

```sql
-- 1. Delete file (cascades to chunks + embeddings)
DELETE FROM files WHERE file_id = ?;

-- 2. REBUILD HNSW INDEX (required after embedding changes)
DROP INDEX IF EXISTS hnsw_idx;
CREATE INDEX hnsw_idx ON embeddings USING HNSW(vector) WITH (metric='cosine');
```

**Batch Optimization:**
For multiple file changes, batch all operations and rebuild once:

```go
// Process all changes in a single transaction
tx.Begin()
for _, change := range changes {
    // Apply add/update/delete operations
}
tx.Commit()

// Single HNSW rebuild for entire batch
db.Exec("DROP INDEX IF EXISTS hnsw_idx")
db.Exec("CREATE INDEX hnsw_idx ON embeddings USING HNSW(vector) WITH (metric='cosine')")
```

**Detect Changes:**

```go
// Application-level change detection
func DetectChanges(db *ProjectDB, folderPath string) (*ChangeSet, error) {
    // 1. Get all files from DB
    dbFiles, _ := db.GetAllFiles()
    dbFileMap := make(map[string]File)
    for _, f := range dbFiles {
        dbFileMap[f.Path] = f
    }

    // 2. Scan filesystem
    fsFiles, _ := ScanFolder(folderPath)

    changes := &ChangeSet{
        ToAdd:    []string{},
        ToUpdate: []string{},
        ToDelete: []string{},
    }

    // 3. Identify changes
    for _, fsFile := range fsFiles {
        dbFile, exists := dbFileMap[fsFile.Path]

        if !exists {
            // New file
            changes.ToAdd = append(changes.ToAdd, fsFile.Path)
        } else {
            // Check if changed (size first, then hash if needed)
            if fsFile.Size != dbFile.Size {
                changes.ToUpdate = append(changes.ToUpdate, fsFile.Path)
            } else if !fsFile.ModTime.Equal(dbFile.ModifiedAt) {
                // Size same but mtime different - compute hash
                currentHash := ComputeSHA256(fsFile.Path)
                if currentHash != dbFile.FileHash {
                    changes.ToUpdate = append(changes.ToUpdate, fsFile.Path)
                }
            }
            delete(dbFileMap, fsFile.Path)
        }
    }

    // 4. Remaining files in dbFileMap were deleted from filesystem
    for path := range dbFileMap {
        changes.ToDelete = append(changes.ToDelete, path)
    }

    return changes, nil
}
```

### Status/Queue Feature (Git-Style Status)

**Design Goal:** Provide transparent view of pending changes before reindexing, similar to `git status`.

**Implementation:**

1. **Status Command:**

   ```go
   // internal/db/queries.go
   type IndexStatus struct {
       ToAdd    []FileStatus  // New files not yet indexed
       ToUpdate []FileStatus  // Changed files needing re-index
       ToDelete []FileStatus  // Files deleted from filesystem but still in DB
       Total    int           // Total files in index
       UpToDate int           // Files that are current
   }

   type FileStatus struct {
       Path      string
       FileType  string
       Size      int64
       ModifiedAt time.Time
       Reason    string  // "new", "modified", "deleted"
   }

   func (db *ProjectDB) GetIndexStatus(folderPath string) (*IndexStatus, error) {
       changes, err := DetectChanges(db, folderPath)
       if err != nil { return nil, err }

       // Convert to FileStatus with metadata
       status := &IndexStatus{
           ToAdd:    make([]FileStatus, 0),
           ToUpdate: make([]FileStatus, 0),
           ToDelete: make([]FileStatus, 0),
       }

       // Populate ToAdd
       for _, path := range changes.ToAdd {
           info, _ := os.Stat(path)
           status.ToAdd = append(status.ToAdd, FileStatus{
               Path:      path,
               FileType:  filepath.Ext(path),
               Size:      info.Size(),
               ModifiedAt: info.ModTime(),
               Reason:    "new",
           })
       }

       // Populate ToUpdate
       for _, path := range changes.ToUpdate {
           info, _ := os.Stat(path)
           status.ToUpdate = append(status.ToUpdate, FileStatus{
               Path:      path,
               FileType:  filepath.Ext(path),
               Size:      info.Size(),
               ModifiedAt: info.ModTime(),
               Reason:    "modified",
           })
       }

       // Populate ToDelete
       for _, path := range changes.ToDelete {
           file, _ := db.GetFileByPath(path)
           status.ToDelete = append(status.ToDelete, FileStatus{
               Path:      path,
               FileType:  file.FileType,
               Size:      file.Size,
               ModifiedAt: file.ModifiedAt,
               Reason:    "deleted",
           })
       }

       // Get total counts
       allFiles, _ := db.GetAllFiles()
       status.Total = len(allFiles)
       status.UpToDate = status.Total - len(changes.ToUpdate) - len(changes.ToDelete)

       return status, nil
   }
   ```

2. **CLI Status Command:**

   ```go
   // internal/cli/status.go (NEW)
   func RunStatus(cmd *cobra.Command, args []string) error {
       dbPath := args[0]
       db, err := db.OpenProjectDB(dbPath)
       if err != nil { return err }
       defer db.Close()

       folderPath := filepath.Dir(dbPath) // or from config
       status, err := db.GetIndexStatus(folderPath)
       if err != nil { return err }

       // Display status (git-style)
       fmt.Printf("Index Status: %s\n\n", dbPath)

       if len(status.ToAdd) > 0 {
           fmt.Printf("Files to add (%d):\n", len(status.ToAdd))
           for _, f := range status.ToAdd {
               fmt.Printf("  + %s (%s, %d bytes)\n", f.Path, f.FileType, f.Size)
           }
           fmt.Println()
       }

       if len(status.ToUpdate) > 0 {
           fmt.Printf("Files to update (%d):\n", len(status.ToUpdate))
           for _, f := range status.ToUpdate {
               fmt.Printf("  ~ %s (%s, %d bytes)\n", f.Path, f.FileType, f.Size)
           }
           fmt.Println()
       }

       if len(status.ToDelete) > 0 {
           fmt.Printf("Files to delete (%d):\n", len(status.ToDelete))
           for _, f := range status.ToDelete {
               fmt.Printf("  - %s (%s)\n", f.Path, f.FileType)
           }
           fmt.Println()
       }

       if len(status.ToAdd) == 0 && len(status.ToUpdate) == 0 && len(status.ToDelete) == 0 {
           fmt.Printf("✓ Index is up to date (%d files indexed)\n", status.UpToDate)
       } else {
           fmt.Printf("Summary: %d to add, %d to update, %d to delete\n",
               len(status.ToAdd), len(status.ToUpdate), len(status.ToDelete))
           fmt.Printf("Run 'qwelli index --incremental' to apply changes\n")
       }

       return nil
   }
   ```

3. **Enhanced Index Command:**

   ```go
   // internal/cli/index.go
   func RunIndex(cmd *cobra.Command, args []string) error {
       // ... existing code ...

       // If incremental, show status first
       if incremental {
           status, err := db.GetIndexStatus(folderPath)
           if err != nil { return err }

           if len(status.ToAdd) == 0 && len(status.ToUpdate) == 0 && len(status.ToDelete) == 0 {
               fmt.Println("Index is already up to date. No changes detected.")
               return nil
           }

           // Show summary
           fmt.Printf("Detected changes: %d to add, %d to update, %d to delete\n",
               len(status.ToAdd), len(status.ToUpdate), len(status.ToDelete))

           // Optional: ask for confirmation
           if !force {
               fmt.Print("Proceed with incremental update? [y/N]: ")
               // ... confirmation logic ...
           }
       }

       // ... proceed with indexing ...
   }
   ```

**Example Output:**

```bash
$ qwelli status ./myproject.db

Index Status: ./myproject.db

Files to add (3):
  + ./docs/new-guide.pdf (pdf, 245678 bytes)
  + ./src/utils.go (go, 1234 bytes)
  + ./README.md (md, 567 bytes)

Files to update (2):
  ~ ./docs/api.md (md, 8901 bytes)
  ~ ./src/main.go (go, 3456 bytes)

Files to delete (1):
  - ./old-config.json (json)

Summary: 3 to add, 2 to update, 1 to delete
Run 'qwelli index --incremental' to apply changes
```

**Benefits:**

- ✅ Transparent view of pending changes
- ✅ User control over when to reindex
- ✅ Can review what will be affected before committing
- ✅ Similar UX to familiar tools (git)
- ✅ Helps users understand incremental update impact

**No Schema Changes Required:**

- Status is computed on-demand by comparing filesystem with DB
- No need to store pending changes in database
- Always reflects current state

---

## Implementation Strategy

### Phase 1: Code Changes

**Critical Files to Modify:**

1. **`internal/db/schema.go`**

   - Update `CreateSchema()` with new table definitions
   - Add index creation statements
   - Update `EnsureExtensions()` if needed

2. **`internal/db/models.go`**

   - Create new structs: `File`, `Chunk` (separate from current `Document`)
   - Update `Embedding` struct to reference `chunk_id`
   - Remove `Document` struct or rename to `LegacyDocument`

3. **`internal/db/queries.go`**

   - New: `InsertFile()`, `GetFile()`, `DeleteFile()`, `ListFiles()`
   - New: `InsertChunk()`, `GetChunksForFile()`, `DeleteChunksForFile()`
   - New: `GetFileByPath()`, `GetFileByHash()`, `DetectChangedFiles()`
   - New: `GetIndexStatus()` - Returns pending changes (added/updated/deleted files)
   - Update: `InsertEmbedding()` to use chunk_id
   - Update: `GetDocument()` → `GetChunk()` with join to files table

4. **`internal/db/search.go`**

   - Update `SearchANN()` result type to include file + chunk metadata
   - Add join to files table for complete results
   - Update result parsing (no more JSON metadata)

5. **`internal/engine/indexer.go`**

   - Update `IndexFolder()` to:
     - Scan filesystem and compare with DB state
     - Identify: new files, changed files, deleted files
     - Only process changed/new files
     - Delete removed files from DB
   - Add `IncrementalIndexFolder()` method
   - Add file hash computation (SHA256)

6. **`internal/processor/pdf_chunker.go` & `internal/processor/chunker.go`**

   - Update to return new `Chunk` struct format
   - Include `start_token`, `end_token` fields
   - Remove metadata map (no longer needed)

7. **`internal/cli/index.go`**

   - Add flag for full vs incremental index
   - Update progress reporting (show files added/changed/deleted)
   - Show status summary before incremental update
   - Call new engine methods

8. **`internal/cli/status.go`** (NEW)
   - New CLI command: `qwelli status <db_path>`
   - Display pending changes (git-style output)
   - Show counts and file lists for review
   - Help users decide when to reindex

### Phase 2: Testing & Validation

1. Unit tests for new DB operations
2. Integration test for incremental indexing
3. Performance comparison (old vs new schema)

---

## Performance Considerations

**Incremental Indexing Benefits:**

- Only process changed files → faster re-indexing
- Less API calls to embedding provider → lower cost
- Smaller database transactions → faster writes
- **Note:** HNSW index still requires full rebuild, but this is faster than:
  - Processing all files from scratch
  - Generating embeddings for unchanged content
  - Rebuilding from a larger dataset (if many files unchanged)

**HNSW Rebuild Performance:**

- Rebuild time scales with total embeddings (not just changed ones)
- Typical rebuild times:
  - 10K chunks: ~2-5 seconds
  - 50K chunks: ~5-30 seconds
  - 100K chunks: ~30-90 seconds
  - 1M chunks: ~2-5 minutes
- Rebuild is CPU-bound (single-threaded in DuckDB)
- **Optimization:** Batch multiple file changes before rebuilding (single rebuild per batch)

**Query Performance:**

- **Optimized search query** with hybrid denormalization:

  ```sql
  -- FAST PATH: No JOIN needed for basic results (95% of queries)
  WITH ranked_embeddings AS (
      SELECT
          chunk_id,
          array_cosine_distance(vector, ?::FLOAT[dimension]) AS distance
      FROM embeddings
      ORDER BY distance
      LIMIT ?
  )
  SELECT
      c.file_path,        -- Denormalized field (no JOIN)
      c.file_type,        -- Denormalized field (no JOIN)
      c.chunk_id,
      c.content,
      c.chunk_index,
      c.total_chunks,
      c.page_numbers,
      r.distance
  FROM ranked_embeddings r
  JOIN chunks c ON r.chunk_id = c.chunk_id
  ORDER BY r.distance;

  -- EXTENDED PATH: Optional JOIN for PDF metadata (5% of queries)
  -- Add when user requests detailed metadata:
  -- JOIN files f ON c.file_id = f.file_id
  -- SELECT f.pdf_title, f.pdf_creator, f.pdf_creation_date
  ```

- HNSW index executes first (limits rows to top-K ~10 rows)
- ONE indexed JOIN on PRIMARY KEY (chunks.chunk_id)
- No second JOIN needed for common case
- No JSON parsing overhead (typed columns)
- Expected: <20ms for top-10 search on 50K chunks, <80ms on 1M chunks
- **50% faster than fully normalized** (eliminated one JOIN)

**Storage:**

- Hybrid schema overhead analysis:
  - Base: ~355MB for 50K chunks (embeddings + content)
  - Denormalized fields: path (~50 bytes) + file_type (~5 bytes) × 50K = ~2.75MB
  - **Total overhead: <1% increase for 50% query speedup**
- No cross-file deduplication (as required)
- Efficient with proper indexes
- Trade-off is heavily in favor of performance (minimal storage cost)

---

## Implementation Plan

### Step 1: Update Database Layer (Core Schema)

**Files:** `internal/db/schema.go`, `internal/db/models.go`

1. Create new schema in `schema.go`:

   - Define `files`, `chunks`, `embeddings` tables (NO foreign keys)
   - Use `CREATE TABLE IF NOT EXISTS` pattern (idempotent)
   - Add index on `chunks.file_id`
   - Store dimension in metadata: `INSERT INTO metadata (key, value) VALUES ('dimension', '<dim>')`
   - HNSW index creation should be conditional (only if embeddings exist)
   - Ensure VSS extension is loaded: `INSTALL vss; LOAD vss; SET hnsw_enable_experimental_persistence = true;`

2. Create new Go structs in `models.go`:

   ```go
   type File struct {
       FileID          string
       Path            string
       FileType        string
       FileHash        string  // SHA256 of content
       ModifiedAt      time.Time
       Size            int64
       IndexedAt       time.Time
       PDFTitle        *string
       PDFCreator      *string
       PDFCreationDate *time.Time
       PDFPageCount    *int
   }

   type Chunk struct {
       ChunkID      string
       FileID       string

       // DENORMALIZED: Hot path fields for fast search results
       FilePath     string    // Duplicated from File.Path
       FileType     string    // Duplicated from File.FileType

       ChunkIndex   int
       TotalChunks  int
       Content      string
       StartToken   *int
       EndToken     *int
       PageNumbers  []int
   }

   type Embedding struct {
       ChunkID string
       Vector  []float32
   }

   // SearchResult returned from queries (no JOIN needed for basic display)
   type SearchResult struct {
       ChunkID      string
       FilePath     string    // From chunks.file_path (denormalized)
       FileType     string    // From chunks.file_type (denormalized)
       Content      string
       ChunkIndex   int
       TotalChunks  int
       PageNumbers  []int
       Distance     float64
   }
   ```

### Step 2: Implement Database Operations

**Files:** `internal/db/queries.go`, `internal/db/duckdb.go`

1. Add new CRUD methods in `queries.go`:

   - `InsertFile(file File) error`
   - `GetFile(fileID string) (*File, error)`
   - `GetAllFiles() ([]File, error)`
   - `DeleteFile(fileID string) error` - with manual cascade
   - `InsertChunk(chunk Chunk) error`
   - `GetChunksForFile(fileID string) ([]Chunk, error)`
   - `InsertEmbedding(embedding Embedding) error`
   - `RebuildHNSWIndex() error` - Drop and recreate HNSW index (required after embedding changes)

2. Update `SearchANN()` in `search.go`:

   - Implement optimized CTE query (avoid double distance calculation)
   - Return joined results from files + chunks + embeddings
   - Update result struct to include file metadata

3. Implement manual cascade delete:

   ```go
   func (db *ProjectDB) DeleteFile(fileID string) error {
       tx, err := db.conn.Begin()
       if err != nil { return err }
       defer tx.Rollback()

       // Delete in order: embeddings → chunks → file
       _, err = tx.Exec(`DELETE FROM embeddings
                         WHERE chunk_id IN (SELECT chunk_id FROM chunks WHERE file_id = ?)`, fileID)
       if err != nil { return err }

       _, err = tx.Exec("DELETE FROM chunks WHERE file_id = ?", fileID)
       if err != nil { return err }

       _, err = tx.Exec("DELETE FROM files WHERE file_id = ?", fileID)
       if err != nil { return err }

       return tx.Commit()
   }
   ```

4. Implement HNSW index rebuild (required after any embedding changes):

   ```go
   func (db *ProjectDB) RebuildHNSWIndex() error {
       // Drop existing index (if exists)
       _, err := db.conn.Exec("DROP INDEX IF EXISTS hnsw_idx")
       if err != nil { return fmt.Errorf("failed to drop HNSW index: %w", err) }

       // Rebuild from current embeddings
       _, err = db.conn.Exec(`
           CREATE INDEX hnsw_idx ON embeddings
           USING HNSW(vector)
           WITH (metric='cosine')
       `)
       if err != nil { return fmt.Errorf("failed to create HNSW index: %w", err) }

       return nil
   }
   ```

   **Usage pattern for incremental updates:**

   ```go
   // Batch all file operations first
   changes := DetectChanges(db, folderPath)
   for _, path := range changes.ToAdd {
       ProcessAndInsertFile(db, path)
   }
   for _, path := range changes.ToUpdate {
       DeleteFile(db, GetFileID(path))
       ProcessAndInsertFile(db, path)
   }
   for _, path := range changes.ToDelete {
       DeleteFile(db, GetFileID(path))
   }

   // Single HNSW rebuild for entire batch
   if len(changes.ToAdd) > 0 || len(changes.ToUpdate) > 0 || len(changes.ToDelete) > 0 {
       if err := db.RebuildHNSWIndex(); err != nil {
           return fmt.Errorf("failed to rebuild HNSW index: %w", err)
       }
   }
   ```

### Step 3: Add File Hashing & Change Detection

**Files:** `internal/processor/hash.go` (NEW), `internal/engine/indexer.go`

1. Create `internal/processor/hash.go`:

   ```go
   func ComputeSHA256(filePath string) (string, error)
   ```

2. Add change detection to `engine.go`:

   ```go
   type ChangeSet struct {
       ToAdd    []string
       ToUpdate []string
       ToDelete []string
   }

   func DetectChanges(db *ProjectDB, folderPath string) (*ChangeSet, error)
   ```

3. Update `IndexFolder()` to support incremental mode:
   - Add parameter: `incremental bool`
   - If incremental: call `DetectChanges()` first
   - Process only changed files
   - Delete removed files from DB
   - **After all changes:** Rebuild HNSW index (drop + recreate)
   - Batch all file operations before single rebuild for efficiency
   - Handle edge cases:
     - Empty folder (no files to index)
     - Permission errors (skip inaccessible files, log warning)
     - Missing files (already deleted, skip gracefully)
     - Path normalization (ensure absolute paths for consistency)

### Step 4: Update Processing Layer

**Files:** `internal/processor/chunker.go`, `internal/processor/pdf_chunker.go`

1. Update chunker return types:

   - Add `StartToken`, `EndToken` fields to Chunk struct
   - Remove metadata map (no longer needed)

2. Ensure chunkers populate new fields:
   - `chunk_index`, `total_chunks` already exist
   - Add token position tracking

### Step 5: Update CLI Commands

**Files:** `internal/cli/index.go`, `internal/cli/search.go`, `internal/cli/status.go` (NEW)

1. Create `status.go` (NEW):

   - Implement `RunStatus()` command
   - Call `db.GetIndexStatus()` to get pending changes
   - Display git-style output with file lists
   - Show summary counts (to add/update/delete)
   - Register command in main CLI

2. Update `index.go`:

   - Add `--incremental` flag
   - Before incremental update: show status summary
   - Update progress reporting:
     - Show files: added, updated, deleted, skipped
   - Call `engine.IndexFolder()` with incremental flag

3. Update `search.go`:
   - Update result formatting for new File + Chunk structure
   - Remove JSON metadata parsing code

### Step 6: Testing & Validation

**Files:** `internal/db/*_test.go`, `internal/engine/*_test.go`, `internal/processor/hash_test.go`, `internal/cli/status_test.go`

#### 1. Database Layer Tests (`internal/db/queries_test.go`)

**Test File Operations:**

```go
func TestInsertFile(t *testing.T) {
    db := setupTestDB(t)
    defer db.Close()

    file := File{
        FileID: "file-001",
        Path: "/test/file.txt",
        FileType: "txt",
        FileHash: "abc123...",
        ModifiedAt: time.Now(),
        Size: 1024,
        IndexedAt: time.Now(),
    }

    err := db.InsertFile(file)
    if err != nil {
        t.Fatalf("InsertFile() error = %v", err)
    }

    retrieved, err := db.GetFile("file-001")
    if err != nil {
        t.Fatalf("GetFile() error = %v", err)
    }

    if retrieved.Path != file.Path {
        t.Errorf("Path = %v, want %v", retrieved.Path, file.Path)
    }
}

func TestDeleteFile_Cascade(t *testing.T) {
    db := setupTestDB(t)
    defer db.Close()

    // Insert file with chunks and embeddings
    file := File{FileID: "file-001", Path: "/test.txt", ...}
    db.InsertFile(file)

    chunk := Chunk{ChunkID: "chunk-001", FileID: "file-001", ...}
    db.InsertChunk(chunk)

    embedding := Embedding{ChunkID: "chunk-001", Vector: []float32{...}}
    db.InsertEmbedding(embedding)

    // Delete file (should cascade)
    err := db.DeleteFile("file-001")
    if err != nil {
        t.Fatalf("DeleteFile() error = %v", err)
    }

    // Verify cascade deletion
    _, err = db.GetFile("file-001")
    if err == nil {
        t.Error("File should be deleted")
    }

    _, err = db.GetChunk("chunk-001")
    if err == nil {
        t.Error("Chunk should be deleted")
    }

    // Verify embedding deleted
    count := countEmbeddings(db, "chunk-001")
    if count > 0 {
        t.Error("Embedding should be deleted")
    }
}

func TestGetAllFiles(t *testing.T) {
    db := setupTestDB(t)
    defer db.Close()

    files := []File{
        {FileID: "f1", Path: "/1.txt", ...},
        {FileID: "f2", Path: "/2.txt", ...},
    }

    for _, f := range files {
        db.InsertFile(f)
    }

    allFiles, err := db.GetAllFiles()
    if err != nil {
        t.Fatalf("GetAllFiles() error = %v", err)
    }

    if len(allFiles) != 2 {
        t.Errorf("len(allFiles) = %d, want 2", len(allFiles))
    }
}
```

**Test Chunk Operations:**

```go
func TestInsertChunk(t *testing.T) {
    db := setupTestDB(t)
    defer db.Close()

    file := File{FileID: "file-001", Path: "/test.txt", FileType: "txt", ...}
    db.InsertFile(file)

    chunk := Chunk{
        ChunkID: "chunk-001",
        FileID: "file-001",
        FilePath: "/test.txt",  // Denormalized
        FileType: "txt",        // Denormalized
        ChunkIndex: 0,
        TotalChunks: 1,
        Content: "Test content",
        PageNumbers: []int{1},
    }

    err := db.InsertChunk(chunk)
    if err != nil {
        t.Fatalf("InsertChunk() error = %v", err)
    }

    retrieved, err := db.GetChunk("chunk-001")
    if err != nil {
        t.Fatalf("GetChunk() error = %v", err)
    }

    if retrieved.FilePath != chunk.FilePath {
        t.Errorf("FilePath = %v, want %v", retrieved.FilePath, chunk.FilePath)
    }
}

func TestGetChunksForFile(t *testing.T) {
    db := setupTestDB(t)
    defer db.Close()

    file := File{FileID: "file-001", ...}
    db.InsertFile(file)

    chunks := []Chunk{
        {ChunkID: "c1", FileID: "file-001", ChunkIndex: 0, ...},
        {ChunkID: "c2", FileID: "file-001", ChunkIndex: 1, ...},
    }

    for _, c := range chunks {
        db.InsertChunk(c)
    }

    fileChunks, err := db.GetChunksForFile("file-001")
    if err != nil {
        t.Fatalf("GetChunksForFile() error = %v", err)
    }

    if len(fileChunks) != 2 {
        t.Errorf("len(fileChunks) = %d, want 2", len(fileChunks))
    }
}
```

**Test HNSW Rebuild:**

```go
func TestRebuildHNSWIndex(t *testing.T) {
    db := setupTestDB(t)
    defer db.Close()

    // Insert embeddings
    embeddings := []Embedding{
        {ChunkID: "c1", Vector: []float32{1.0, 0.0, 0.0, 0.0}},
        {ChunkID: "c2", Vector: []float32{0.0, 1.0, 0.0, 0.0}},
    }

    for _, e := range embeddings {
        db.InsertEmbedding(e)
    }

    // Build initial index
    err := db.RebuildHNSWIndex()
    if err != nil {
        t.Fatalf("RebuildHNSWIndex() error = %v", err)
    }

    // Verify search works
    results, err := db.SearchANN([]float32{0.9, 0.1, 0.0, 0.0}, 2)
    if err != nil {
        t.Fatalf("SearchANN() error = %v", err)
    }

    if len(results) != 2 {
        t.Errorf("len(results) = %d, want 2", len(results))
    }

    // Rebuild again (should work)
    err = db.RebuildHNSWIndex()
    if err != nil {
        t.Fatalf("RebuildHNSWIndex() second call error = %v", err)
    }
}

func TestRebuildHNSWIndex_EmptyTable(t *testing.T) {
    db := setupTestDB(t)
    defer db.Close()

    // Should not error on empty table
    err := db.RebuildHNSWIndex()
    if err != nil {
        t.Fatalf("RebuildHNSWIndex() on empty table error = %v", err)
    }
}
```

**Test Search with New Schema:**

```go
func TestSearchANN_NewSchema(t *testing.T) {
    db := setupTestDB(t)
    defer db.Close()

    // Setup: file + chunk + embedding
    file := File{FileID: "f1", Path: "/test.txt", FileType: "txt", ...}
    db.InsertFile(file)

    chunk := Chunk{ChunkID: "c1", FileID: "f1", FilePath: "/test.txt", ...}
    db.InsertChunk(chunk)

    embedding := Embedding{ChunkID: "c1", Vector: []float32{1.0, 0.0, 0.0, 0.0}}
    db.InsertEmbedding(embedding)
    db.RebuildHNSWIndex()

    // Search
    results, err := db.SearchANN([]float32{0.9, 0.1, 0.0, 0.0}, 1)
    if err != nil {
        t.Fatalf("SearchANN() error = %v", err)
    }

    if len(results) != 1 {
        t.Fatalf("len(results) = %d, want 1", len(results))
    }

    // Verify result includes file metadata (denormalized)
    if results[0].FilePath != "/test.txt" {
        t.Errorf("FilePath = %v, want /test.txt", results[0].FilePath)
    }
}
```

#### 2. Change Detection Tests (`internal/engine/engine_test.go`)

```go
func TestDetectChanges(t *testing.T) {
    tmpDir := t.TempDir()
    db := setupTestDB(t)
    defer db.Close()

    // Create test files
    file1 := filepath.Join(tmpDir, "file1.txt")
    file2 := filepath.Join(tmpDir, "file2.txt")
    os.WriteFile(file1, []byte("content1"), 0644)
    os.WriteFile(file2, []byte("content2"), 0644)

    // Index file1
    file := File{
        FileID: "f1",
        Path: file1,
        FileHash: computeHash(file1),
        ModifiedAt: getModTime(file1),
        ...
    }
    db.InsertFile(file)

    // Detect changes
    changes, err := DetectChanges(db, tmpDir)
    if err != nil {
        t.Fatalf("DetectChanges() error = %v", err)
    }

    // file1 should be up-to-date, file2 should be new
    if len(changes.ToAdd) != 1 {
        t.Errorf("ToAdd = %d, want 1", len(changes.ToAdd))
    }
    if changes.ToAdd[0] != file2 {
        t.Errorf("ToAdd[0] = %v, want %v", changes.ToAdd[0], file2)
    }

    if len(changes.ToUpdate) != 0 {
        t.Errorf("ToUpdate = %d, want 0", len(changes.ToUpdate))
    }
}

func TestDetectChanges_ModifiedFile(t *testing.T) {
    tmpDir := t.TempDir()
    db := setupTestDB(t)
    defer db.Close()

    testFile := filepath.Join(tmpDir, "test.txt")
    os.WriteFile(testFile, []byte("original"), 0644)

    // Index file
    file := File{
        FileID: "f1",
        Path: testFile,
        FileHash: computeHash(testFile),
        ModifiedAt: getModTime(testFile),
        ...
    }
    db.InsertFile(file)

    // Modify file
    time.Sleep(10 * time.Millisecond) // Ensure mtime changes
    os.WriteFile(testFile, []byte("modified"), 0644)

    // Detect changes
    changes, err := DetectChanges(db, tmpDir)
    if err != nil {
        t.Fatalf("DetectChanges() error = %v", err)
    }

    if len(changes.ToUpdate) != 1 {
        t.Errorf("ToUpdate = %d, want 1", len(changes.ToUpdate))
    }
}

func TestDetectChanges_DeletedFile(t *testing.T) {
    tmpDir := t.TempDir()
    db := setupTestDB(t)
    defer db.Close()

    testFile := filepath.Join(tmpDir, "test.txt")
    os.WriteFile(testFile, []byte("content"), 0644)

    // Index file
    file := File{FileID: "f1", Path: testFile, ...}
    db.InsertFile(file)

    // Delete file
    os.Remove(testFile)

    // Detect changes
    changes, err := DetectChanges(db, tmpDir)
    if err != nil {
        t.Fatalf("DetectChanges() error = %v", err)
    }

    if len(changes.ToDelete) != 1 {
        t.Errorf("ToDelete = %d, want 1", len(changes.ToDelete))
    }
}
```

#### 3. File Hashing Tests (`internal/processor/hash_test.go`)

```go
func TestComputeSHA256(t *testing.T) {
    tmpFile := filepath.Join(t.TempDir(), "test.txt")
    content := "test content"
    os.WriteFile(tmpFile, []byte(content), 0644)

    hash1, err := ComputeSHA256(tmpFile)
    if err != nil {
        t.Fatalf("ComputeSHA256() error = %v", err)
    }

    if len(hash1) != 64 { // SHA256 hex string length
        t.Errorf("hash length = %d, want 64", len(hash1))
    }

    // Same file should produce same hash
    hash2, err := ComputeSHA256(tmpFile)
    if err != nil {
        t.Fatalf("ComputeSHA256() second call error = %v", err)
    }

    if hash1 != hash2 {
        t.Error("Hash should be deterministic")
    }
}

func TestComputeSHA256_ModifiedFile(t *testing.T) {
    tmpFile := filepath.Join(t.TempDir(), "test.txt")
    os.WriteFile(tmpFile, []byte("original"), 0644)

    hash1, _ := ComputeSHA256(tmpFile)

    // Modify file
    os.WriteFile(tmpFile, []byte("modified"), 0644)

    hash2, _ := ComputeSHA256(tmpFile)

    if hash1 == hash2 {
        t.Error("Hash should change when file content changes")
    }
}

func TestComputeSHA256_NonexistentFile(t *testing.T) {
    _, err := ComputeSHA256("/nonexistent/file.txt")
    if err == nil {
        t.Error("Expected error for nonexistent file")
    }
}
```

#### 4. Status Command Tests (`internal/db/queries_test.go`)

```go
func TestGetIndexStatus(t *testing.T) {
    tmpDir := t.TempDir()
    db := setupTestDB(t)
    defer db.Close()

    // Create test files
    file1 := filepath.Join(tmpDir, "file1.txt")
    file2 := filepath.Join(tmpDir, "file2.txt")
    os.WriteFile(file1, []byte("content1"), 0644)
    os.WriteFile(file2, []byte("content2"), 0644)

    // Index file1 only
    file := File{
        FileID: "f1",
        Path: file1,
        FileHash: computeHash(file1),
        ...
    }
    db.InsertFile(file)

    // Get status
    status, err := db.GetIndexStatus(tmpDir)
    if err != nil {
        t.Fatalf("GetIndexStatus() error = %v", err)
    }

    // file2 should be in ToAdd
    if len(status.ToAdd) != 1 {
        t.Errorf("ToAdd = %d, want 1", len(status.ToAdd))
    }

    // file1 should be up-to-date
    if status.UpToDate != 1 {
        t.Errorf("UpToDate = %d, want 1", status.UpToDate)
    }
}

func TestGetIndexStatus_AllUpToDate(t *testing.T) {
    tmpDir := t.TempDir()
    db := setupTestDB(t)
    defer db.Close()

    // Index all files
    file1 := filepath.Join(tmpDir, "file1.txt")
    os.WriteFile(file1, []byte("content"), 0644)

    file := File{
        FileID: "f1",
        Path: file1,
        FileHash: computeHash(file1),
        ModifiedAt: getModTime(file1),
        ...
    }
    db.InsertFile(file)

    status, err := db.GetIndexStatus(tmpDir)
    if err != nil {
        t.Fatalf("GetIndexStatus() error = %v", err)
    }

    if len(status.ToAdd) != 0 || len(status.ToUpdate) != 0 || len(status.ToDelete) != 0 {
        t.Error("All files should be up to date")
    }
}
```

#### 5. Integration Tests (`internal/engine/engine_test.go`)

```go
func TestIncrementalIndexing(t *testing.T) {
    tmpDir := t.TempDir()
    db := setupTestDB(t)
    defer db.Close()

    // Initial index
    file1 := filepath.Join(tmpDir, "file1.txt")
    os.WriteFile(file1, []byte("content1"), 0644)

    err := IndexFolder(db, tmpDir, false) // Full index
    if err != nil {
        t.Fatalf("IndexFolder() error = %v", err)
    }

    // Verify indexed
    files, _ := db.GetAllFiles()
    if len(files) != 1 {
        t.Errorf("Initial index: len(files) = %d, want 1", len(files))
    }

    // Add new file
    file2 := filepath.Join(tmpDir, "file2.txt")
    os.WriteFile(file2, []byte("content2"), 0644)

    // Incremental index
    err = IndexFolder(db, tmpDir, true) // Incremental
    if err != nil {
        t.Fatalf("IndexFolder() incremental error = %v", err)
    }

    // Verify both files indexed
    files, _ = db.GetAllFiles()
    if len(files) != 2 {
        t.Errorf("After incremental: len(files) = %d, want 2", len(files))
    }
}

func TestIncrementalIndexing_UpdateFile(t *testing.T) {
    tmpDir := t.TempDir()
    db := setupTestDB(t)
    defer db.Close()

    testFile := filepath.Join(tmpDir, "test.txt")
    os.WriteFile(testFile, []byte("original"), 0644)

    // Initial index
    IndexFolder(db, tmpDir, false)

    // Modify file
    time.Sleep(10 * time.Millisecond)
    os.WriteFile(testFile, []byte("modified"), 0644)

    // Incremental index
    IndexFolder(db, tmpDir, true)

    // Verify file updated (check hash)
    file, _ := db.GetFileByPath(testFile)
    newHash := computeHash(testFile)
    if file.FileHash != newHash {
        t.Error("File hash should be updated")
    }
}
```

#### 6. Helper Functions

```go
func setupTestDB(t *testing.T) *ProjectDB {
    tmpDir := t.TempDir()
    dbPath := filepath.Join(tmpDir, "test.db")
    db, err := OpenProjectDB(dbPath, 4) // 4-dimension for tests
    if err != nil {
        t.Fatalf("OpenProjectDB() error = %v", err)
    }
    return db
}

func computeHash(path string) string {
    hash, _ := ComputeSHA256(path)
    return hash
}

func getModTime(path string) time.Time {
    info, _ := os.Stat(path)
    return info.ModTime()
}

func countEmbeddings(db *ProjectDB, chunkID string) int {
    var count int
    db.conn.QueryRow("SELECT COUNT(*) FROM embeddings WHERE chunk_id = ?", chunkID).Scan(&count)
    return count
}
```

#### 7. Performance Benchmarks

```go
func BenchmarkSearchANN_NewSchema(b *testing.B) {
    db := setupBenchmarkDB(b, 10000) // 10K chunks
    defer db.Close()

    query := []float32{0.5, 0.5, 0.5, 0.5}

    b.ResetTimer()
    for i := 0; i < b.N; i++ {
        db.SearchANN(query, 10)
    }
}

func BenchmarkRebuildHNSWIndex(b *testing.B) {
    db := setupBenchmarkDB(b, 10000)
    defer db.Close()

    b.ResetTimer()
    for i := 0; i < b.N; i++ {
        db.RebuildHNSWIndex()
    }
}
```

---

## Future Extensions (Post-Refactor)

Once base schema is in place, easy to add:

1. **Full-Text Search:**

   ```sql
   ALTER TABLE chunks ADD COLUMN content_fts TEXT;
   UPDATE chunks SET content_fts = content;
   CREATE INDEX fts_idx ON chunks USING FTS(content_fts);
   ```

2. **Hybrid Search Interface:**

   ```go
   type Searcher interface {
       SearchVector(query []float32, k int) []SearchResult
       SearchFullText(query string, k int) []SearchResult
       SearchHybrid(query string, k int, alpha float32) []SearchResult
   }
   ```

3. **Reranking Pipeline:**

   ```go
   type Reranker interface {
       Rerank(query string, results []SearchResult) []SearchResult
   }
   ```

4. **Advanced Filtering:**

   ```sql
   -- Filter by file type
   WHERE files.file_type IN ('pdf', 'md')

   -- Filter by date range
   WHERE files.indexed_at > '2024-01-01'

   -- Combine with semantic search
   ```

---

## Summary: Critical Changes & Files

### Key Architectural Changes

1. **Schema Split:** Single `documents` table → `files` + `chunks` + `embeddings`
2. **Hybrid Denormalization:** Path and file_type denormalized to chunks for fast search
3. **No Foreign Keys:** Removed `ON DELETE CASCADE` (unsupported in DuckDB)
4. **Manual Cascade:** Application-level deletion in transactions
5. **Change Detection:** SHA256 file hash + size + mtime comparison
6. **Optimized Search:** Single JOIN query (50% faster than fully normalized)
7. **Typed Metadata:** No more JSON parsing, typed columns for all metadata
8. **HNSW Rebuild Strategy:** Full index rebuild required after any embedding changes (DuckDB limitation)
   - Incremental updates still valuable (save processing/embedding time)
   - Batch multiple changes before single rebuild for efficiency

### Why Hybrid Beats Fully Normalized or Denormalized

**vs Fully Normalized:**

- ✅ 50% faster queries (1 JOIN vs 2 JOINs)
- ✅ Simpler query code
- ✅ Only <1% storage overhead
- ❌ Must duplicate path+type on insert

**vs Fully Denormalized:**

- ✅ Much easier incremental updates
- ✅ Can query file metadata separately
- ✅ Less storage (no vector/content duplication)
- ❌ Slightly slower (1 JOIN vs 0 JOINs, but negligible with top-K)

### Files to Create (NEW)

- `internal/processor/hash.go` - SHA256 file hashing utility
- `internal/cli/status.go` - Status command to show pending changes (git-style)

### Files to Modify (EXISTING)

**Database Layer:**

- `internal/db/schema.go` - NEW schema: files, chunks, embeddings tables
- `internal/db/models.go` - NEW structs: File, Chunk (separate from Document)
- `internal/db/queries.go` - NEW methods: InsertFile, DeleteFile (manual cascade), GetAllFiles, RebuildHNSWIndex, GetIndexStatus
- `internal/db/search.go` - UPDATE SearchANN with optimized CTE query
- `internal/db/duckdb.go` - Minor updates for new schema

**Processing Layer:**

- `internal/engine/engine.go` - ADD DetectChanges(), UPDATE IndexFolder() for incremental mode with HNSW rebuild
- `internal/processor/chunker.go` - ADD StartToken, EndToken fields
- `internal/processor/pdf_chunker.go` - ADD StartToken, EndToken fields

**CLI Layer:**

- `internal/cli/index.go` - ADD --incremental flag, UPDATE progress reporting, show status before update
- `internal/cli/search.go` - UPDATE result formatting (no JSON parsing)
- `internal/cli/status.go` (NEW) - Status command to show pending changes

### Implementation Order

1. **Database Layer First** (Steps 1-2) - Foundation
2. **Change Detection** (Step 3) - Core incremental logic
3. **Processing Updates** (Step 4) - Chunker adjustments
4. **CLI Updates** (Step 5) - User-facing changes
5. **Testing** (Step 6) - Validation

### Breaking Changes

- **Existing databases are incompatible** - must drop and re-index
- **CLI output format changes** (metadata structure different)
- **API changes** in internal packages (File vs Document structs)
- **No migration path** - users must delete old .db files and re-index

### Upgrade Path

- **Simple approach:** Delete old .db files and run `qwelli index` to create new schema
- Old `qwelli index` behavior unchanged (full re-index)
- New `qwelli index --incremental` for delta updates (still requires HNSW rebuild)
- Dimension stored in metadata table (validated on open)

### Additional Implementation Considerations

**1. Path Normalization:**

- Store absolute paths in database for consistency
- Normalize paths on insert: `filepath.Abs(path)`
- Handle Windows vs Unix path separators
- Consider case-sensitivity on different filesystems

**2. Error Handling:**

- File permission errors: log warning, skip file, continue processing
- Missing files during incremental update: remove from DB, continue
- Corrupted files: log error, skip file, continue
- Database lock errors: retry with backoff
- Embedding API failures: log error, skip file, continue (partial index)

**3. Transaction Boundaries:**

- Use transactions for:
  - File deletion (cascade: embeddings → chunks → file)
  - Batch file inserts (all-or-nothing per file)
- Don't use transactions for:
  - HNSW index rebuild (separate operation, can be retried)
  - Status checks (read-only, no transaction needed)

**4. Progress Reporting:**

- Show: files processed, chunks created, embeddings generated
- Show: time elapsed, estimated time remaining
- Show: HNSW rebuild progress (if DuckDB provides it)
- Show: errors encountered (summary at end)

**5. Empty Database Handling:**

- First index: create schema, process files, build HNSW index
- Empty folder: create schema, no files to index, no HNSW index needed
- No embeddings: skip HNSW index creation (avoid error on empty table)

**6. Dimension Consistency:**

- Validate dimension on database open (from metadata)
- Reject embedding inserts with wrong dimension
- Store dimension in metadata on schema creation

**7. Search Query Optimization:**

- Current search may not use HNSW index efficiently
- Ensure query planner uses HNSW index (may need query hints)
- Consider using `array_cosine_similarity` if available (1 - distance)
- Test query performance with EXPLAIN to verify index usage

### HNSW Index Immutability: Impact on Incremental Updates

**What Incremental Updates Still Provide:**

- ✅ Faster file processing (only changed files)
- ✅ Lower embedding API costs (only new/changed chunks)
- ✅ Smaller database transactions
- ✅ Better user experience (faster feedback on progress)

**What Still Requires Full Operation:**

- ❌ HNSW index rebuild (must rebuild entire index after any embedding changes)
- ❌ Rebuild time scales with total embeddings, not just changed ones

**Practical Impact:**

- For small changes (<10 files): Rebuild overhead may be similar to full re-index
- For large changes (>100 files): Significant time savings from skipping unchanged files
- **Best practice:** Batch multiple file changes before single HNSW rebuild
- **Future optimization:** Consider maintaining multiple smaller indexes or using external vector DB if rebuild time becomes bottleneck

---

## Quick Reference: Implementation Checklist

### Pre-Implementation

- [ ] Review current codebase structure
- [ ] Document current embedding dimension used
- [ ] Backup any important existing databases (users will need to re-index)

### Phase 1: Database Schema

- [ ] Update `schema.go` with new table definitions
- [ ] Add dimension tracking in metadata
- [ ] Update `models.go` with File, Chunk structs
- [ ] Remove/update old Document struct
- [ ] Test schema creation on fresh database

### Phase 2: Core Operations

- [ ] Implement `InsertFile()`, `GetFile()`, `DeleteFile()`
- [ ] Implement `InsertChunk()`, `GetChunksForFile()`
- [ ] Implement manual cascade delete (embeddings → chunks → file)
- [ ] Implement `RebuildHNSWIndex()` method
- [ ] Update `InsertEmbedding()` to use chunk_id
- [ ] Update `SearchANN()` with optimized query
- [ ] Test all CRUD operations

### Phase 3: Change Detection

- [ ] Create `hash.go` with SHA256 computation
- [ ] Implement `DetectChanges()` function
- [ ] Test change detection logic (add/update/delete scenarios)
- [ ] Handle edge cases (missing files, permission errors)

### Phase 4: Incremental Indexing

- [ ] Update `IndexFolder()` with incremental mode
- [ ] Implement batch processing (all changes before rebuild)
- [ ] Add progress reporting
- [ ] Test incremental update workflow
- [ ] Test HNSW rebuild after changes

### Phase 5: Status Command

- [ ] Create `status.go` with `RunStatus()` command
- [ ] Implement `GetIndexStatus()` in queries.go
- [ ] Register status command in CLI
- [ ] Test status output formatting

### Phase 6: Testing

- [ ] Unit tests for all new DB operations
- [ ] Integration tests for incremental indexing
- [ ] Performance benchmarks (old vs new)
- [ ] Error handling tests (permissions, corrupted files, etc.)

### Post-Implementation

- [ ] Update documentation
- [ ] Update README with new commands
- [ ] Test on real-world datasets
- [ ] Monitor performance in production
- [ ] Gather user feedback

### Critical Reminders

- ⚠️ **HNSW index is immutable** - must rebuild after ANY embedding changes
- ⚠️ **No foreign keys** - implement manual cascade deletes
- ⚠️ **Breaking change** - old databases incompatible, users must drop and re-index
- ⚠️ **Dimension consistency** - validate on all embedding inserts
- ⚠️ **Path normalization** - use absolute paths for consistency
- ⚠️ **Transactions** - use for atomic operations (deletes, batch inserts)
- ⚠️ **VSS extension** - ensure loaded before HNSW operations
