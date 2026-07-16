-- +goose Up
-- blur_hash is not null default '' so NULLs never reach the Go string field; '' means "not computed".
alter table album add column blur_hash varchar not null default '';
alter table album add column blur_hash_updated_at datetime;
alter table artist add column blur_hash varchar not null default '';
alter table artist add column blur_hash_updated_at datetime;
alter table playlist add column blur_hash varchar not null default '';
alter table playlist add column blur_hash_updated_at datetime;

-- +goose Down
alter table album drop column blur_hash;
alter table album drop column blur_hash_updated_at;
alter table artist drop column blur_hash;
alter table artist drop column blur_hash_updated_at;
alter table playlist drop column blur_hash;
alter table playlist drop column blur_hash_updated_at;
