-- +goose Up
-- COLLATE NOCASE covering indexes for the LIKE-based search fallback (CJK and
-- punctuation-only queries route to it; see persistence/sql_search_like.go
-- likeSearchColumns). Without these, a CJK search3 is a full scan of the wide
-- media_file table across 4 columns (~4s on 1M songs); with them SQLite scans the
-- narrow per-column indexes instead (~0.3s). Columns MUST match likeSearchColumns.
-- A plain LIKE uses these because Navidrome runs with case_sensitive_like = OFF.
CREATE INDEX IF NOT EXISTS idx_media_file_title_nocase ON media_file (title COLLATE NOCASE);
CREATE INDEX IF NOT EXISTS idx_media_file_album_nocase ON media_file (album COLLATE NOCASE);
CREATE INDEX IF NOT EXISTS idx_media_file_artist_nocase ON media_file (artist COLLATE NOCASE);
CREATE INDEX IF NOT EXISTS idx_media_file_album_artist_nocase ON media_file (album_artist COLLATE NOCASE);
CREATE INDEX IF NOT EXISTS idx_album_name_nocase ON album (name COLLATE NOCASE);
CREATE INDEX IF NOT EXISTS idx_album_album_artist_nocase ON album (album_artist COLLATE NOCASE);
CREATE INDEX IF NOT EXISTS idx_artist_name_nocase ON artist (name COLLATE NOCASE);

-- +goose Down
DROP INDEX IF EXISTS idx_media_file_title_nocase;
DROP INDEX IF EXISTS idx_media_file_album_nocase;
DROP INDEX IF EXISTS idx_media_file_artist_nocase;
DROP INDEX IF EXISTS idx_media_file_album_artist_nocase;
DROP INDEX IF EXISTS idx_album_name_nocase;
DROP INDEX IF EXISTS idx_album_album_artist_nocase;
DROP INDEX IF EXISTS idx_artist_name_nocase;
