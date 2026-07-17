-- +goose Up
ALTER TABLE album ADD COLUMN uploaded_image varchar NOT NULL DEFAULT '';

-- +goose Down
ALTER TABLE album DROP COLUMN uploaded_image;
