package migrations

import (
	"context"
	"database/sql"
	"strings"

	"github.com/pressly/goose/v3"
)

func init() {
	goose.AddMigrationContext(upPlayQueueCurrentToIndex, downPlayQueueCurrentToIndex)
}

func upPlayQueueCurrentToIndex(ctx context.Context, tx *sql.Tx) error {
	_, err := tx.ExecContext(ctx, `
create table playqueue_dg_tmp(
    id varchar(255) not null,
    user_id varchar(255) not null
        references user(id)
        on update cascade on delete cascade,
    current integer not null default 0,
    position real,
    changed_by varchar(255),
    items varchar(255),
    created_at datetime,
    updated_at datetime
);`)
	if err != nil {
		return err
	}

	rows, err := tx.QueryContext(ctx, `select id, user_id, current, position, changed_by, items, created_at, updated_at from playqueue`)
	if err != nil {
		return err
	}
	defer rows.Close()

	stmt, err := tx.PrepareContext(ctx, `insert into playqueue_dg_tmp(id, user_id, current, position, changed_by, items, created_at, updated_at) values(?,?,?,?,?,?,?,?)`)
	if err != nil {
		return err
	}
	defer stmt.Close()

	for rows.Next() {
		var id, userID, currentID, changedBy, items string
		var position sql.NullFloat64
		var createdAt, updatedAt sql.NullString
		if err = rows.Scan(&id, &userID, &currentID, &position, &changedBy, &items, &createdAt, &updatedAt); err != nil {
			return err
		}
		index := 0
		if currentID != "" && items != "" {
			parts := strings.Split(items, ",")
			for i, p := range parts {
				if p == currentID {
					index = i
					break
				}
			}
		}
		_, err = stmt.Exec(id, userID, index, position, changedBy, items, createdAt, updatedAt)
		if err != nil {
			return err
		}
	}
	if err = rows.Err(); err != nil {
		return err
	}

	if _, err = tx.ExecContext(ctx, `drop table playqueue;`); err != nil {
		return err
	}
	_, err = tx.ExecContext(ctx, `alter table playqueue_dg_tmp rename to playqueue;`)
	return err
}

func downPlayQueueCurrentToIndex(ctx context.Context, tx *sql.Tx) error {
	return nil
}
