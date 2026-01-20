-- Qwelli Database Initialization Script
-- This script sets up the PostgreSQL database with pgvector extension
-- and creates all necessary tables and indexes

-- Enable pgvector extension
CREATE EXTENSION IF NOT EXISTS vector;

-- Files table - metadata about indexed files
CREATE TABLE IF NOT EXISTS files (
    file_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    path TEXT NOT NULL UNIQUE,
    file_type TEXT NOT NULL,
    file_hash TEXT NOT NULL,
    size BIGINT NOT NULL,
    modified_at TIMESTAMPTZ NOT NULL,
    indexed_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT path_not_empty CHECK (path <> ''),
    CONSTRAINT file_type_not_empty CHECK (file_type <> ''),
    CONSTRAINT file_hash_not_empty CHECK (file_hash <> ''),
    CONSTRAINT size_positive CHECK (size >= 0)
);

-- Indexes for files table
CREATE INDEX IF NOT EXISTS idx_files_path ON files(path);
CREATE INDEX IF NOT EXISTS idx_files_hash ON files(file_hash);
CREATE INDEX IF NOT EXISTS idx_files_indexed_at ON files(indexed_at DESC);
CREATE INDEX IF NOT EXISTS idx_files_file_type ON files(file_type);

-- Add comment to files table
COMMENT ON TABLE files IS 'Stores metadata about indexed files';
COMMENT ON COLUMN files.file_id IS 'Unique identifier for the file';
COMMENT ON COLUMN files.path IS 'Full path to the file on disk';
COMMENT ON COLUMN files.file_type IS 'Type of file (pdf, text, markdown, etc.)';
COMMENT ON COLUMN files.file_hash IS 'SHA256 hash of file contents for change detection';
COMMENT ON COLUMN files.size IS 'Size of the file in bytes';
COMMENT ON COLUMN files.modified_at IS 'Last modification timestamp of the file';
COMMENT ON COLUMN files.indexed_at IS 'When the file was last indexed';

-- Chunks table - text/image segments from files
CREATE TABLE IF NOT EXISTS chunks (
    chunk_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    file_id UUID NOT NULL REFERENCES files(file_id) ON DELETE CASCADE,
    file_path TEXT NOT NULL,
    file_type TEXT NOT NULL,
    chunk_index INTEGER NOT NULL,
    total_chunks INTEGER NOT NULL,
    content TEXT NOT NULL,
    page_numbers INTEGER[],
    content_type TEXT NOT NULL CHECK (content_type IN ('text', 'image')),
    image_data BYTEA,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT chunk_index_valid CHECK (chunk_index >= 0),
    CONSTRAINT total_chunks_positive CHECK (total_chunks > 0),
    CONSTRAINT chunk_index_less_than_total CHECK (chunk_index < total_chunks)
);

-- Indexes for chunks table
CREATE INDEX IF NOT EXISTS idx_chunks_file_id ON chunks(file_id);
CREATE INDEX IF NOT EXISTS idx_chunks_content_type ON chunks(content_type);
CREATE INDEX IF NOT EXISTS idx_chunks_file_path ON chunks(file_path);
CREATE INDEX IF NOT EXISTS idx_chunks_created_at ON chunks(created_at DESC);

-- Add comments to chunks table
COMMENT ON TABLE chunks IS 'Stores chunked content from files for embedding';
COMMENT ON COLUMN chunks.chunk_id IS 'Unique identifier for the chunk';
COMMENT ON COLUMN chunks.file_id IS 'Reference to parent file';
COMMENT ON COLUMN chunks.file_path IS 'Denormalized file path for query performance';
COMMENT ON COLUMN chunks.file_type IS 'Denormalized file type for query performance';
COMMENT ON COLUMN chunks.chunk_index IS 'Index of this chunk within the file';
COMMENT ON COLUMN chunks.total_chunks IS 'Total number of chunks for this file';
COMMENT ON COLUMN chunks.content IS 'Text content of the chunk';
COMMENT ON COLUMN chunks.page_numbers IS 'PDF page numbers where this chunk appears';
COMMENT ON COLUMN chunks.content_type IS 'Type of content: text or image';
COMMENT ON COLUMN chunks.image_data IS 'Base64-encoded image data for image chunks';

-- Embeddings table - vector embeddings for chunks
-- Note: dimension is set to 1024 for voyage-multimodal-3 model
-- If using a different model, adjust the dimension accordingly
CREATE TABLE IF NOT EXISTS embeddings (
    chunk_id UUID PRIMARY KEY REFERENCES chunks(chunk_id) ON DELETE CASCADE,
    embedding vector(1024) NOT NULL
);

-- HNSW index for fast approximate nearest neighbor search
-- Using cosine distance metric for semantic similarity
CREATE INDEX IF NOT EXISTS idx_embeddings_hnsw ON embeddings
USING hnsw (embedding vector_cosine_ops)
WITH (m = 16, ef_construction = 64);

-- Add comments to embeddings table
COMMENT ON TABLE embeddings IS 'Stores vector embeddings for semantic search';
COMMENT ON COLUMN embeddings.chunk_id IS 'Reference to chunk';
COMMENT ON COLUMN embeddings.embedding IS 'Vector embedding (1024 dimensions for voyage-multimodal-3)';

-- Metadata table - key-value store for index metadata
CREATE TABLE IF NOT EXISTS metadata (
    key TEXT PRIMARY KEY,
    value TEXT NOT NULL,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Add comments to metadata table
COMMENT ON TABLE metadata IS 'Stores configuration and metadata about the index';

-- Initialize metadata
INSERT INTO metadata (key, value)
VALUES
    ('schema_version', '1'),
    ('dimension', '1024'),
    ('model', 'voyage-multimodal-3'),
    ('created_at', NOW()::TEXT)
ON CONFLICT (key) DO NOTHING;

-- Create function to update the updated_at timestamp
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- Add trigger to metadata table
DROP TRIGGER IF EXISTS update_metadata_updated_at ON metadata;
CREATE TRIGGER update_metadata_updated_at
    BEFORE UPDATE ON metadata
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

-- Grant permissions (adjust as needed for your security requirements)
-- GRANT SELECT, INSERT, UPDATE, DELETE ON ALL TABLES IN SCHEMA public TO qwelli;
-- GRANT USAGE, SELECT ON ALL SEQUENCES IN SCHEMA public TO qwelli;

-- Print success message
DO $$
BEGIN
    RAISE NOTICE 'Qwelli database initialized successfully!';
    RAISE NOTICE 'Schema version: 1';
    RAISE NOTICE 'Embedding dimension: 1024';
    RAISE NOTICE 'Model: voyage-multimodal-3';
END $$;
