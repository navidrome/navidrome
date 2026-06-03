ALTER TABLE playlist ADD COLUMN physical_folder_id VARCHAR DEFAULT '' NOT NULL;
CREATE INDEX IF NOT EXISTS idx_playlist_physical_folder_id ON playlist (physical_folder_id);
