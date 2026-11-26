-- +goose Up
-- +goose StatementBegin

-- The oldest file birth time moves from album.created_at to birth_time, and created_at
-- becomes the first scan time of the album's tracks, matching media_file.created_at.
ALTER TABLE album ADD COLUMN birth_time datetime NOT NULL DEFAULT '0000-00-00 00:00:00';
UPDATE album SET birth_time = created_at;

UPDATE album SET created_at = (
    SELECT MIN(created_at) FROM media_file
    WHERE media_file.album_id = album.id
      AND created_at IS NOT NULL AND created_at != ''
      AND created_at NOT LIKE '0001-%' AND created_at NOT LIKE '0000-%')
WHERE EXISTS (
    SELECT 1 FROM media_file
    WHERE media_file.album_id = album.id
      AND created_at IS NOT NULL AND created_at != ''
      AND created_at NOT LIKE '0001-%' AND created_at NOT LIKE '0000-%');

CREATE INDEX IF NOT EXISTS album_birth_time ON album(birth_time, id);
CREATE INDEX IF NOT EXISTS media_file_birth_time ON media_file(birth_time, id);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP INDEX IF EXISTS media_file_birth_time;
DROP INDEX IF EXISTS album_birth_time;
UPDATE album SET created_at = birth_time;
ALTER TABLE album DROP COLUMN birth_time;
-- +goose StatementEnd
