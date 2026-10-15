-- +goose Up
ALTER TABLE item_artwork ADD COLUMN trace jsonb NOT NULL DEFAULT '[]';
ALTER TABLE item_artwork ADD COLUMN last_failure jsonb NOT NULL DEFAULT '[]';
ALTER TABLE artwork_queue ADD COLUMN trace jsonb NOT NULL DEFAULT '[]';

-- +goose Down
ALTER TABLE artwork_queue DROP COLUMN trace;
ALTER TABLE item_artwork DROP COLUMN last_failure;
ALTER TABLE item_artwork DROP COLUMN trace;
