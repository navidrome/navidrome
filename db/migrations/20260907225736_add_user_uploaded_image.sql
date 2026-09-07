-- +goose Up
ALTER TABLE user ADD COLUMN uploaded_image VARCHAR(255) DEFAULT '';

-- +goose Down
ALTER TABLE user DROP COLUMN uploaded_image;
