-- +goose Up

-- Fix 1: library_artist.artist_id has contradictory "not null default null".
-- Recreate the table with the corrected column definition (remove "default null").
CREATE TABLE library_artist_fixed (
    library_id integer not null default 1
        references library(id)
            on delete cascade,
    artist_id varchar not null
        references artist(id)
            on delete cascade,
    stats text default '{}',
    constraint library_artist_ux
        unique (library_id, artist_id)
);
INSERT INTO library_artist_fixed (library_id, artist_id, stats)
    SELECT library_id, artist_id, stats FROM library_artist;
DROP TABLE library_artist;
ALTER TABLE library_artist_fixed RENAME TO library_artist;

-- Fix 2: scrobble_buffer unique constraint has duplicate user_id column.
-- Recreate the table with the corrected constraint (remove duplicate user_id).
DROP INDEX IF EXISTS scrobble_buffer_id_ix;
CREATE TABLE scrobble_buffer_fixed (
    id varchar not null default '',
    user_id varchar not null
        constraint scrobble_buffer_user_id_fk
            references user
                on update cascade on delete cascade,
    service varchar not null,
    media_file_id varchar not null
        constraint scrobble_buffer_media_file_id_fk
            references media_file
                on update cascade on delete cascade,
    play_time datetime not null,
    enqueue_time datetime not null default current_timestamp,
    constraint scrobble_buffer_pk
        unique (user_id, service, media_file_id, play_time)
);
INSERT INTO scrobble_buffer_fixed (id, user_id, service, media_file_id, play_time, enqueue_time)
    SELECT id, user_id, service, media_file_id, play_time, enqueue_time FROM scrobble_buffer;
DROP TABLE scrobble_buffer;
ALTER TABLE scrobble_buffer_fixed RENAME TO scrobble_buffer;
CREATE UNIQUE INDEX scrobble_buffer_id_ix ON scrobble_buffer (id);

-- Fix 3: goose_db_version.tstamp uses formula default.
-- Recreate with standard DEFAULT CURRENT_TIMESTAMP.
CREATE TABLE goose_db_version_fixed (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    version_id INTEGER NOT NULL DEFAULT 0,
    is_applied INTEGER NOT NULL DEFAULT 0,
    tstamp TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);
INSERT INTO goose_db_version_fixed (id, version_id, is_applied, tstamp)
    SELECT id, version_id, is_applied, tstamp FROM goose_db_version;
DROP TABLE goose_db_version;
ALTER TABLE goose_db_version_fixed RENAME TO goose_db_version;

-- +goose Down

-- Revert Fix 3: Restore goose_db_version with original formula default
CREATE TABLE goose_db_version_original (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    version_id INTEGER NOT NULL DEFAULT 0,
    is_applied INTEGER NOT NULL DEFAULT 0,
    tstamp TIMESTAMP DEFAULT (datetime('now'))
);
INSERT INTO goose_db_version_original (id, version_id, is_applied, tstamp)
    SELECT id, version_id, is_applied, tstamp FROM goose_db_version;
DROP TABLE goose_db_version;
ALTER TABLE goose_db_version_original RENAME TO goose_db_version;

-- Revert Fix 2: Restore scrobble_buffer with duplicate user_id in constraint
DROP INDEX IF EXISTS scrobble_buffer_id_ix;
CREATE TABLE scrobble_buffer_original (
    id varchar not null default '',
    user_id varchar not null
        constraint scrobble_buffer_user_id_fk
            references user
                on update cascade on delete cascade,
    service varchar not null,
    media_file_id varchar not null
        constraint scrobble_buffer_media_file_id_fk
            references media_file
                on update cascade on delete cascade,
    play_time datetime not null,
    enqueue_time datetime not null default current_timestamp,
    constraint scrobble_buffer_pk
        unique (user_id, service, media_file_id, play_time, user_id)
);
INSERT INTO scrobble_buffer_original (id, user_id, service, media_file_id, play_time, enqueue_time)
    SELECT id, user_id, service, media_file_id, play_time, enqueue_time FROM scrobble_buffer;
DROP TABLE scrobble_buffer;
ALTER TABLE scrobble_buffer_original RENAME TO scrobble_buffer;
CREATE UNIQUE INDEX scrobble_buffer_id_ix ON scrobble_buffer (id);

-- Revert Fix 1: Restore library_artist with "not null default null"
CREATE TABLE library_artist_original (
    library_id integer not null default 1
        references library(id)
            on delete cascade,
    artist_id varchar not null default null
        references artist(id)
            on delete cascade,
    stats text default '{}',
    constraint library_artist_ux
        unique (library_id, artist_id)
);
INSERT INTO library_artist_original (library_id, artist_id, stats)
    SELECT library_id, artist_id, stats FROM library_artist;
DROP TABLE library_artist;
ALTER TABLE library_artist_original RENAME TO library_artist;
