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
create table player_dg_tmp
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

insert into player_dg_tmp(id, name, user_agent, user_id, client, ip, last_seen, max_bit_rate, transcoding_id, report_real_path, scrobble_enabled) select id, name, user_agent, user_name, client, ip_address, last_seen, max_bit_rate, transcoding_id, report_real_path, scrobble_enabled from player;
	`)
	if err != nil {
		return err
	}

	userRows, err := tx.QueryContext(ctx, `SELECT id, user_name FROM user`)
	if err != nil {
		return err
	}
	users := map[string]string{}

	var userId, userName string
	for userRows.Next() {
		err = userRows.Scan(&userId, &userName)
		if err != nil {
			return err
		}

		users[userName] = userId
	}

	err = userRows.Err()
	if err != nil {
		return err
	}

	updStmt, err := tx.PrepareContext(ctx, `update player_dg_tmp SET user_id = ? where id = ?`)
	if err != nil {
		return err
	}

	delStmt, err := tx.PrepareContext(ctx, `DELETE FROM player_dg_tmp WHERE id = ?`)
	if err != nil {
		return err
	}

	playerRows, err := tx.QueryContext(ctx, `SELECT id, user_id FROM player_dg_tmp`)
	if err != nil {
		return err
	}

	var playerId string
	for playerRows.Next() {
		err = playerRows.Scan(&playerId, &userName)
		if err != nil {
			return err
		}

		userId, existing := users[userName]
		if !existing {
			_, err = delStmt.ExecContext(ctx, playerId)
		} else {
			_, err = updStmt.ExecContext(ctx, userId, playerId)
		}

		if err != nil {
			return err
		}
	}

	err = playerRows.Err()
	if err != nil {
		return err
	}

	_, err = tx.ExecContext(ctx, `
drop table player;
alter table player_dg_tmp rename to player;
create index if not exists player_match
	on player (client, user_agent, user_id);
create index if not exists player_name
	on player (name);
	`)
	return err
}

func downPlayerUseUserIdOverUsername(ctx context.Context, tx *sql.Tx) error {
	// This code is executed when the migration is rolled back.
	return nil
}
