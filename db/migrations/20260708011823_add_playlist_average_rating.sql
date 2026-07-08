-- +goose Up
ALTER TABLE playlist ADD COLUMN average_rating REAL NOT NULL DEFAULT 0;

-- Populate average_rating from any existing playlist ratings (none yet, but keep parity)
UPDATE playlist SET average_rating = coalesce(
    (SELECT round(avg(rating), 2) FROM annotation WHERE item_id = playlist.id AND item_type = 'playlist' AND rating > 0),
    0
);

-- +goose Down
ALTER TABLE playlist DROP COLUMN average_rating;
