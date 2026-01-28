-- +goose Up
ALTER TABLE library ADD COLUMN ignored BOOLEAN NOT NULL DEFAULT false;

-- +goose Down
ALTER TABLE library DROP COLUMN ignored;
