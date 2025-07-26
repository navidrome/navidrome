package migrations

import (
	"context"
	"database/sql"

	"github.com/pressly/goose/v3"
)

func init() {
	goose.AddMigrationContext(upAddApiKeyTable, downAddApiKeyTable)
}

func upAddApiKeyTable(ctx context.Context, tx *sql.Tx) error {
	_, err := tx.ExecContext(ctx, `
		create table if not exists api_key (
			id text not null primary key,
			player_id text not null,
			name text not null,
			key text not null unique,
			created_at datetime not null,

			foreign key (player_id) 
				references player(id)
		    	on delete cascade
		);

		create index if not exists api_key_key on api_key(key);
		create index if not exists api_key_player_id on api_key(player_id);
`)
	return err
}

func downAddApiKeyTable(ctx context.Context, tx *sql.Tx) error {
	_, err := tx.ExecContext(ctx, `
		drop table api_key;
`)
	return err
}
