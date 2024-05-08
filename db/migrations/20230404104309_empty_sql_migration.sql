--  This file has intentionally no SQL logic. It is here to avoid an error in the linter:
--  db/db.go:23:4: invalid go:embed: build system did not supply embed configuration (typecheck)
--

-- +goose Up
-- +goose StatementBegin
SELECT 'up SQL query';
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
SELECT 'down SQL query';
-- +goose StatementEnd
