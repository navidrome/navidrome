-- +goose Up

-- Change order_*/sort_* column collation from NOCASE to NATURALSORT.
-- This way bare ORDER BY on these columns automatically uses natural sorting,
-- without needing explicit COLLATE NATURALSORT in every query.
PRAGMA writable_schema = ON;
UPDATE sqlite_master
SET sql = replace(sql, 'collate NOCASE', 'collate NATURALSORT')
WHERE type = 'table' AND name IN ('artist', 'album', 'media_file', 'playlist', 'radio');
PRAGMA writable_schema = OFF;

-- Recreate indexes on order_* and sort expression fields to use NATURALSORT collation.
-- This enables natural number ordering (e.g., "Album 2" before "Album 10").

-- Artist indexes
drop index if exists artist_order_artist_name;
create index artist_order_artist_name
    on artist (order_artist_name collate NATURALSORT);

drop index if exists artist_sort_name;
create index artist_sort_name
    on artist (coalesce(nullif(sort_artist_name,''),order_artist_name) collate NATURALSORT);

-- Album indexes
drop index if exists album_order_album_name;
create index album_order_album_name
    on album (order_album_name collate NATURALSORT);

drop index if exists album_order_album_artist_name;
create index album_order_album_artist_name
    on album (order_album_artist_name collate NATURALSORT);

drop index if exists album_alphabetical_by_artist;
create index album_alphabetical_by_artist
    on album (compilation, order_album_artist_name collate NATURALSORT, order_album_name collate NATURALSORT);

drop index if exists album_sort_name;
create index album_sort_name
    on album (coalesce(nullif(sort_album_name,''),order_album_name) collate NATURALSORT);

drop index if exists album_sort_album_artist_name;
create index album_sort_album_artist_name
    on album (coalesce(nullif(sort_album_artist_name,''),order_album_artist_name) collate NATURALSORT);

-- Media file indexes
drop index if exists media_file_order_title;
create index media_file_order_title
    on media_file (order_title collate NATURALSORT);

drop index if exists media_file_order_album_name;
create index media_file_order_album_name
    on media_file (order_album_name collate NATURALSORT);

drop index if exists media_file_order_artist_name;
create index media_file_order_artist_name
    on media_file (order_artist_name collate NATURALSORT);

drop index if exists media_file_sort_title;
create index media_file_sort_title
    on media_file (coalesce(nullif(sort_title,''),order_title) collate NATURALSORT);

drop index if exists media_file_sort_artist_name;
create index media_file_sort_artist_name
    on media_file (coalesce(nullif(sort_artist_name,''),order_artist_name) collate NATURALSORT);

drop index if exists media_file_sort_album_name;
create index media_file_sort_album_name
    on media_file (coalesce(nullif(sort_album_name,''),order_album_name) collate NATURALSORT);

-- Playlist and radio indexes: recreate to match new NATURALSORT column collation
drop index if exists playlist_name;
create index playlist_name
    on playlist (name collate NATURALSORT);

drop index if exists radio_name;
create index radio_name
    on radio (name collate NATURALSORT);

-- +goose Down

-- Restore NOCASE column collation
PRAGMA writable_schema = ON;
UPDATE sqlite_master
SET sql = replace(sql, 'collate NATURALSORT', 'collate NOCASE')
WHERE type = 'table' AND name IN ('artist', 'album', 'media_file', 'playlist', 'radio');
PRAGMA writable_schema = OFF;

-- Restore NOCASE collation indexes

-- Artist indexes
drop index if exists artist_order_artist_name;
create index artist_order_artist_name
    on artist (order_artist_name);

drop index if exists artist_sort_name;
create index artist_sort_name
    on artist (coalesce(nullif(sort_artist_name,''),order_artist_name) collate NOCASE);

-- Album indexes
drop index if exists album_order_album_name;
create index album_order_album_name
    on album (order_album_name);

drop index if exists album_order_album_artist_name;
create index album_order_album_artist_name
    on album (order_album_artist_name);

drop index if exists album_alphabetical_by_artist;
create index album_alphabetical_by_artist
    on album (compilation, order_album_artist_name, order_album_name);

drop index if exists album_sort_name;
create index album_sort_name
    on album (coalesce(nullif(sort_album_name,''),order_album_name) collate NOCASE);

drop index if exists album_sort_album_artist_name;
create index album_sort_album_artist_name
    on album (coalesce(nullif(sort_album_artist_name,''),order_album_artist_name) collate NOCASE);

-- Media file indexes
drop index if exists media_file_order_title;
create index media_file_order_title
    on media_file (order_title);

drop index if exists media_file_order_album_name;
create index media_file_order_album_name
    on media_file (order_album_name);

drop index if exists media_file_order_artist_name;
create index media_file_order_artist_name
    on media_file (order_artist_name);

drop index if exists media_file_sort_title;
create index media_file_sort_title
    on media_file (coalesce(nullif(sort_title,''),order_title) collate NOCASE);

drop index if exists media_file_sort_artist_name;
create index media_file_sort_artist_name
    on media_file (coalesce(nullif(sort_artist_name,''),order_artist_name) collate NOCASE);

drop index if exists media_file_sort_album_name;
create index media_file_sort_album_name
    on media_file (coalesce(nullif(sort_album_name,''),order_album_name) collate NOCASE);

-- Restore playlist and radio indexes
drop index if exists playlist_name;
create index playlist_name
    on playlist (name);

drop index if exists radio_name;
create index radio_name
    on radio (name);
