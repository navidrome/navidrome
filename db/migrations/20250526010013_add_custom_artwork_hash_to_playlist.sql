-- +goose Up
-- +goose StatementBegin
ALTER TABLE playlist ADD COLUMN custom_artwork_hash VARCHAR(32) DEFAULT '';
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE playlist DROP COLUMN custom_artwork_hash;
-- +goose StatementEnd
