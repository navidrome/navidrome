-- +goose Up
-- +goose StatementBegin
alter table library add column pid_album varchar default '' not null;
alter table library add column pid_track varchar default '' not null;
alter table library add column scanned_pid_album varchar default '' not null;
alter table library add column scanned_pid_track varchar default '' not null;

-- Every library was scanned with the global PID config, so seed it as their scanned config.
-- This way the upgrade does not trigger a full rescan.
update library set
    scanned_pid_album = coalesce((select value from property where id = 'PIDAlbum'), ''),
    scanned_pid_track = coalesce((select value from property where id = 'PIDTrack'), '');

delete from property where id in ('PIDAlbum', 'PIDTrack');
-- +goose StatementEnd

-- +goose Down
SELECT 1;
