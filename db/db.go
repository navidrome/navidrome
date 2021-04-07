package db

import (
	"database/sql"
	"os"
	"sync"

	_ "github.com/mattn/go-sqlite3"
	"github.com/navidrome/navidrome/conf"
	_ "github.com/navidrome/navidrome/db/migration"
	"github.com/navidrome/navidrome/log"
	"github.com/pressly/goose"
)

var (
	Driver = "sqlite3"
	Path   string
)

var (
	once sync.Once
	db   *sql.DB
)

func Db() *sql.DB {
	once.Do(func() {
		var err error
		Path = conf.Server.DbPath
		if Path == ":memory:" {
			Path = "file::memory:?cache=shared&_foreign_keys=on"
			conf.Server.DbPath = Path
		}
		log.Debug("Opening DataBase", "dbPath", Path, "driver", Driver)
		db, err = sql.Open(Driver, Path)
		if err != nil {
			panic(err)
		}
	})
	return db
}

func EnsureLatestVersion() {
	db := Db()

	// Disable foreign_keys to allow re-creating tables in migrations
	_, err := db.Exec("PRAGMA foreign_keys=off")
	defer func() {
		_, err := db.Exec("PRAGMA foreign_keys=on")
		if err != nil {
			log.Error("Error re-enabling foreign_keys", err)
		}
	}()
	if err != nil {
		log.Error("Error disabling foreign_keys", err)
	}

	err = goose.SetDialect(Driver)
	if err != nil {
		log.Error("Invalid DB driver", "driver", Driver, err)
		os.Exit(1)
	}
	err = goose.Run("up", db, "./")
	if err != nil {
		log.Error("Failed to apply new migrations", err)
		os.Exit(1)
	}
}
