package migrations

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"sync"

	"github.com/navidrome/navidrome/consts"
)

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
	_, err := tx.Exec(`ANALYZE;`)
	if err != nil {
		return err
	}
	_, err = tx.Exec(fmt.Sprintf(`
INSERT OR REPLACE into property (id, value) values ('%s', '1');
`, consts.FullScanAfterMigrationFlagKey))
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
		rows, err := tx.Query("select count(*) from property where id=?", consts.InitialSetupFlagKey)
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
			_, err := tx.ExecContext(ctx, stmt)
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
