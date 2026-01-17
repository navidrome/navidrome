-- +goose Up
CREATE INDEX IF NOT EXISTS annotation_item_type_rating
    ON annotation (item_id, item_type, rating);

-- +goose Down
DROP INDEX IF EXISTS annotation_item_type_rating;
