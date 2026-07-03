-- +goose Up
-- +goose StatementBegin

-- Composite indexes matching the media_file sort mappings for album, artist and
-- albumArtist. Without them, SQLite cannot satisfy the multi-column ORDER BY and
-- falls back to a full scan + temp B-tree sort of the whole table (including all
-- its large columns) even for a small LIMIT.
create index if not exists media_file_album_sort
	on media_file(order_album_name, album_id, disc_number, track_number, order_artist_name, title);
create index if not exists media_file_artist_sort
	on media_file(order_artist_name, order_album_name, release_date, disc_number, track_number);
create index if not exists media_file_album_artist_sort
	on media_file(order_album_artist_name, order_album_name, release_date, disc_number, track_number);

-- These two are strict prefixes of the composites above, so they are redundant now.
drop index if exists media_file_order_album_name;
drop index if exists media_file_order_artist_name;

-- No query filters or sorts on these columns: birth_time is only read in Go code;
-- artist/album_artist conditions go through the media_file_artists table.
drop index if exists media_file_birth_time;
drop index if exists media_file_artist;
drop index if exists media_file_album_artist;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
drop index if exists media_file_album_sort;
drop index if exists media_file_artist_sort;
drop index if exists media_file_album_artist_sort;

create index if not exists media_file_order_album_name
	on media_file(order_album_name);
create index if not exists media_file_order_artist_name
	on media_file(order_artist_name);
create index if not exists media_file_birth_time
	on media_file(birth_time);
create index if not exists media_file_artist
	on media_file(artist);
create index if not exists media_file_album_artist
	on media_file(album_artist);
-- +goose StatementEnd
