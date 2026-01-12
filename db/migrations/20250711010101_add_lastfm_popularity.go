package migrations

import (
	"context"
	"database/sql"

	"github.com/pressly/goose/v3"
)

func init() {
	goose.AddMigrationContext(upAddLastFMPopularity, downAddLastFMPopularity)
}

func upAddLastFMPopularity(_ context.Context, tx *sql.Tx) error {
	_, err := tx.Exec(`
alter table artist
	add lastfm_listeners integer default 0 not null;
alter table artist
	add lastfm_playcount integer default 0 not null;
alter table album
	add lastfm_listeners integer default 0 not null;
alter table album
	add lastfm_playcount integer default 0 not null;
`)
	return err
}

func downAddLastFMPopularity(_ context.Context, tx *sql.Tx) error {
	return nil
}
