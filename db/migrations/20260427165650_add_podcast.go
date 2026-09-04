package migrations

import (
	"context"
	"database/sql"

	"github.com/pressly/goose/v3"
)

func init() {
	goose.AddMigrationContext(upAddPodcast, downAddPodcast)
}

func upAddPodcast(ctx context.Context, tx *sql.Tx) error {
	_, err := tx.ExecContext(ctx, `
CREATE TABLE podcast_channel (
    id            VARCHAR(255) PRIMARY KEY,
    url           VARCHAR(4096) NOT NULL,
    title         VARCHAR(1024) NOT NULL DEFAULT '',
    description   TEXT          NOT NULL DEFAULT '',
    image_url     VARCHAR(4096) NOT NULL DEFAULT '',
    status        VARCHAR(32)   NOT NULL DEFAULT 'new',
    error_message TEXT          NOT NULL DEFAULT '',
    created_at    DATETIME      NOT NULL,
    updated_at    DATETIME      NOT NULL
);

CREATE TABLE podcast_episode (
    id            VARCHAR(255)  PRIMARY KEY,
    channel_id    VARCHAR(255)  NOT NULL REFERENCES podcast_channel(id) ON DELETE CASCADE,
    stream_id     VARCHAR(255)  NOT NULL DEFAULT '',
    guid          VARCHAR(4096) NOT NULL DEFAULT '',
    title         VARCHAR(1024) NOT NULL DEFAULT '',
    description   TEXT          NOT NULL DEFAULT '',
    publish_date  DATETIME,
    duration      INTEGER       NOT NULL DEFAULT 0,
    size          INTEGER       NOT NULL DEFAULT 0,
    bit_rate      INTEGER       NOT NULL DEFAULT 0,
    suffix        VARCHAR(32)   NOT NULL DEFAULT '',
    content_type  VARCHAR(255)  NOT NULL DEFAULT '',
    path          VARCHAR(4096) NOT NULL DEFAULT '',
    enclosure_url VARCHAR(4096) NOT NULL DEFAULT '',
    status        VARCHAR(32)   NOT NULL DEFAULT 'new',
    error_message TEXT          NOT NULL DEFAULT '',
    created_at    DATETIME      NOT NULL,
    updated_at    DATETIME      NOT NULL
);

CREATE INDEX podcast_episode_channel_id   ON podcast_episode(channel_id);
CREATE INDEX podcast_episode_publish_date ON podcast_episode(publish_date);
`)
	return err
}

func downAddPodcast(ctx context.Context, tx *sql.Tx) error {
	_, err := tx.ExecContext(ctx, `
DROP INDEX IF EXISTS podcast_episode_publish_date;
DROP INDEX IF EXISTS podcast_episode_channel_id;
DROP TABLE IF EXISTS podcast_episode;
DROP TABLE IF EXISTS podcast_channel;
`)
	return err
}
