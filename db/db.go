package db

import (
	"database/sql"
	"os"
	"sync"

	"github.com/deluan/navidrome/conf"
	_ "github.com/deluan/navidrome/db/migrations"
	"github.com/deluan/navidrome/log"
	_ "github.com/mattn/go-sqlite3"
	"github.com/pressly/goose"
)

var (
	once   sync.Once
	Driver = "sqlite3"
	Path   string
)

func Init() {
	once.Do(func() {
		Path = conf.Server.DbPath
		if Path == ":memory:" {
			Path = "file::memory:?cache=shared"
			conf.Server.DbPath = Path
		}
		log.Debug("Opening DataBase", "dbPath", Path, "driver", Driver)
	})
}

func EnsureLatestVersion() {
	Init()
	db, err := sql.Open(Driver, Path)
	defer db.Close()
	if err != nil {
		log.Error("Failed to open DB", err)
		os.Exit(1)
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
