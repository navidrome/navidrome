-- +goose Up
ALTER TABLE album ADD COLUMN avg_rating REAL NOT NULL DEFAULT 0;
ALTER TABLE media_file ADD COLUMN avg_rating REAL NOT NULL DEFAULT 0;
ALTER TABLE artist ADD COLUMN avg_rating REAL NOT NULL DEFAULT 0;

-- Populate avg_rating from existing ratings
UPDATE album SET avg_rating = coalesce(
    (SELECT round(avg(rating), 2) FROM annotation WHERE item_id = album.id AND item_type = 'album' AND rating > 0),
    0
);
UPDATE media_file SET avg_rating = coalesce(
    (SELECT round(avg(rating), 2) FROM annotation WHERE item_id = media_file.id AND item_type = 'media_file' AND rating > 0),
    0
);
UPDATE artist SET avg_rating = coalesce(
    (SELECT round(avg(rating), 2) FROM annotation WHERE item_id = artist.id AND item_type = 'artist' AND rating > 0),
    0
);

-- +goose Down
ALTER TABLE artist DROP COLUMN avg_rating;
ALTER TABLE media_file DROP COLUMN avg_rating;
ALTER TABLE album DROP COLUMN avg_rating;
