-- +goose Up
-- +goose StatementBegin
ALTER TABLE playlist ADD COLUMN global BOOL DEFAULT FALSE NOT NULL;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE playlist DROP COLUMN global;
-- +goose StatementEnd
