-- +goose Up

ALTER TABLE user ADD COLUMN token_epoch INTEGER NOT NULL DEFAULT 0;

-- +goose Down

ALTER TABLE user DROP COLUMN token_epoch;
