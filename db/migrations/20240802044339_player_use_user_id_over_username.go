package migrations

import (
	"context"
	"database/sql"

	"github.com/pressly/goose/v3"
)

func init() {
	goose.AddMigrationContext(upPlayerUseUserIdOverUsername, downPlayerUseUserIdOverUsername)
}

func upPlayerUseUserIdOverUsername(ctx context.Context, tx *sql.Tx) error {
	_, err := tx.ExecContext(ctx, `
CREATE TABLE player_dg_tmp
(
	id varchar(255) not null
		primary key,
	name varchar not null,
	user_agent varchar,
	user_id varchar not null
		references user (id)
			on update cascade on delete cascade,
	client varchar not null,
	ip varchar,
	last_seen timestamp,
	max_bit_rate int default 0,
	transcoding_id varchar,
	report_real_path bool default FALSE not null,
	scrobble_enabled bool default true
);

INSERT INTO player_dg_tmp(
	id, name, user_agent, user_id, client, ip, last_seen, max_bit_rate,
	transcoding_id, report_real_path, scrobble_enabled
)
SELECT
	id, name, user_agent,
	IFNULL(
		(select id from user where user_name = player.user_name), 'UNKNOWN_USERNAME'
	),
	client, ip_address, last_seen, max_bit_rate, transcoding_id, report_real_path, scrobble_enabled
FROM player;

DELETE FROM player_dg_tmp WHERE user_id = 'UNKNOWN_USERNAME';
DROP TABLE player;
ALTER TABLE player_dg_tmp RENAME TO player;

CREATE INDEX IF NOT EXISTS player_match
	on player (client, user_agent, user_id);
CREATE INDEX IF NOT EXISTS player_name
	on player (name);
`)

	return err
}

func downPlayerUseUserIdOverUsername(ctx context.Context, tx *sql.Tx) error {
	// This code is executed when the migration is rolled back.
	return nil
}
