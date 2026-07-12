package migrations

import (
	"context"
	"database/sql"

	"github.com/pressly/goose/v3"
)

func init() {
	goose.AddMigrationContext(upCreatePodcasts, downCreatePodcasts)
}

func upCreatePodcasts(ctx context.Context, tx *sql.Tx) error {
	_, err := tx.ExecContext(ctx, `
create table if not exists podcast_channel
(
    id                 varchar(255) not null primary key,
    url                varchar not null unique,
    title              varchar not null,
    description        varchar default '' not null,
    cover_art_url      varchar default '' not null,
    uploaded_image     varchar default '' not null,
    original_image_url varchar default '' not null,
    home_page_url      varchar default '' not null,
    status             varchar default 'new' not null,
    error_message      varchar default '' not null,
    download_policy    varchar default 'none' not null,
    retention_count    integer default 0 not null,
    retention_days     integer default 0 not null,
    last_checked_at    datetime,
    created_at         datetime,
    updated_at         datetime
);

create table if not exists podcast_episode
(
    id              varchar(255) not null primary key,
    channel_id      varchar(255) not null references podcast_channel (id) on delete cascade,
    guid            varchar not null,
    title           varchar not null,
    description     varchar default '' not null,
    enclosure_url   varchar not null,
    content_type    varchar default '' not null,
    size            integer default 0 not null,
    duration        real default 0 not null,
    publish_date    datetime,
    download_status varchar default 'not_downloaded' not null,
    error_message   varchar default '' not null,
    path            varchar default '' not null,
    suffix          varchar default '' not null,
    bit_rate        integer default 0 not null,
    created_at      datetime,
    updated_at      datetime,
    unique (channel_id, guid)
);

create index if not exists idx_podcast_episode_channel_id on podcast_episode (channel_id);
create index if not exists idx_podcast_episode_download_status on podcast_episode (download_status);
create index if not exists idx_podcast_episode_publish_date on podcast_episode (publish_date);
`)
	return err
}

func downCreatePodcasts(ctx context.Context, tx *sql.Tx) error {
	_, err := tx.ExecContext(ctx, `drop table if exists podcast_episode; drop table if exists podcast_channel;`)
	return err
}
