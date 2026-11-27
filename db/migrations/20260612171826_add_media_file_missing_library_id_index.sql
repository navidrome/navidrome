-- +goose Up
-- Covering index for the rowid-only pagination query used by search3 with an empty query
-- (full library sync). It must cover both `missing` and `library_id` so SQLite never touches
-- the (wide) media_file rows while skipping over large offsets.
-- Replaces media_file_missing: the composite serves all `missing = ?` lookups via its prefix.
create index if not exists media_file_missing_library_id
    on media_file(missing, library_id);
drop index if exists media_file_missing;

-- +goose Down
create index if not exists media_file_missing
    on media_file(missing);
drop index if exists media_file_missing_library_id;
