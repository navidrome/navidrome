package db

import (
	"context"
	"database/sql"
	"embed"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"slices"
	"time"

	"github.com/mattn/go-sqlite3"
	"github.com/navidrome/navidrome/conf"
	_ "github.com/navidrome/navidrome/db/migrations"
	"github.com/navidrome/navidrome/log"
	"github.com/navidrome/navidrome/utils/gg"
	"github.com/navidrome/navidrome/utils/hasher"
	"github.com/navidrome/navidrome/utils/singleton"
	"github.com/pressly/goose/v3"
)

var (
	Driver = "sqlite3"
	Path   string
)

//go:embed migrations/*.sql
var embedMigrations embed.FS

const (
	backupPrefix      = "navidrome_backup"
	migrationsFolder  = "migrations"
	backupRegexString = backupPrefix + "_(.+)\\.db"
)

var backupRegex = regexp.MustCompile(backupRegexString)

type DB interface {
	ReadDB() *sql.DB
	WriteDB() *sql.DB
	Close()

	Backup(ctx context.Context) error
	Restore(ctx context.Context, path string) error
}

type db struct {
	readDB  *sql.DB
	writeDB *sql.DB
}

func (d *db) ReadDB() *sql.DB {
	return d.readDB
}

func (d *db) WriteDB() *sql.DB {
	return d.writeDB
}

func (d *db) Close() {
	if err := d.readDB.Close(); err != nil {
		log.Error("Error closing read DB", err)
	}
	if err := d.writeDB.Close(); err != nil {
		log.Error("Error closing write DB", err)
	}
}

func backupPath(t *time.Time) string {
	return filepath.Join(
		conf.Server.BackupPath,
		fmt.Sprintf("%s_%s.db", backupPrefix, t.Format(time.RFC3339)),
	)
}

func (d *db) Backup(ctx context.Context) error {
	destPath := backupPath(gg.P(time.Now()))
	return d.backupOrRestore(ctx, true, destPath)
}

func (d *db) Restore(ctx context.Context, path string) error {
	return d.backupOrRestore(ctx, false, path)
}

func (d *db) backupOrRestore(ctx context.Context, isBackup bool, path string) error {
	// heavily inspired by https://codingrabbits.dev/posts/go_and_sqlite_backup_and_maybe_restore/
	backupDb, err := sql.Open(Driver+"_custom", path)
	if err != nil {
		return err
	}
	defer backupDb.Close()

	existingConn, err := d.writeDB.Conn(ctx)
	if err != nil {
		return err
	}
	defer existingConn.Close()

	backupConn, err := backupDb.Conn(ctx)
	if err != nil {
		return err
	}
	defer backupConn.Close()

	err = existingConn.Raw(func(existing any) error {
		return backupConn.Raw(func(backup any) error {
			var sourceOk, destOk bool
			var sourceConn, destConn *sqlite3.SQLiteConn

			if isBackup {
				sourceConn, sourceOk = existing.(*sqlite3.SQLiteConn)
				destConn, destOk = backup.(*sqlite3.SQLiteConn)
			} else {
				sourceConn, sourceOk = backup.(*sqlite3.SQLiteConn)
				destConn, destOk = existing.(*sqlite3.SQLiteConn)
			}

			if !sourceOk {
				return fmt.Errorf("error trying to convert source to sqlite connection")
			}
			if !destOk {
				return fmt.Errorf("error trying to convert destination to sqlite connection")
			}

			backupOp, err := destConn.Backup("main", sourceConn, "main")
			if err != nil {
				return fmt.Errorf("error starting sqlite backup: %w", err)
			}
			defer backupOp.Close()

			// Caution: -1 means that sqlite will hold a read lock until the operation finishes
			// This will lock out other writes that could happen at the same time
			done, err := backupOp.Step(-1)
			if !done {
				return fmt.Errorf("backup not done with step -1")
			}
			if err != nil {
				return fmt.Errorf("error during backup step: %w", err)
			}

			err = backupOp.Finish()
			if err != nil {
				return fmt.Errorf("error finishing backup: %w", err)
			}

			return nil
		})
	})

	if err == nil {
		files, err := os.ReadDir(conf.Server.BackupPath)
		if err != nil {
			return fmt.Errorf("unable to read database backup entries: %w", err)
		}

		times := []time.Time{}

		for _, file := range files {
			if !file.IsDir() {
				submatch := backupRegex.FindStringSubmatch(file.Name())
				if len(submatch) == 2 {
					timestamp, err := time.Parse(time.RFC3339, submatch[1])
					if err == nil {
						times = append(times, timestamp)
					}
				}
			}
		}

		slices.SortFunc(times, func(a, b time.Time) int {
			return b.Compare(a)
		})

		if len(times) > conf.Server.BackupCount && !conf.Server.BackupCountIgnore {
			var errs []error

			for _, time := range times[conf.Server.BackupCount:] {
				path := backupPath(&time)
				err = os.Remove(path)

				if err != nil {
					errs = append(errs, err)
				}
			}

			if len(errs) > 0 {
				log.Error(ctx, "Failed to delete one or more files", "errors", errors.Join(errs...))
			}
		}
	}

	return err
}

func Db() DB {
	return singleton.GetInstance(func() *db {
		sql.Register(Driver+"_custom", &sqlite3.SQLiteDriver{
			ConnectHook: func(conn *sqlite3.SQLiteConn) error {
				return conn.RegisterFunc("SEEDEDRAND", hasher.HashFunc(), false)
			},
		})

		Path = conf.Server.DbPath
		if Path == ":memory:" {
			Path = "file::memory:?cache=shared&_foreign_keys=on"
			conf.Server.DbPath = Path
		}
		log.Debug("Opening DataBase", "dbPath", Path, "driver", Driver)

		// Create a read database connection
		rdb, err := sql.Open(Driver+"_custom", Path)
		if err != nil {
			log.Fatal("Error opening read database", err)
		}
		rdb.SetMaxOpenConns(max(4, runtime.NumCPU()))

		// Create a write database connection
		wdb, err := sql.Open(Driver+"_custom", Path)
		if err != nil {
			log.Fatal("Error opening write database", err)
		}
		wdb.SetMaxOpenConns(1)

		return &db{
			readDB:  rdb,
			writeDB: wdb,
		}
	})
}

func Close() {
	log.Info("Closing Database")
	Db().Close()
}

func Init() func() {
	db := Db().WriteDB()

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

	return Close
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
