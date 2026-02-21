package migrations

import (
	"context"
	"database/sql"

	"github.com/pressly/goose/v3"
)

func init() {
	goose.AddMigrationContext(upAddAnnotationRatingDate, downAddAnnotationRatingDate)
}

func upAddAnnotationRatingDate(ctx context.Context, tx *sql.Tx) error {
	_, err := tx.ExecContext(ctx, adaptSQL(`ALTER TABLE annotation ADD COLUMN rated_at datetime;`))
	return err
}

func downAddAnnotationRatingDate(_ context.Context, _ *sql.Tx) error {
	return nil
}
