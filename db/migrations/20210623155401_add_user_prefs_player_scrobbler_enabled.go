package migrations

import (
	"context"
	"database/sql"

	"github.com/pressly/goose/v3"
)

func init() {
	goose.AddMigrationContext(upAddUserPrefsPlayerScrobblerEnabled, downAddUserPrefsPlayerScrobblerEnabled)
}

func upAddUserPrefsPlayerScrobblerEnabled(ctx context.Context, tx *sql.Tx) error {
	err := upAddUserPrefs(ctx, tx)
	if err != nil {
		return err
	}
	return upPlayerScrobblerEnabled(ctx, tx)
}

func upAddUserPrefs(ctx context.Context, tx *sql.Tx) error {
	_, err := tx.ExecContext(ctx, `
create table user_props
(
    user_id varchar not null,
    key     varchar not null,
    value   varchar,
    constraint user_props_pk
        primary key (user_id, key)
);
`)
	return err
}

func upPlayerScrobblerEnabled(ctx context.Context, tx *sql.Tx) error {
	_, err := tx.ExecContext(ctx, `
alter table player add scrobble_enabled bool default true;
`)
	return err
}

func downAddUserPrefsPlayerScrobblerEnabled(_ context.Context, _ *sql.Tx) error {
	return nil
}
