-- +goose Up
-- +goose StatementBegin
ALTER TABLE playqueue ADD COLUMN position_int integer;
UPDATE playqueue SET position_int = CAST(position as INTEGER) ;
ALTER TABLE playqueue DROP COLUMN position;
ALTER TABLE playqueue RENAME COLUMN position_int TO position;
-- +goose StatementEnd

-- +goose Down
