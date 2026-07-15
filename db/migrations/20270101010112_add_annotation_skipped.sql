-- +goose Up
-- +goose StatementBegin
ALTER TABLE annotation ADD COLUMN skipped bool default FALSE not null;
-- +goose StatementEnd
-- +goose StatementBegin
ALTER TABLE annotation ADD COLUMN skipped_at datetime;
-- +goose StatementEnd

-- +goose Down
