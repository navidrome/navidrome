-- +goose Up
ALTER TABLE album ADD COLUMN avg_rating REAL NOT NULL DEFAULT 0;
ALTER TABLE media_file ADD COLUMN avg_rating REAL NOT NULL DEFAULT 0;
ALTER TABLE artist ADD COLUMN avg_rating REAL NOT NULL DEFAULT 0;

CREATE INDEX IF NOT EXISTS album_avg_rating ON album(avg_rating);
CREATE INDEX IF NOT EXISTS media_file_avg_rating ON media_file(avg_rating);
CREATE INDEX IF NOT EXISTS artist_avg_rating ON artist(avg_rating);

-- +goose Down
DROP INDEX IF EXISTS artist_avg_rating;
DROP INDEX IF EXISTS media_file_avg_rating;
DROP INDEX IF EXISTS album_avg_rating;

ALTER TABLE artist DROP COLUMN avg_rating;
ALTER TABLE media_file DROP COLUMN avg_rating;
ALTER TABLE album DROP COLUMN avg_rating;
