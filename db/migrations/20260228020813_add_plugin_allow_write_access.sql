-- +goose Up
ALTER TABLE plugin ADD COLUMN allow_write_access BOOL NOT NULL DEFAULT false;

-- +goose Down
ALTER TABLE plugin DROP COLUMN allow_write_access;
