-- +goose Up
ALTER TABLE scrobbles ADD COLUMN client VARCHAR DEFAULT '' NOT NULL;
ALTER TABLE scrobbles ADD COLUMN source VARCHAR DEFAULT '' NOT NULL;
ALTER TABLE scrobbles ADD COLUMN origin VARCHAR DEFAULT '' NOT NULL;
ALTER TABLE scrobbles ADD COLUMN playback_mode VARCHAR DEFAULT '' NOT NULL;

ALTER TABLE scrobble_buffer ADD COLUMN client VARCHAR DEFAULT '' NOT NULL;
ALTER TABLE scrobble_buffer ADD COLUMN source VARCHAR DEFAULT '' NOT NULL;
ALTER TABLE scrobble_buffer ADD COLUMN origin VARCHAR DEFAULT '' NOT NULL;
ALTER TABLE scrobble_buffer ADD COLUMN playback_mode VARCHAR DEFAULT '' NOT NULL;

CREATE INDEX IF NOT EXISTS idx_scrobble_attribution ON scrobbles (client, source, origin);

-- +goose Down
DROP INDEX IF EXISTS idx_scrobble_attribution;
ALTER TABLE scrobbles DROP COLUMN client;
ALTER TABLE scrobbles DROP COLUMN source;
ALTER TABLE scrobbles DROP COLUMN origin;
ALTER TABLE scrobbles DROP COLUMN playback_mode;

ALTER TABLE scrobble_buffer DROP COLUMN client;
ALTER TABLE scrobble_buffer DROP COLUMN source;
ALTER TABLE scrobble_buffer DROP COLUMN origin;
ALTER TABLE scrobble_buffer DROP COLUMN playback_mode;

