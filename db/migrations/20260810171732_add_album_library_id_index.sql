-- +goose Up
-- id leads so library_id cannot drive a seek: the index covers library-filtered counts without
-- tempting the planner to sort a whole library instead of walking a sort-satisfying index.
create index if not exists album_id_library_id
	on album(id, library_id);

drop index if exists album_order_album_name;
create index album_order_album_name
	on album(order_album_name, order_album_artist_name, id);

-- +goose Down
drop index if exists album_id_library_id;

drop index if exists album_order_album_name;
create index album_order_album_name
	on album(order_album_name);
