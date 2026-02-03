-- +goose Up

-- Add generated tsvector column for full-text search
-- Automatically updates when original_filename or text_content changes
ALTER TABLE documents
    ADD COLUMN search_vector tsvector
    GENERATED ALWAYS AS (
        to_tsvector('english',
            coalesce(original_filename, '') || ' ' ||
            coalesce(text_content, '')
        )
    ) STORED;

-- GIN index for fast full-text search
CREATE INDEX idx_documents_search ON documents USING GIN (search_vector);

-- +goose Down

DROP INDEX IF EXISTS idx_documents_search;

ALTER TABLE documents DROP COLUMN IF EXISTS search_vector;
