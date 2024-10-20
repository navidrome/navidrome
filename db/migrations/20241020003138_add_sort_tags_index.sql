-- +goose Up
create index if not exists media_file_sort_title on media_file(coalesce(nullif(sort_title,''),order_title));
create index if not exists album_sort_name on album(coalesce(nullif(sort_album_name,''),order_album_name));
create index if not exists artist_sort_name on artist(coalesce(nullif(sort_artist_name,''),order_artist_name));

-- +goose Down
drop index if exists media_file_sort_title;
drop index if exists album_sort_name;
drop index if exists artist_sort_name;