-- +goose Up
ALTER TABLE playlist ADD COLUMN average_rating REAL NOT NULL DEFAULT 0;

-- Older builds mis-wrote playlist stars/ratings as item_type='media_file' (Subsonic
-- star/setRating fell through to the media_file branch before playlists were
-- annotatable). Reclassify them to item_type='playlist' so the user's prior
-- star/rating is preserved and read correctly. A media_file id never equals a
-- playlist id, so this touches only the mis-typed rows; no playlist-typed rows exist
-- yet, so the (user_id, item_id, item_type) unique key cannot conflict. Run before
-- the backfill below so reclassified ratings are included in average_rating.
UPDATE annotation SET item_type = 'playlist' WHERE item_type = 'media_file' AND item_id IN (SELECT id FROM playlist);

-- Populate average_rating from existing playlist ratings (including any just reclassified).
UPDATE playlist SET average_rating = coalesce(
    (SELECT round(avg(rating), 2) FROM annotation WHERE item_id = playlist.id AND item_type = 'playlist' AND rating > 0),
    0
);

-- +goose Down
ALTER TABLE playlist DROP COLUMN average_rating;
