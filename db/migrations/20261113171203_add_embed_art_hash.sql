-- +goose Up

ALTER TABLE media_file ADD COLUMN embed_art_hash TEXT NOT NULL DEFAULT '';
ALTER TABLE album ADD COLUMN embed_art_hash TEXT NOT NULL DEFAULT '';

-- +goose Down

ALTER TABLE media_file DROP COLUMN embed_art_hash;
ALTER TABLE album DROP COLUMN embed_art_hash;
