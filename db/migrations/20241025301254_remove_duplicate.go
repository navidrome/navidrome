package migrations

import (
	"context"
	"database/sql"
	"github.com/pressly/goose/v3"
)

func init() {
	goose.AddMigrationContext(upRemoveDuplicate, downRemoveDuplicate)
}

func upRemoveDuplicate(ctx context.Context, tx *sql.Tx) error {
	// This code is executed when the migration is applied.
   _, err := tx.Exec(`
alter table media_file
   drop column is_duplicate;

`)
       return err
}

func downRemoveDuplicate(ctx context.Context, tx *sql.Tx) error {
	// This code is executed when the migration is rolled back.
	return nil
}
