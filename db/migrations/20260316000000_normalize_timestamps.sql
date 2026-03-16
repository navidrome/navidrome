-- +goose Up

-- Normalize T-format timestamps (RFC3339Nano with 'T' separator) to SQLite-compatible format.
-- SQLite uses string comparison for ORDER BY on TEXT columns, so 'T' (ASCII 84) > ' ' (ASCII 32)
-- causes T-format timestamps to sort after space-format ones, breaking "Recently Added" ordering.

UPDATE album SET created_at = replace(replace(created_at, 'T', ' '), 'Z', '+00:00') WHERE created_at LIKE '%T%';
UPDATE album SET updated_at = replace(replace(updated_at, 'T', ' '), 'Z', '+00:00') WHERE updated_at LIKE '%T%';
UPDATE album SET imported_at = replace(replace(imported_at, 'T', ' '), 'Z', '+00:00') WHERE imported_at LIKE '%T%';
UPDATE album SET external_info_updated_at = replace(replace(external_info_updated_at, 'T', ' '), 'Z', '+00:00') WHERE external_info_updated_at LIKE '%T%';

UPDATE media_file SET created_at = replace(replace(created_at, 'T', ' '), 'Z', '+00:00') WHERE created_at LIKE '%T%';
UPDATE media_file SET updated_at = replace(replace(updated_at, 'T', ' '), 'Z', '+00:00') WHERE updated_at LIKE '%T%';
UPDATE media_file SET birth_time = replace(replace(birth_time, 'T', ' '), 'Z', '+00:00') WHERE birth_time LIKE '%T%';

UPDATE artist SET created_at = replace(replace(created_at, 'T', ' '), 'Z', '+00:00') WHERE created_at LIKE '%T%';
UPDATE artist SET updated_at = replace(replace(updated_at, 'T', ' '), 'Z', '+00:00') WHERE updated_at LIKE '%T%';
UPDATE artist SET external_info_updated_at = replace(replace(external_info_updated_at, 'T', ' '), 'Z', '+00:00') WHERE external_info_updated_at LIKE '%T%';

UPDATE annotation SET play_date = replace(replace(play_date, 'T', ' '), 'Z', '+00:00') WHERE play_date LIKE '%T%';
UPDATE annotation SET starred_at = replace(replace(starred_at, 'T', ' '), 'Z', '+00:00') WHERE starred_at LIKE '%T%';
UPDATE annotation SET rated_at = replace(replace(rated_at, 'T', ' '), 'Z', '+00:00') WHERE rated_at LIKE '%T%';

UPDATE playlist SET created_at = replace(replace(created_at, 'T', ' '), 'Z', '+00:00') WHERE created_at LIKE '%T%';
UPDATE playlist SET updated_at = replace(replace(updated_at, 'T', ' '), 'Z', '+00:00') WHERE updated_at LIKE '%T%';
UPDATE playlist SET evaluated_at = replace(replace(evaluated_at, 'T', ' '), 'Z', '+00:00') WHERE evaluated_at LIKE '%T%';

UPDATE user SET created_at = replace(replace(created_at, 'T', ' '), 'Z', '+00:00') WHERE created_at LIKE '%T%';
UPDATE user SET updated_at = replace(replace(updated_at, 'T', ' '), 'Z', '+00:00') WHERE updated_at LIKE '%T%';
UPDATE user SET last_login_at = replace(replace(last_login_at, 'T', ' '), 'Z', '+00:00') WHERE last_login_at LIKE '%T%';
UPDATE user SET last_access_at = replace(replace(last_access_at, 'T', ' '), 'Z', '+00:00') WHERE last_access_at LIKE '%T%';

UPDATE player SET last_seen = replace(replace(last_seen, 'T', ' '), 'Z', '+00:00') WHERE last_seen LIKE '%T%';

UPDATE playqueue SET created_at = replace(replace(created_at, 'T', ' '), 'Z', '+00:00') WHERE created_at LIKE '%T%';
UPDATE playqueue SET updated_at = replace(replace(updated_at, 'T', ' '), 'Z', '+00:00') WHERE updated_at LIKE '%T%';

UPDATE bookmark SET created_at = replace(replace(created_at, 'T', ' '), 'Z', '+00:00') WHERE created_at LIKE '%T%';
UPDATE bookmark SET updated_at = replace(replace(updated_at, 'T', ' '), 'Z', '+00:00') WHERE updated_at LIKE '%T%';

UPDATE share SET created_at = replace(replace(created_at, 'T', ' '), 'Z', '+00:00') WHERE created_at LIKE '%T%';
UPDATE share SET updated_at = replace(replace(updated_at, 'T', ' '), 'Z', '+00:00') WHERE updated_at LIKE '%T%';
UPDATE share SET expires_at = replace(replace(expires_at, 'T', ' '), 'Z', '+00:00') WHERE expires_at LIKE '%T%';
UPDATE share SET last_visited_at = replace(replace(last_visited_at, 'T', ' '), 'Z', '+00:00') WHERE last_visited_at LIKE '%T%';

UPDATE radio SET created_at = replace(replace(created_at, 'T', ' '), 'Z', '+00:00') WHERE created_at LIKE '%T%';
UPDATE radio SET updated_at = replace(replace(updated_at, 'T', ' '), 'Z', '+00:00') WHERE updated_at LIKE '%T%';

UPDATE folder SET created_at = replace(replace(created_at, 'T', ' '), 'Z', '+00:00') WHERE created_at LIKE '%T%';
UPDATE folder SET updated_at = replace(replace(updated_at, 'T', ' '), 'Z', '+00:00') WHERE updated_at LIKE '%T%';
UPDATE folder SET images_updated_at = replace(replace(images_updated_at, 'T', ' '), 'Z', '+00:00') WHERE images_updated_at LIKE '%T%';

UPDATE library SET created_at = replace(replace(created_at, 'T', ' '), 'Z', '+00:00') WHERE created_at LIKE '%T%';
UPDATE library SET updated_at = replace(replace(updated_at, 'T', ' '), 'Z', '+00:00') WHERE updated_at LIKE '%T%';
UPDATE library SET last_scan_at = replace(replace(last_scan_at, 'T', ' '), 'Z', '+00:00') WHERE last_scan_at LIKE '%T%';
UPDATE library SET last_scan_started_at = replace(replace(last_scan_started_at, 'T', ' '), 'Z', '+00:00') WHERE last_scan_started_at LIKE '%T%';

UPDATE scrobble_buffer SET play_time = replace(replace(play_time, 'T', ' '), 'Z', '+00:00') WHERE play_time LIKE '%T%';
UPDATE scrobble_buffer SET enqueue_time = replace(replace(enqueue_time, 'T', ' '), 'Z', '+00:00') WHERE enqueue_time LIKE '%T%';

UPDATE plugin SET created_at = replace(replace(created_at, 'T', ' '), 'Z', '+00:00') WHERE created_at LIKE '%T%';
UPDATE plugin SET updated_at = replace(replace(updated_at, 'T', ' '), 'Z', '+00:00') WHERE updated_at LIKE '%T%';

-- Replace plain indexes with expression indexes for datetime()-based sorting
DROP INDEX IF EXISTS album_created_at;
CREATE INDEX album_created_at ON album(datetime(created_at));
DROP INDEX IF EXISTS album_updated_at;
CREATE INDEX album_updated_at ON album(datetime(updated_at));

-- +goose Down
DROP INDEX IF EXISTS album_created_at;
CREATE INDEX album_created_at ON album(created_at);
DROP INDEX IF EXISTS album_updated_at;
CREATE INDEX album_updated_at ON album(updated_at);
