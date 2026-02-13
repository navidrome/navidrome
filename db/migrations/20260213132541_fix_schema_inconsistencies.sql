-- +goose Up
-- +goose StatementBegin

-- Fix library_artist table: Remove contradictory 'default null' from 'not null' column
CREATE TABLE library_artist_new
(
    library_id integer NOT NULL DEFAULT 1
        REFERENCES library(id) ON DELETE CASCADE,
    artist_id varchar NOT NULL
        REFERENCES artist(id) ON DELETE CASCADE,
    stats text DEFAULT '{}',
    CONSTRAINT library_artist_ux UNIQUE (library_id, artist_id)
);

INSERT INTO library_artist_new (library_id, artist_id, stats)
SELECT library_id, artist_id, stats FROM library_artist;

DROP TABLE library_artist;

ALTER TABLE library_artist_new RENAME TO library_artist;

-- Fix scrobble_buffer table: Remove duplicate user_id from unique constraint
CREATE TABLE scrobble_buffer_new
(
    user_id varchar NOT NULL
        CONSTRAINT scrobble_buffer_user_id_fk
            REFERENCES user ON UPDATE CASCADE ON DELETE CASCADE,
    service varchar NOT NULL,
    media_file_id varchar NOT NULL
        CONSTRAINT scrobble_buffer_media_file_id_fk
            REFERENCES media_file ON UPDATE CASCADE ON DELETE CASCADE,
    play_time datetime NOT NULL,
    enqueue_time datetime NOT NULL DEFAULT CURRENT_TIMESTAMP,
    id varchar NOT NULL DEFAULT '',
    CONSTRAINT scrobble_buffer_pk UNIQUE (user_id, service, media_file_id, play_time)
);

INSERT INTO scrobble_buffer_new (user_id, service, media_file_id, play_time, enqueue_time, id)
SELECT user_id, service, media_file_id, play_time, enqueue_time, id FROM scrobble_buffer;

DROP TABLE scrobble_buffer;

ALTER TABLE scrobble_buffer_new RENAME TO scrobble_buffer;

CREATE UNIQUE INDEX scrobble_buffer_id_ix ON scrobble_buffer (id);

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin

-- Restore library_artist table with original schema (including the contradictory default null)
CREATE TABLE library_artist_new
(
    library_id integer NOT NULL DEFAULT 1
        REFERENCES library(id) ON DELETE CASCADE,
    artist_id varchar NOT NULL DEFAULT NULL
        REFERENCES artist(id) ON DELETE CASCADE,
    stats text DEFAULT '{}',
    CONSTRAINT library_artist_ux UNIQUE (library_id, artist_id)
);

INSERT INTO library_artist_new (library_id, artist_id, stats)
SELECT library_id, artist_id, stats FROM library_artist;

DROP TABLE library_artist;

ALTER TABLE library_artist_new RENAME TO library_artist;

-- Restore scrobble_buffer table with original schema (including duplicate user_id in constraint)
CREATE TABLE scrobble_buffer_new
(
    user_id varchar NOT NULL
        CONSTRAINT scrobble_buffer_user_id_fk
            REFERENCES user ON UPDATE CASCADE ON DELETE CASCADE,
    service varchar NOT NULL,
    media_file_id varchar NOT NULL
        CONSTRAINT scrobble_buffer_media_file_id_fk
            REFERENCES media_file ON UPDATE CASCADE ON DELETE CASCADE,
    play_time datetime NOT NULL,
    enqueue_time datetime NOT NULL DEFAULT CURRENT_TIMESTAMP,
    id varchar NOT NULL DEFAULT '',
    CONSTRAINT scrobble_buffer_pk UNIQUE (user_id, service, media_file_id, play_time, user_id)
);

INSERT INTO scrobble_buffer_new (user_id, service, media_file_id, play_time, enqueue_time, id)
SELECT user_id, service, media_file_id, play_time, enqueue_time, id FROM scrobble_buffer;

DROP TABLE scrobble_buffer;

ALTER TABLE scrobble_buffer_new RENAME TO scrobble_buffer;

CREATE UNIQUE INDEX scrobble_buffer_id_ix ON scrobble_buffer (id);

-- +goose StatementEnd
