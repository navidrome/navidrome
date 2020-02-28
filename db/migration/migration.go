package migration

import (
	"database/sql"
	"fmt"
	"sync"

	"github.com/deluan/navidrome/consts"
)

// Use this in migrations that need to communicate something important (braking changes, forced reindexes, etc...)
func notice(tx *sql.Tx, msg string) {
	if isDBInitialized(tx) {
		fmt.Printf(`
*************************************************************************************
NOTICE: %s
*************************************************************************************

`, msg)
	}
}

var once sync.Once

func isDBInitialized(tx *sql.Tx) (initialized bool) {
	once.Do(func() {
		rows, err := tx.Query("select count(*) from property where id='" + consts.InitialSetupFlagKey + "'")
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
