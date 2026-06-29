-- +goose Up

-- The "Recently Added" sort now uses the raw timestamp with an id tiebreak
-- instead of datetime(), so the indexes become plain composite (col, id) to
-- cover it. Timestamps were already normalized to space-format by
-- 20260316000000_normalize_timestamps, so raw-string comparison is safe.

DROP INDEX IF EXISTS album_created_at;
CREATE INDEX album_created_at ON album(created_at, id);
DROP INDEX IF EXISTS album_updated_at;
CREATE INDEX album_updated_at ON album(updated_at, id);

DROP INDEX IF EXISTS media_file_created_at;
CREATE INDEX media_file_created_at ON media_file(created_at, id);
DROP INDEX IF EXISTS media_file_updated_at;
CREATE INDEX media_file_updated_at ON media_file(updated_at, id);

-- +goose Down

DROP INDEX IF EXISTS album_created_at;
CREATE INDEX album_created_at ON album(datetime(created_at));
DROP INDEX IF EXISTS album_updated_at;
CREATE INDEX album_updated_at ON album(datetime(updated_at));

DROP INDEX IF EXISTS media_file_created_at;
CREATE INDEX media_file_created_at ON media_file(created_at);
DROP INDEX IF EXISTS media_file_updated_at;
CREATE INDEX media_file_updated_at ON media_file(updated_at);
