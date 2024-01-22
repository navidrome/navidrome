package migrations

import (
	"context"
	"database/sql"

	"github.com/pressly/goose/v3"
)

func init() {
	goose.AddMigrationContext(Up20240122223340, Down20240122223340)
}

func Up20240122223340(ctx context.Context, tx *sql.Tx) error {
	_, err := tx.Exec(`
	alter table album add image_files_new varchar not null default '';
	update album set image_files_new = image_files where image_files is not null;
	alter table album drop image_files;
	alter table album rename image_files_new to image_files;
`)
	return err
}

func Down20240122223340(ctx context.Context, tx *sql.Tx) error {
	return nil
}
