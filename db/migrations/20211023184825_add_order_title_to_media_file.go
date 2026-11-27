package migrations

import (
	"context"
	"database/sql"
	"strings"

	"github.com/deluan/sanitize"
	"github.com/navidrome/navidrome/log"
	"github.com/pressly/goose/v3"
)

func init() {
	goose.AddMigrationContext(upAddOrderTitleToMediaFile, downAddOrderTitleToMediaFile)
}

func upAddOrderTitleToMediaFile(ctx context.Context, tx *sql.Tx) error {
	_, err := tx.ExecContext(ctx, `
alter table main.media_file
	add order_title varchar null collate NOCASE;
create index if not exists media_file_order_title
    on media_file (order_title);
`)
	if err != nil {
		return err
	}

	return upAddOrderTitleToMediaFile_populateOrderTitle(ctx, tx)
}

//goland:noinspection GoSnakeCaseUsage
func upAddOrderTitleToMediaFile_populateOrderTitle(ctx context.Context, tx *sql.Tx) error {
	rows, err := tx.QueryContext(ctx, `select id, title from media_file`)
	if err != nil {
		return err
	}
	defer rows.Close()

	stmt, err := tx.Prepare("update media_file set order_title = ? where id = ?")
	if err != nil {
		return err
	}

	var id, title string
	for rows.Next() {
		err = rows.Scan(&id, &title)
		if err != nil {
			return err
		}

		orderTitle := strings.TrimSpace(sanitize.Accents(title))
		_, err = stmt.Exec(orderTitle, id)
		if err != nil {
			log.Error("Error setting media_file's order_title", "title", title, "id", id, err)
		}
	}
	return rows.Err()
}

func downAddOrderTitleToMediaFile(_ context.Context, _ *sql.Tx) error {
	return nil
}
