-- +goose Up
-- +goose StatementBegin
alter table user add column auth_source varchar(32) default '' not null;
alter table user add column auth_source_id varchar(255) default '' not null;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
alter table user drop column auth_source;
alter table user drop column auth_source_id;
-- +goose StatementEnd
