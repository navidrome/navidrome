package migrations

import (
	"database/sql"
	"fmt"
	"sync"

	"github.com/navidrome/navidrome/consts"
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

func isDBInitialized(tx *sql.Tx) bool {
	return sync.OnceValue(func() bool {
		rows, err := tx.Query("select count(*) from property where id=?", consts.InitialSetupFlagKey)
		checkErr(err)
		return checkCount(rows) > 0
	})()
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
