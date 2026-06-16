-- +goose Up
-- Covering index so the empty-query artist search Phase 1 can resolve the per-user
-- EXISTS(library_artist ... where artist_id = ?) with an index seek keyed on artist_id,
-- instead of probing the (library_id, artist_id) UNIQUE autoindex by library.
CREATE INDEX IF NOT EXISTS idx_library_artist_artist_id_library_id
    ON library_artist (artist_id, library_id);

-- +goose Down
DROP INDEX IF EXISTS idx_library_artist_artist_id_library_id;
