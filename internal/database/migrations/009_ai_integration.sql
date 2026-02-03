-- +goose Up

-- Suggestion status enum for AI suggestion workflow
CREATE TYPE suggestion_status AS ENUM ('pending', 'accepted', 'rejected', 'auto_applied');

-- Suggestion type enum for categorizing AI suggestions
CREATE TYPE suggestion_type AS ENUM ('tag', 'correspondent');

-- AI settings table: singleton row for global AI configuration
CREATE TABLE ai_settings (
    id INTEGER PRIMARY KEY DEFAULT 1 CHECK (id = 1),
    preferred_provider VARCHAR(50),
    max_pages INTEGER NOT NULL DEFAULT 5,
    auto_process BOOLEAN NOT NULL DEFAULT false,
    auto_apply_threshold DECIMAL(3,2) NOT NULL DEFAULT 0.85,
    review_threshold DECIMAL(3,2) NOT NULL DEFAULT 0.50,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- AI suggestions table: per-document suggestions with confidence scores
CREATE TABLE ai_suggestions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    document_id UUID NOT NULL REFERENCES documents(id) ON DELETE CASCADE,
    job_id UUID REFERENCES jobs(id) ON DELETE SET NULL,
    suggestion_type suggestion_type NOT NULL,
    value VARCHAR(255) NOT NULL,
    confidence DECIMAL(3,2) NOT NULL,
    reasoning TEXT,
    is_new BOOLEAN NOT NULL DEFAULT false,
    status suggestion_status NOT NULL DEFAULT 'pending',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    resolved_at TIMESTAMPTZ,
    resolved_by VARCHAR(50)
);

-- AI usage table: track each AI request for cost monitoring
CREATE TABLE ai_usage (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    document_id UUID NOT NULL REFERENCES documents(id) ON DELETE CASCADE,
    job_id UUID REFERENCES jobs(id) ON DELETE SET NULL,
    provider VARCHAR(50) NOT NULL,
    model VARCHAR(100) NOT NULL,
    input_tokens INTEGER NOT NULL,
    output_tokens INTEGER NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Indexes for efficient queries
CREATE INDEX idx_ai_suggestions_document ON ai_suggestions (document_id);
CREATE INDEX idx_ai_suggestions_status ON ai_suggestions (status) WHERE status = 'pending';
CREATE INDEX idx_ai_usage_created ON ai_usage (created_at DESC);

-- Insert default settings row
INSERT INTO ai_settings (id) VALUES (1);

-- +goose Down

DELETE FROM ai_settings WHERE id = 1;
DROP INDEX IF EXISTS idx_ai_usage_created;
DROP INDEX IF EXISTS idx_ai_suggestions_status;
DROP INDEX IF EXISTS idx_ai_suggestions_document;
DROP TABLE IF EXISTS ai_usage;
DROP TABLE IF EXISTS ai_suggestions;
DROP TABLE IF EXISTS ai_settings;
DROP TYPE IF EXISTS suggestion_type;
DROP TYPE IF EXISTS suggestion_status;
