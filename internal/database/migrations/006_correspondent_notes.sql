-- +goose Up
ALTER TABLE correspondents ADD COLUMN notes TEXT;

-- +goose Down
ALTER TABLE correspondents DROP COLUMN notes;
