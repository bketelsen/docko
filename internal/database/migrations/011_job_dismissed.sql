-- +goose Up
ALTER TYPE job_status ADD VALUE 'dismissed' AFTER 'failed';

-- +goose Down
-- PostgreSQL does not support removing enum values
-- Would require recreating the enum and updating all references
-- This is intentionally left as a no-op for safety
