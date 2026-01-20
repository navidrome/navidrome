-- +goose Up
ALTER TABLE album ADD COLUMN average_rating REAL NOT NULL DEFAULT 0;
ALTER TABLE media_file ADD COLUMN average_rating REAL NOT NULL DEFAULT 0;
ALTER TABLE artist ADD COLUMN average_rating REAL NOT NULL DEFAULT 0;

-- Populate average_rating from existing ratings
UPDATE album SET average_rating = coalesce(
    (SELECT round(avg(rating), 2) FROM annotation WHERE item_id = album.id AND item_type = 'album' AND rating > 0),
    0
);
UPDATE media_file SET average_rating = coalesce(
    (SELECT round(avg(rating), 2) FROM annotation WHERE item_id = media_file.id AND item_type = 'media_file' AND rating > 0),
    0
);
UPDATE artist SET average_rating = coalesce(
    (SELECT round(avg(rating), 2) FROM annotation WHERE item_id = artist.id AND item_type = 'artist' AND rating > 0),
    0
);

-- +goose Down
ALTER TABLE artist DROP COLUMN average_rating;
ALTER TABLE media_file DROP COLUMN average_rating;
ALTER TABLE album DROP COLUMN average_rating;
