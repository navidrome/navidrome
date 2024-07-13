package migrations

import (
	"context"
	"database/sql"

	"github.com/pressly/goose/v3"
)

func init() {
	goose.AddMigrationContext(upAddPodcasts, downAddPodcasts)
}

func upAddPodcasts(ctx context.Context, tx *sql.Tx) error {
	_, err := tx.ExecContext(ctx, `
create table if not exists podcast
(
	id 			varchar(255) not null primary key,
	url			varchar not null,
	title		varchar default '' not null,
	description varchar default '' not null,
	image_url   varchar default '' not null,
	state       varchar(20) default 'new' not null,
	error       varchar default '' not null,
	created_at  datetime,
	updated_at  datetime
);

create table podcast_episode
(
	id 			 varchar(255) not null primary key,
	guid         varchar not null,
	podcast_id   varchar(255) not null
		references podcast (id)
			on update cascade on delete cascade,
	url			 varchar not null,
	title		 varchar default '' not null,
	description  varchar default '' not null,
	image_url   varchar default '' not null,
	publish_date datetime,
	duration     integer default 0 not null,
	suffix		 varchar(255) default '' not null,
	size         integer default 0 not null,
	bit_rate     integer default 0 not null,
	state        varchar(20) default 'new' not null,
	error        varchar default '' not null,
	created_at   datetime,
	updated_at   datetime
)
`)

	return err
}

func downAddPodcasts(ctx context.Context, tx *sql.Tx) error {
	// This code is executed when the migration is rolled back.
	return nil
}
