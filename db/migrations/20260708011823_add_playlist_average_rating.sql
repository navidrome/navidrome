-- +goose Up
ALTER TABLE playlist ADD COLUMN average_rating REAL NOT NULL DEFAULT 0;

-- Older builds mis-wrote playlist star/rating as item_type='media_file' (Subsonic
-- star/setRating fell through to the media_file branch before playlists were
-- annotatable). Reclassify to 'playlist' to preserve them: playlist and media_file
-- ids never collide, so only mis-typed rows match and the unique key can't conflict.
-- Runs before the backfill so reclassified ratings feed average_rating.
UPDATE annotation SET item_type = 'playlist' WHERE item_type = 'media_file' AND item_id IN (SELECT id FROM playlist);

UPDATE playlist SET average_rating = coalesce(
    (SELECT round(avg(rating), 2) FROM annotation WHERE item_id = playlist.id AND item_type = 'playlist' AND rating > 0),
    0
);

-- +goose Down
ALTER TABLE playlist DROP COLUMN average_rating;
