-- +goose Up
-- +goose StatementBegin
ALTER TABLE playqueue ADD COLUMN current_new integer;
UPDATE playqueue SET current_new = current;
ALTER TABLE playqueue DROP COLUMN current;
ALTER TABLE playqueue RENAME COLUMN current_new TO current;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
SELECT 'down SQL query';
-- +goose StatementEnd
