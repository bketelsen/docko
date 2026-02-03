-- +goose Up

-- Add min_word_count column to ai_settings
-- When > 0, documents with fewer words in extracted text will be quarantined
-- Default 0 means no minimum enforced (feature disabled)
ALTER TABLE ai_settings ADD COLUMN min_word_count INTEGER NOT NULL DEFAULT 0;

-- +goose Down

ALTER TABLE ai_settings DROP COLUMN min_word_count;
