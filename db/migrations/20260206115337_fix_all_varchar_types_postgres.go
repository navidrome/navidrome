package migrations

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/navidrome/navidrome/db/dialect"
	"github.com/pressly/goose/v3"
)

func init() {
	goose.AddMigrationContext(upFixAllVarcharTypesPostgres, downFixAllVarcharTypesPostgres)
}

func upFixAllVarcharTypesPostgres(ctx context.Context, tx *sql.Tx) error {
	if dialect.Current == nil || dialect.Current.Name() != "postgres" {
		return nil
	}

	// Catchall for any varchar(255) columns missed by the previous migration.
	rows, err := tx.QueryContext(ctx, `
		SELECT table_name, column_name
		FROM information_schema.columns
		WHERE table_schema = 'public'
		  AND character_maximum_length = 255
		  AND data_type = 'character varying'
		ORDER BY table_name, column_name
	`)
	if err != nil {
		return fmt.Errorf("querying varchar(255) columns: %w", err)
	}
	defer rows.Close()

	type col struct {
		table, column string
	}
	var cols []col
	for rows.Next() {
		var c col
		if err := rows.Scan(&c.table, &c.column); err != nil {
			return fmt.Errorf("scanning column info: %w", err)
		}
		cols = append(cols, c)
	}
	if err := rows.Err(); err != nil {
		return fmt.Errorf("iterating columns: %w", err)
	}

	for _, c := range cols {
		_, err := tx.ExecContext(ctx, fmt.Sprintf(
			`ALTER TABLE %q ALTER COLUMN %q TYPE text`, c.table, c.column))
		if err != nil {
			return fmt.Errorf("altering %s.%s to text: %w", c.table, c.column, err)
		}
	}

	return nil
}

func downFixAllVarcharTypesPostgres(ctx context.Context, tx *sql.Tx) error {
	return nil
}
