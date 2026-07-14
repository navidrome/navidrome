-- +goose Up
ALTER TABLE playlist ADD COLUMN average_rating REAL NOT NULL DEFAULT 0;

-- +goose Down
ALTER TABLE playlist DROP COLUMN average_rating;
