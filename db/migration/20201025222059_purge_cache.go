package migration

import (
	"database/sql"
	"os"
	"path/filepath"

	"github.com/deluan/navidrome/conf"
	"github.com/pressly/goose"
)

func init() {
	goose.AddMigration(Up20201025222059, Down20201025222059)
}

func Up20201025222059(tx *sql.Tx) error {
	cacheFolder := filepath.Join(conf.Server.DataFolder, "cache")
	notice(tx, "Purging all cache entries, as the format of the cache changed.")
	return os.RemoveAll(cacheFolder)
}

func Down20201025222059(tx *sql.Tx) error {
	// This code is executed when the migration is rolled back.
	return nil
}
