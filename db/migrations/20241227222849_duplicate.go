package migrations

import (
   "context"
   "database/sql"
   "github.com/pressly/goose/v3"
)

func init() {
   goose.AddMigrationContext(upDuplicate, downDuplicate)
}

func upDuplicate(ctx context.Context, tx *sql.Tx) error {
   _, err := tx.Exec(`
alter table media_file
   add is_duplicate bool not null default false;

`)
       return err
}

func downDuplicate(ctx context.Context, tx *sql.Tx) error {
   // This code is executed when the migration is rolled back.
   return nil
}