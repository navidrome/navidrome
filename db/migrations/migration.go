package migrations

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"sync"

	"github.com/navidrome/navidrome/conf"
	"github.com/navidrome/navidrome/consts"
	"github.com/navidrome/navidrome/log"
)

// Use this in migrations that need to communicate something important (breaking changes, forced reindexes, etc...)
func notice(tx *sql.Tx, msg string) {
	if isDBInitialized(tx) {
		fmt.Printf(`
*************************************************************************************
NOTICE: %s
*************************************************************************************

`, msg)
	}
}

// Call this in migrations that requires a full rescan
func forceFullRescan(tx *sql.Tx) error {
	_, err := tx.Exec(`
delete from property where id like 'LastScan%';
update media_file set updated_at = '0001-01-01';
`)
	return err
}

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

// Hack way to add a column with a default value to a table. This is needed because SQLite does not support
// adding a column with a default value to a table that already has data, and there is no ALTER TABLE MODIFY COLUMN.
//
// Based on https://stackoverflow.com/a/25917323
//
// Add a new column to a table, setting the initial value for existing rows based on a SQL expression.
// It is done in 3 steps:
//  1. Add the column as nullable. Due the way SQLite manipulates the DDL in memory,
//     (we need to add extra padding to the default value to avoid truncating it when changing the column to not null)
//  2. Update the column with the initial value
//  3. Change the column to not null with the default value
func addColumn(ctx context.Context, tx *sql.Tx, tableName, columnName, columnType, defaultValue, initialValue string) error {
	log.Error(conf.Server.DbPath)
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

func execute(ctx context.Context, tx *sql.Tx, stmt string) error {
	_, err := tx.ExecContext(ctx, stmt)
	return err
}
