-- +goose Up
alter table user add column scrobble_filter varchar default '' not null;

-- +goose Down
alter table user drop column scrobble_filter;
