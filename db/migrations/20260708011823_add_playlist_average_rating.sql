-- +goose Up
ALTER TABLE playlist ADD COLUMN average_rating REAL NOT NULL DEFAULT 0;

-- Populate average_rating from any existing playlist ratings (none yet, but keep parity)
UPDATE playlist SET average_rating = coalesce(
    (SELECT round(avg(rating), 2) FROM annotation WHERE item_id = playlist.id AND item_type = 'playlist' AND rating > 0),
    0
);

-- Remove stale annotation rows older builds mis-wrote for playlist ids as
-- item_type='media_file' (Subsonic star/setRating fell through to the media_file
-- branch before playlists were annotatable). Left in place they would surface via
-- the playlist annotation join (which matches on item_id) and could duplicate a
-- playlist in list results. A random media_file id never equals a playlist id, so
-- this touches only the mis-typed rows.
DELETE FROM annotation WHERE item_type = 'media_file' AND item_id IN (SELECT id FROM playlist);

-- +goose Down
ALTER TABLE playlist DROP COLUMN average_rating;
