-- +goose Up
alter table album add column blur_hash varchar;
alter table album add column blur_hash_updated_at datetime;
alter table artist add column blur_hash varchar;
alter table artist add column blur_hash_updated_at datetime;
alter table playlist add column blur_hash varchar;
alter table playlist add column blur_hash_updated_at datetime;

-- +goose Down
alter table album drop column blur_hash;
alter table album drop column blur_hash_updated_at;
alter table artist drop column blur_hash;
alter table artist drop column blur_hash_updated_at;
alter table playlist drop column blur_hash;
alter table playlist drop column blur_hash_updated_at;
