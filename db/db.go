package db

import (
	"database/sql"
	"embed"
	"fmt"

	_ "github.com/mattn/go-sqlite3"
	"github.com/navidrome/navidrome/conf"
	_ "github.com/navidrome/navidrome/db/migration"
	"github.com/navidrome/navidrome/log"
	"github.com/navidrome/navidrome/utils/singleton"
	"github.com/pressly/goose/v3"
)

var (
	Driver = "sqlite3"
	Path   string
)

//go:embed migration/*.sql
var embedMigrations embed.FS

const migrationsFolder = "migration"

func Db() *sql.DB {
	return singleton.GetInstance(func() *sql.DB {
		Path = conf.Server.DbPath
		if Path == ":memory:" {
			Path = "file::memory:?cache=shared&_foreign_keys=on"
			conf.Server.DbPath = Path
		}
		log.Debug("Opening DataBase", "dbPath", Path, "driver", Driver)
		instance, err := sql.Open(Driver, Path)
		if err != nil {
			panic(err)
		}
		return instance
	})
}

func Close() error {
	log.Info("Closing Database")
	return Db().Close()
}

func Init() {
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

	gooseLogger := &logAdapter{silent: isSchemaEmpty(db)}
	goose.SetBaseFS(embedMigrations)

	err = goose.SetDialect(Driver)
	if err != nil {
		log.Fatal("Invalid DB driver", "driver", Driver, err)
	}
	if !isSchemaEmpty(db) && hasPendingMigrations(db, migrationsFolder) {
		log.Info("Upgrading DB Schema to latest version")
	}
	goose.SetLogger(gooseLogger)
	err = goose.Up(db, migrationsFolder)
	if err != nil {
		log.Fatal("Failed to apply new migrations", err)
	}
}

type statusLogger struct{ numPending int }

func (*statusLogger) Fatalf(format string, v ...interface{}) { log.Fatal(fmt.Sprintf(format, v...)) }
func (l *statusLogger) Printf(format string, v ...interface{}) {
	if len(v) < 1 {
		return
	}
	if v0, ok := v[0].(string); !ok {
		return
	} else if v0 == "Pending" {
		l.numPending++
	}
}

func hasPendingMigrations(db *sql.DB, folder string) bool {
	l := &statusLogger{}
	goose.SetLogger(l)
	err := goose.Status(db, folder)
	if err != nil {
		log.Fatal("Failed to check for pending migrations", err)
	}
	return l.numPending > 0
}

func isSchemaEmpty(db *sql.DB) bool {
	rows, err := db.Query("SELECT name FROM sqlite_master WHERE type='table' AND name='goose_db_version';") // nolint:rowserrcheck
	if err != nil {
		log.Fatal("Database could not be opened!", err)
	}
	defer rows.Close()
	return !rows.Next()
}

type logAdapter struct {
	silent bool
}

func (l *logAdapter) Fatal(v ...interface{}) {
	log.Fatal(fmt.Sprint(v...))
}

func (l *logAdapter) Fatalf(format string, v ...interface{}) {
	log.Fatal(fmt.Sprintf(format, v...))
}

func (l *logAdapter) Print(v ...interface{}) {
	if !l.silent {
		log.Info(fmt.Sprint(v...))
	}
}

func (l *logAdapter) Println(v ...interface{}) {
	if !l.silent {
		log.Info(fmt.Sprintln(v...))
	}
}

func (l *logAdapter) Printf(format string, v ...interface{}) {
	if !l.silent {
		log.Info(fmt.Sprintf(format, v...))
	}
}
