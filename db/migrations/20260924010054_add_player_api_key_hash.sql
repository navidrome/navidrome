-- +goose Up
-- +goose StatementBegin
alter table player add column api_key_hash varchar default null;
create unique index if not exists player_api_key_hash on player(api_key_hash);
-- +goose StatementEnd

-- +goose Down
SELECT 1;
