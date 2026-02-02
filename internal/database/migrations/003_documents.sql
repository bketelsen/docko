-- +goose Up

-- Documents table: stores metadata about uploaded documents
CREATE TABLE documents (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    original_filename VARCHAR(255) NOT NULL,
    content_hash VARCHAR(64) NOT NULL,
    file_size BIGINT NOT NULL,
    page_count INT,
    pdf_title TEXT,
    pdf_author TEXT,
    pdf_created_at TIMESTAMPTZ,
    document_date TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT documents_content_hash_unique UNIQUE (content_hash)
);

-- Job status enum for queue processing
CREATE TYPE job_status AS ENUM ('pending', 'processing', 'completed', 'failed');

-- Jobs table: queue for background processing with SKIP LOCKED support
CREATE TABLE jobs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    queue_name VARCHAR(50) NOT NULL DEFAULT 'default',
    job_type VARCHAR(50) NOT NULL,
    payload JSONB NOT NULL,
    status job_status NOT NULL DEFAULT 'pending',
    attempt INT NOT NULL DEFAULT 0,
    max_attempts INT NOT NULL DEFAULT 3,
    scheduled_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    visible_until TIMESTAMPTZ,
    started_at TIMESTAMPTZ,
    completed_at TIMESTAMPTZ,
    last_error TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Index for efficient job dequeuing with SKIP LOCKED
CREATE INDEX idx_jobs_dequeue ON jobs (queue_name, status, scheduled_at, created_at)
    WHERE status IN ('pending', 'processing');

-- Document events table: audit trail for document processing
CREATE TABLE document_events (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    document_id UUID NOT NULL REFERENCES documents(id) ON DELETE CASCADE,
    event_type VARCHAR(50) NOT NULL,
    payload JSONB,
    error_message TEXT,
    duration_ms INT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Index for fetching document history
CREATE INDEX idx_document_events_document_created ON document_events (document_id, created_at DESC);

-- Tags table: for document categorization
CREATE TABLE tags (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(100) NOT NULL UNIQUE,
    color VARCHAR(7),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Correspondents table: people/organizations associated with documents
CREATE TABLE correspondents (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(255) NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Junction table: many-to-many relationship between documents and tags
CREATE TABLE document_tags (
    document_id UUID NOT NULL REFERENCES documents(id) ON DELETE CASCADE,
    tag_id UUID NOT NULL REFERENCES tags(id) ON DELETE CASCADE,
    PRIMARY KEY (document_id, tag_id)
);

-- Junction table: one correspondent per document
CREATE TABLE document_correspondents (
    document_id UUID PRIMARY KEY REFERENCES documents(id) ON DELETE CASCADE,
    correspondent_id UUID NOT NULL REFERENCES correspondents(id) ON DELETE CASCADE
);

-- +goose Down
DROP TABLE IF EXISTS document_correspondents;
DROP TABLE IF EXISTS document_tags;
DROP TABLE IF EXISTS correspondents;
DROP TABLE IF EXISTS tags;
DROP TABLE IF EXISTS document_events;
DROP TABLE IF EXISTS jobs;
DROP TYPE IF EXISTS job_status;
DROP TABLE IF EXISTS documents;
