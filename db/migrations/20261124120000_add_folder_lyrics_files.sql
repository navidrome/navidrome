-- +goose Up
-- +goose StatementBegin
alter table folder add column lyrics_files jsonb default null;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
alter table folder drop column lyrics_files;
-- +goose StatementEnd
