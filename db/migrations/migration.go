package migrations

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"sync"

	"github.com/navidrome/navidrome/conf"
	"github.com/navidrome/navidrome/consts"
	"github.com/navidrome/navidrome/db/dialect"
)

func dropCascadeIfPostgres() string {
	if dialect.Current != nil && dialect.Current.Name() == "postgres" {
		return " CASCADE"
	}
	return ""
}

func IsPostgres() bool {
	return dialect.Current != nil && dialect.Current.Name() == "postgres"
}

// adaptSQL rewrites SQLite-flavored SQL for PostgreSQL when needed.
func adaptSQL(sql string) string {
	if dialect.Current != nil && dialect.Current.Name() == "postgres" {
		// Replace SQLite datetime functions with PostgreSQL equivalents
		sql = strings.ReplaceAll(sql, "datetime('now')", "NOW()")
		sql = strings.ReplaceAll(sql, "(datetime(current_timestamp, 'localtime'))", "NOW()")
		// Replace datetime type with timestamp for PostgreSQL
		sql = strings.ReplaceAll(sql, " datetime", " timestamp")
		sql = strings.ReplaceAll(sql, "(datetime", "(timestamp")
		sql = strings.ReplaceAll(sql, " DATETIME", " timestamp")
		sql = strings.ReplaceAll(sql, "(DATETIME", "(timestamp")
		// Replace autoincrement with PostgreSQL serial
		sql = strings.ReplaceAll(sql, "integer primary key autoincrement", "SERIAL PRIMARY KEY")
		// Replace SQLite string type with text
		sql = strings.ReplaceAll(sql, " string ", " text ")
		sql = strings.ReplaceAll(sql, " string default", " text default")
		// Replace SQLite bool literal with PostgreSQL
		sql = strings.ReplaceAll(sql, "default FALSE", "default false")
		sql = strings.ReplaceAll(sql, "default TRUE", "default true")
		// Remove SQLite collate nocase
		sql = strings.ReplaceAll(sql, " collate nocase", "")
		sql = strings.ReplaceAll(sql, " collate NOCASE", "")
		sql = strings.ReplaceAll(sql, " COLLATE nocase", "")
		sql = strings.ReplaceAll(sql, " COLLATE NOCASE", "")
		// Fix SQLite "where x not null" to PostgreSQL "where x IS NOT NULL"
		sql = strings.ReplaceAll(sql, "where id not null", "where id IS NOT NULL")
		sql = strings.ReplaceAll(sql, "WHERE id NOT NULL", "WHERE id IS NOT NULL")
		// Replace SQLite "zero date" with PostgreSQL-compatible minimum date
		sql = strings.ReplaceAll(sql, "'0000-00-00 00:00:00'", "'1970-01-01 00:00:00'")
		sql = strings.ReplaceAll(sql, "\"0000-00-00 00:00:00\"", "'1970-01-01 00:00:00'")
		// Replace current_time with current_timestamp
		// Handle exact match first
		if sql == "current_time" {
			sql = "current_timestamp"
		} else {
			// Use word boundary checks to avoid replacing current_timestamp -> current_timestampstamp
			for _, sep := range []string{",", ")", " ", ";", "\n", "\t"} {
				sql = strings.ReplaceAll(sql, "current_time"+sep, "current_timestamp"+sep)
			}
		}
		// PostgreSQL can't have constraint and table with the same name
		sql = strings.ReplaceAll(sql, "constraint album_artists\n", "constraint album_artists_unique\n")
		sql = strings.ReplaceAll(sql, "constraint album_artists ", "constraint album_artists_unique ")
		// PostgreSQL needs bigint for size columns (files > 2GB)
		sql = strings.ReplaceAll(sql, "size integer", "size bigint")
		sql = strings.ReplaceAll(sql, "total_size integer", "total_size bigint")
		// PostgreSQL enforces varchar(255) strictly; SQLite ignores it. text works the same.
		sql = strings.ReplaceAll(sql, "varchar(255)", "text")
	}
	return sql
}

// Use this in migrations that need to communicate something important (breaking changes, forced reindexes, etc...)
func notice(tx *sql.Tx, msg string) {
	if isDBInitialized(tx) {
		line := strings.Repeat("*", len(msg)+8)
		fmt.Printf("\n%s\nNOTICE: %s\n%s\n\n", line, msg, line)
	}
}

// Call this in migrations that requires a full rescan
func forceFullRescan(tx *sql.Tx) error {
	// If a full scan is required, most probably the query optimizer is outdated, so we run `analyze`.
	if conf.Server.DevOptimizeDB {
		_, err := tx.Exec(`ANALYZE;`)
		if err != nil {
			return err
		}
	}
	sql := fmt.Sprintf(`INSERT OR REPLACE into property (id, value) values ('%s', '1');`, consts.FullScanAfterMigrationFlagKey)
	if dialect.Current != nil && dialect.Current.Name() == "postgres" {
		sql = fmt.Sprintf(`INSERT INTO property (id, value) values ('%s', '1') ON CONFLICT (id) DO UPDATE SET value = '1';`, consts.FullScanAfterMigrationFlagKey)
	}
	_, err := tx.Exec(sql)
	return err
}

// 	sq := Update(r.tableName).
//		Set("last_scan_started_at", time.Now()).
//		Set("full_scan_in_progress", fullScan).
//		Where(Eq{"id": id})

var (
	once        sync.Once
	initialized bool
)

func isDBInitialized(tx *sql.Tx) bool {
	once.Do(func() {
		query := "select count(*) from property where id=$1"
		if dialect.Current == nil || dialect.Current.Name() != "postgres" {
			query = "select count(*) from property where id=?"
		}
		rows, err := tx.Query(query, consts.InitialSetupFlagKey)
		checkErr(err)
		initialized = checkCount(rows) > 0
	})
	return initialized
}

func checkCount(rows *sql.Rows) (count int) {
	for rows.Next() {
		err := rows.Scan(&count)
		checkErr(err)
	}
	return count
}

func checkErr(err error) {
	if err != nil {
		panic(err)
	}
}

type (
	execFunc      func() error
	execStmtFunc  func(stmt string) execFunc
	addColumnFunc func(tableName, columnName, columnType, defaultValue, initialValue string) execFunc
)

func createExecuteFunc(ctx context.Context, tx *sql.Tx) execStmtFunc {
	return func(stmt string) execFunc {
		return func() error {
			_, err := tx.ExecContext(ctx, adaptSQL(stmt))
			return err
		}
	}
}

// Hack way to add a new `not null` column to a table, setting the initial value for existing rows based on a
// SQL expression. It is done in 3 steps:
//  1. Add the column as nullable. Due to the way SQLite manipulates the DDL in memory, we need to add extra padding
//     to the default value to avoid truncating it when changing the column to not null
//  2. Update the column with the initial value
//  3. Change the column to not null with the default value
//
// Based on https://stackoverflow.com/a/25917323
func createAddColumnFunc(ctx context.Context, tx *sql.Tx) addColumnFunc {
	return func(tableName, columnName, columnType, defaultValue, initialValue string) execFunc {
		return func() error {
			adaptedColumnType := adaptSQL(" " + columnType)[1:]

			if dialect.Current != nil && dialect.Current.Name() == "postgres" {
				// PostgreSQL: add column with default, update, then set NOT NULL
				_, err := tx.ExecContext(ctx, fmt.Sprintf(
					`ALTER TABLE %s ADD COLUMN %s %s DEFAULT %s;`,
					tableName, columnName, adaptedColumnType, adaptSQL(defaultValue)))
				if err != nil {
					return err
				}
				_, err = tx.ExecContext(ctx, fmt.Sprintf(
					`UPDATE %s SET %s = %s WHERE %[2]s IS NULL;`,
					tableName, columnName, adaptSQL(initialValue)))
				if err != nil {
					return err
				}
				_, err = tx.ExecContext(ctx, fmt.Sprintf(
					`ALTER TABLE %s ALTER COLUMN %s SET NOT NULL;`,
					tableName, columnName))
				return err
			}

			// SQLite: use the PRAGMA hack
			// Format the `default null` value to have the same length as the final defaultValue
			finalLen := len(fmt.Sprintf(`%s not`, defaultValue))
			tempDefault := fmt.Sprintf(`default %s null`, strings.Repeat(" ", finalLen))
			_, err := tx.ExecContext(ctx, fmt.Sprintf(`
alter table %s add column %s %s %s;`, tableName, columnName, columnType, tempDefault))
			if err != nil {
				return err
			}
			_, err = tx.ExecContext(ctx, fmt.Sprintf(`
update %s set %s = %s where %[2]s is null;`, tableName, columnName, initialValue))
			if err != nil {
				return err
			}
			_, err = tx.ExecContext(ctx, fmt.Sprintf(`
PRAGMA writable_schema = on;
UPDATE sqlite_master
SET sql = replace(sql, '%[1]s %[2]s %[5]s', '%[1]s %[2]s default %[3]s not null')
WHERE type = 'table'
  AND name = '%[4]s';
PRAGMA writable_schema = off;
`, columnName, columnType, defaultValue, tableName, tempDefault))
			if err != nil {
				return err
			}
			return err
		}
	}
}
