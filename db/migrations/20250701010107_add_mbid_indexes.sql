-- +goose Up
-- +goose StatementBegin

-- Add indexes for MBID fields to improve lookup performance
-- Artists table
create index if not exists artist_mbz_artist_id
	on artist (mbz_artist_id);

-- Albums table  
create index if not exists album_mbz_album_id
	on album (mbz_album_id);

-- Media files table
create index if not exists media_file_mbz_release_track_id
	on media_file (mbz_release_track_id);

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin

-- Remove MBID indexes
drop index if exists artist_mbz_artist_id;
drop index if exists album_mbz_album_id;
drop index if exists media_file_mbz_release_track_id;

-- +goose StatementEnd
