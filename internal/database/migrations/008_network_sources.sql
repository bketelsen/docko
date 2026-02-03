-- +goose Up

-- Protocol types for network sources
CREATE TYPE network_protocol AS ENUM ('smb', 'nfs');

-- Post-import action for processed files
CREATE TYPE post_import_action AS ENUM ('leave', 'delete', 'move');

-- Network sources table: configured network shares to watch for documents
CREATE TABLE network_sources (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(255) NOT NULL,
    protocol network_protocol NOT NULL,
    host VARCHAR(255) NOT NULL,
    share_path VARCHAR(1024) NOT NULL,
    username VARCHAR(255),
    password_encrypted TEXT,
    enabled BOOLEAN NOT NULL DEFAULT false,
    continuous_sync BOOLEAN NOT NULL DEFAULT true,
    post_import_action post_import_action NOT NULL DEFAULT 'leave',
    move_subfolder VARCHAR(255) DEFAULT 'imported',
    duplicate_action duplicate_action NOT NULL DEFAULT 'delete',
    batch_size INTEGER NOT NULL DEFAULT 100,
    connection_state VARCHAR(50) DEFAULT 'unknown',
    consecutive_failures INTEGER NOT NULL DEFAULT 0,
    last_sync_at TIMESTAMPTZ,
    last_error TEXT,
    files_imported INTEGER NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Network source events table: log of files processed from each source
CREATE TABLE network_source_events (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    source_id UUID NOT NULL REFERENCES network_sources(id) ON DELETE CASCADE,
    filename VARCHAR(255) NOT NULL,
    remote_path VARCHAR(1024) NOT NULL,
    action VARCHAR(50) NOT NULL,
    document_id UUID REFERENCES documents(id) ON DELETE SET NULL,
    error_message TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Index for fetching recent network source activity
CREATE INDEX idx_network_source_events_source_created ON network_source_events (source_id, created_at DESC);

-- +goose Down
DROP INDEX IF EXISTS idx_network_source_events_source_created;
DROP TABLE IF EXISTS network_source_events;
DROP TABLE IF EXISTS network_sources;
DROP TYPE IF EXISTS post_import_action;
DROP TYPE IF EXISTS network_protocol;
