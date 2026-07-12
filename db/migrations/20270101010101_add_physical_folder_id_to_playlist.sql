-- +goose Up
ALTER TABLE playlist ADD COLUMN physical_folder_id VARCHAR DEFAULT '' NOT NULL;
CREATE INDEX IF NOT EXISTS idx_playlist_physical_folder_id ON playlist (physical_folder_id);

-- +goose Down
DROP INDEX IF EXISTS idx_playlist_physical_folder_id;
ALTER TABLE playlist DROP COLUMN physical_folder_id;
