-- +goose Up
-- +goose StatementBegin
-- The default album PID no longer has releasedate. Libraries scanned with the old default keep it,
-- so their album IDs do not change.
update library set pid_album = 'musicbrainz_albumid|albumartistid,album,albumversion,releasedate'
where pid_album = ''
  and lower(scanned_pid_album) = 'musicbrainz_albumid|albumartistid,album,albumversion,releasedate';
-- +goose StatementEnd

-- +goose Down
SELECT 1;
