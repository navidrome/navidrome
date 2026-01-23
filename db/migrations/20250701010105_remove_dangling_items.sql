-- +goose Up
-- +goose StatementBegin
update media_file set missing = 1 where folder_id = '';
update album set missing = 1 where folder_ids = '[]';
-- +goose StatementEnd

-- +goose Down
