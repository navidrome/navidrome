package db

import (
	"database/sql"
	"fmt"
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
	goose.SetLogger(&logAdapter{})

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

type logAdapter struct{}

func (l *logAdapter) Fatal(v ...interface{}) {
	log.Error(fmt.Sprint(v...))
	os.Exit(-1)
}

func (l *logAdapter) Fatalf(format string, v ...interface{}) {
	log.Error(fmt.Sprintf(format, v...))
	os.Exit(-1)
}

func (l *logAdapter) Print(v ...interface{}) {
	log.Info(fmt.Sprint(v...))
}

func (l *logAdapter) Println(v ...interface{}) {
	log.Info(fmt.Sprintln(v...))
}

func (l *logAdapter) Printf(format string, v ...interface{}) {
	log.Info(fmt.Sprintf(format, v...))
}
