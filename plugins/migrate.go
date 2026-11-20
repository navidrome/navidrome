package plugins

import (
	"context"
	"database/sql"
	"fmt"
)

// migrateDB applies schema migrations to a SQLite database.
//
// Each entry in migrations is a single SQL statement. The current schema version
// is tracked using SQLite's built-in PRAGMA user_version. Only statements after
// the current version are executed, within a single transaction.
func migrateDB(ctx context.Context, db *sql.DB, migrations []string) error {
	var version int
	if err := db.QueryRowContext(ctx, `PRAGMA user_version`).Scan(&version); err != nil {
		return fmt.Errorf("reading schema version: %w", err)
	}

	if version >= len(migrations) {
		return nil
	}

	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("starting migration transaction: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	for i := version; i < len(migrations); i++ {
		if _, err := tx.ExecContext(ctx, migrations[i]); err != nil {
			return fmt.Errorf("migration %d failed: %w", i+1, err)
		}
	}

	// PRAGMA statements cannot be executed inside a transaction in some SQLite
	// drivers, but with mattn/go-sqlite3 this works. We set it inside the tx
	// so that a failed commit leaves the version unchanged.
	if _, err := tx.ExecContext(ctx, fmt.Sprintf(`PRAGMA user_version = %d`, len(migrations))); err != nil {
		return fmt.Errorf("updating schema version: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("committing migrations: %w", err)
	}

	return nil
}
