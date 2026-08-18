-- +goose Up
alter table playlist add imported_hash varchar default '' not null;

-- +goose Down
alter table playlist drop column imported_hash;
