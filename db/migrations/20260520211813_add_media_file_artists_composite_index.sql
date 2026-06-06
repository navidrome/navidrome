-- +goose Up
CREATE INDEX IF NOT EXISTS media_file_artists_media_file_id_role
    ON media_file_artists (media_file_id, role);
DROP INDEX IF EXISTS media_file_artists_media_file_id;

-- +goose Down
CREATE INDEX IF NOT EXISTS media_file_artists_media_file_id
    ON media_file_artists (media_file_id);
DROP INDEX IF EXISTS media_file_artists_media_file_id_role;
