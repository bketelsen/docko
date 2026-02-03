-- +goose Up

-- Processing status enum for document processing state
CREATE TYPE processing_status AS ENUM ('pending', 'processing', 'completed', 'failed');

-- Add processing-related columns to documents table
ALTER TABLE documents
    ADD COLUMN processing_status processing_status NOT NULL DEFAULT 'pending',
    ADD COLUMN text_content TEXT,
    ADD COLUMN thumbnail_generated BOOLEAN NOT NULL DEFAULT false,
    ADD COLUMN processing_error TEXT,
    ADD COLUMN processed_at TIMESTAMPTZ;

-- Index on processing_status for efficient queue queries
CREATE INDEX idx_documents_processing_status ON documents (processing_status)
    WHERE processing_status IN ('pending', 'processing');

-- +goose Down

DROP INDEX IF EXISTS idx_documents_processing_status;

ALTER TABLE documents
    DROP COLUMN IF EXISTS processed_at,
    DROP COLUMN IF EXISTS processing_error,
    DROP COLUMN IF EXISTS thumbnail_generated,
    DROP COLUMN IF EXISTS text_content,
    DROP COLUMN IF EXISTS processing_status;

DROP TYPE IF EXISTS processing_status;
