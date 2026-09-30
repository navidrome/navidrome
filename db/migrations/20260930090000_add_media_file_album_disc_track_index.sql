-- +goose Up
create index if not exists media_file_album_disc_track
    on media_file (album_id, disc_number, track_number);

-- +goose Down
drop index if exists media_file_album_disc_track;
