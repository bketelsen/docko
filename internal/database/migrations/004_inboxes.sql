-- +goose Up

-- Duplicate handling strategies for inbox sources
CREATE TYPE duplicate_action AS ENUM ('delete', 'rename', 'skip');

-- Inboxes table: configured directories to watch for new documents
CREATE TABLE inboxes (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    path VARCHAR(1024) NOT NULL UNIQUE,
    name VARCHAR(255) NOT NULL,
    enabled BOOLEAN NOT NULL DEFAULT true,
    error_path VARCHAR(1024),
    duplicate_action duplicate_action NOT NULL DEFAULT 'delete',
    last_scan_at TIMESTAMPTZ,
    last_error TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Inbox events table: log of files processed from each inbox
CREATE TABLE inbox_events (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    inbox_id UUID NOT NULL REFERENCES inboxes(id) ON DELETE CASCADE,
    filename VARCHAR(255) NOT NULL,
    action VARCHAR(50) NOT NULL,  -- imported, duplicate, error, moved_to_error
    document_id UUID REFERENCES documents(id) ON DELETE SET NULL,
    error_message TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Index for fetching recent inbox activity
CREATE INDEX idx_inbox_events_inbox_created ON inbox_events (inbox_id, created_at DESC);

-- +goose Down
DROP TABLE IF EXISTS inbox_events;
DROP TABLE IF EXISTS inboxes;
DROP TYPE IF EXISTS duplicate_action;
