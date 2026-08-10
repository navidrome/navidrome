-- +goose Up
-- +goose StatementBegin
-- Lead the unique constraint with artist_id (mirroring media_file_artists) so artist-driven
-- filters and the artist-delete cascade get an index seek; album_id lookups keep album_artists_album_id.
CREATE TABLE album_artists_tmp
(
    album_id  varchar not null
        REFERENCES album (id)
            ON DELETE CASCADE,
    artist_id varchar not null
        REFERENCES artist (id)
            ON DELETE CASCADE,
    role      varchar default '' not null,
    sub_role  varchar default '' not null,
    CONSTRAINT album_artists
        UNIQUE (artist_id, album_id, role, sub_role)
);

INSERT INTO album_artists_tmp(album_id, artist_id, role, sub_role)
SELECT album_id, artist_id, role, sub_role
FROM album_artists;

DROP TABLE album_artists;
ALTER TABLE album_artists_tmp RENAME TO album_artists;

CREATE INDEX album_artists_album_id
    ON album_artists (album_id);
CREATE INDEX album_artists_role
    ON album_artists (role);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
CREATE TABLE album_artists_tmp
(
    album_id  varchar not null
        REFERENCES album (id)
            ON DELETE CASCADE,
    artist_id varchar not null
        REFERENCES artist (id)
            ON DELETE CASCADE,
    role      varchar default '' not null,
    sub_role  varchar default '' not null,
    CONSTRAINT album_artists
        UNIQUE (album_id, artist_id, role, sub_role)
);

INSERT INTO album_artists_tmp(album_id, artist_id, role, sub_role)
SELECT album_id, artist_id, role, sub_role
FROM album_artists;

DROP TABLE album_artists;
ALTER TABLE album_artists_tmp RENAME TO album_artists;

CREATE INDEX album_artists_album_id
    ON album_artists (album_id);
CREATE INDEX album_artists_role
    ON album_artists (role);
-- +goose StatementEnd
