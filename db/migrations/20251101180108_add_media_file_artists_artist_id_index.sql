-- +goose Up
-- Add index on artist_id for media_file_artists table to improve query performance
-- This index is crucial for the RefreshStats query in artistRepository
CREATE INDEX IF NOT EXISTS media_file_artists_artist_id ON media_file_artists(artist_id);

-- +goose Down
DROP INDEX IF EXISTS media_file_artists_artist_id;
