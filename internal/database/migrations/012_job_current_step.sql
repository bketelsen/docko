-- +goose Up
ALTER TABLE jobs ADD COLUMN current_step VARCHAR(50);
-- Allowed values: 'starting', 'extracting_text', 'running_ocr', 'generating_thumbnail', 'finalizing'
-- NULL indicates job hasn't started processing

-- +goose Down
ALTER TABLE jobs DROP COLUMN IF EXISTS current_step;
