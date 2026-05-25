-- +goose Up
-- Backfill PIDArtist property to reflect the historical artist-ID computation.
-- Existing artist IDs were produced by the legacy hardcoded artistID() function,
-- which is byte-identical to computeArtistPID(p, "name", ...). Recording "name"
-- here ensures that on the next scan, prevArtistPIDConf is never empty — closing
-- the upgrade-time window where a user who pre-configured a non-default PID.Artist
-- would have artist IDs silently regenerated without annotation migration.
insert into property (id, value) values ('PIDArtist', 'name') on conflict do nothing;

-- +goose Down
delete from property where id = 'PIDArtist';
