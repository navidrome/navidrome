-- +goose Up
-- +goose StatementBegin

-- Covering index for the title-sorted, library-scoped song listing:
--   WHERE missing = ? AND library_id = ? ORDER BY order_title LIMIT n OFFSET m
-- (Jellyfin clients page through the whole library this way; non-admin native and
-- Subsonic song lists produce the same shape.)
--
-- Without it, SQLite walks media_file_order_title and must fetch the table row for
-- every *skipped* entry just to evaluate the WHERE, so a deep page costs offset+limit
-- random row reads (seconds on cold spinning disks). With the filter columns in the
-- index the skip is index-only. `id` is included because the annotation/bookmark
-- LEFT JOINs run per candidate row and need the join key; without it each skipped
-- entry still triggers a row fetch.
create index if not exists media_file_missing_library_order_title
	on media_file(missing, library_id, order_title, id);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
drop index if exists media_file_missing_library_order_title;
-- +goose StatementEnd
