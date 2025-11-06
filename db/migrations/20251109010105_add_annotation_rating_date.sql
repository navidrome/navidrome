-- +goose Up
-- +goose StatementBegin
ALTER TABLE annotation ADD COLUMN rated_at datetime;
-- +goose StatementEnd

-- +goose Down
 