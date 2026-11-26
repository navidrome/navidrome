-- +goose Up
-- +goose StatementBegin

-- The oldest file birth time moves from album.created_at to birth_time, and created_at
-- becomes the first scan time of the album's tracks, matching media_file.created_at.
ALTER TABLE album ADD COLUMN birth_time datetime NOT NULL DEFAULT '0000-00-00 00:00:00';
UPDATE album SET birth_time = created_at;

-- One pass over media_file: walking media_file_album_id costs a random row read per track.
UPDATE album SET created_at = first_scan.created_at
FROM (SELECT album_id, MIN(created_at) AS created_at
      FROM media_file NOT INDEXED
      WHERE created_at IS NOT NULL AND created_at != ''
        AND created_at NOT LIKE '0001-%' AND created_at NOT LIKE '0000-%'
      GROUP BY album_id) AS first_scan
WHERE first_scan.album_id = album.id;

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
