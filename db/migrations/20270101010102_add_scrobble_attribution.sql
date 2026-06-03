-- +goose Up
ALTER TABLE scrobble ADD COLUMN client VARCHAR DEFAULT '' NOT NULL;
ALTER TABLE scrobble ADD COLUMN source VARCHAR DEFAULT '' NOT NULL;
ALTER TABLE scrobble ADD COLUMN origin VARCHAR DEFAULT '' NOT NULL;
ALTER TABLE scrobble ADD COLUMN playback_mode VARCHAR DEFAULT '' NOT NULL;

ALTER TABLE scrobble_buffer ADD COLUMN client VARCHAR DEFAULT '' NOT NULL;
ALTER TABLE scrobble_buffer ADD COLUMN source VARCHAR DEFAULT '' NOT NULL;
ALTER TABLE scrobble_buffer ADD COLUMN origin VARCHAR DEFAULT '' NOT NULL;
ALTER TABLE scrobble_buffer ADD COLUMN playback_mode VARCHAR DEFAULT '' NOT NULL;

CREATE INDEX IF NOT EXISTS idx_scrobble_attribution ON scrobble (client, source, origin);

-- +goose Down
DROP INDEX IF EXISTS idx_scrobble_attribution;
ALTER TABLE scrobble DROP COLUMN client;
ALTER TABLE scrobble DROP COLUMN source;
ALTER TABLE scrobble DROP COLUMN origin;
ALTER TABLE scrobble DROP COLUMN playback_mode;

ALTER TABLE scrobble_buffer DROP COLUMN client;
ALTER TABLE scrobble_buffer DROP COLUMN source;
ALTER TABLE scrobble_buffer DROP COLUMN origin;
ALTER TABLE scrobble_buffer DROP COLUMN playback_mode;
