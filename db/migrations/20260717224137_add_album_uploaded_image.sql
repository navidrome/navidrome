-- +goose Up
ALTER TABLE album ADD COLUMN uploaded_image varchar NOT NULL DEFAULT '';
-- cover_art_updated_at tracks manual cover edits separately from updated_at, so
-- busting the artwork cache never disturbs updated_at-based Recently Added ordering.
ALTER TABLE album ADD COLUMN cover_art_updated_at datetime;

-- +goose Down
ALTER TABLE album DROP COLUMN uploaded_image;
ALTER TABLE album DROP COLUMN cover_art_updated_at;
