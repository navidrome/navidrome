package db

import (
	"context"
	"database/sql"
	"embed"
	"fmt"
	"runtime"

	"github.com/mattn/go-sqlite3"
	"github.com/navidrome/navidrome/conf"
	_ "github.com/navidrome/navidrome/db/migrations"
	"github.com/navidrome/navidrome/log"
	"github.com/navidrome/navidrome/utils/hasher"
	"github.com/navidrome/navidrome/utils/singleton"
	"github.com/pressly/goose/v3"
)

var (
	Dialect = "sqlite3"
	Driver  = Dialect + "_custom"
	Path    string
)

//go:embed migrations/*.sql
var embedMigrations embed.FS

const migrationsFolder = "migrations"

func Db() *sql.DB {
	return singleton.GetInstance(func() *sql.DB {
		sql.Register(Driver, &sqlite3.SQLiteDriver{
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
		db, err := sql.Open(Driver, Path)
		db.SetMaxOpenConns(max(4, runtime.NumCPU()))
		if err != nil {
			log.Fatal("Error opening database", err)
		}
		_, err = db.Exec("PRAGMA optimize=0x10002")
		if err != nil {
			log.Error("Error applying PRAGMA optimize", err)
			return nil
		}
		return db
	})
}

func Close(ctx context.Context) {
	// Ignore cancellations when closing the DB
	ctx = context.WithoutCancel(ctx)

	// Run optimize before closing
	Optimize(ctx)

	log.Info(ctx, "Closing Database")
	err := Db().Close()
	if err != nil {
		log.Error(ctx, "Error closing Database", err)
	}
}

func Init(ctx context.Context) func() {
	db := Db()

	// Disable foreign_keys to allow re-creating tables in migrations
	_, err := db.ExecContext(ctx, "PRAGMA foreign_keys=off")
	defer func() {
		_, err := db.ExecContext(ctx, "PRAGMA foreign_keys=on")
		if err != nil {
			log.Error(ctx, "Error re-enabling foreign_keys", err)
		}
	}()
	if err != nil {
		log.Error(ctx, "Error disabling foreign_keys", err)
	}

	goose.SetBaseFS(embedMigrations)
	err = goose.SetDialect(Dialect)
	if err != nil {
		log.Fatal(ctx, "Invalid DB driver", "driver", Driver, err)
	}
	schemaEmpty := isSchemaEmpty(ctx, db)
	hasSchemaChanges := hasPendingMigrations(ctx, db, migrationsFolder)
	if !schemaEmpty && hasSchemaChanges {
		log.Info(ctx, "Upgrading DB Schema to latest version")
	}
	goose.SetLogger(&logAdapter{ctx: ctx, silent: schemaEmpty})
	err = goose.UpContext(ctx, db, migrationsFolder)
	if err != nil {
		log.Fatal(ctx, "Failed to apply new migrations", err)
	}

	if hasSchemaChanges {
		log.Debug(ctx, "Applying PRAGMA optimize after schema changes")
		_, err = db.ExecContext(ctx, "PRAGMA optimize")
		if err != nil {
			log.Error(ctx, "Error applying PRAGMA optimize", err)
		}
	}

	return func() {
		Close(ctx)
	}
}

// Optimize runs PRAGMA optimize on each connection in the pool
func Optimize(ctx context.Context) {
	numConns := Db().Stats().OpenConnections
	if numConns == 0 {
		log.Debug(ctx, "No open connections to optimize")
		return
	}
	log.Debug(ctx, "Optimizing open connections", "numConns", numConns)
	var conns []*sql.Conn
	for i := 0; i < numConns; i++ {
		conn, err := Db().Conn(ctx)
		conns = append(conns, conn)
		if err != nil {
			log.Error(ctx, "Error getting connection from pool", err)
			continue
		}
		_, err = conn.ExecContext(ctx, "PRAGMA optimize;")
		if err != nil {
			log.Error(ctx, "Error running PRAGMA optimize", err)
		}
	}

	// Return all connections to the Connection Pool
	for _, conn := range conns {
		conn.Close()
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

func hasPendingMigrations(ctx context.Context, db *sql.DB, folder string) bool {
	l := &statusLogger{}
	goose.SetLogger(l)
	err := goose.StatusContext(ctx, db, folder)
	if err != nil {
		log.Fatal(ctx, "Failed to check for pending migrations", err)
	}
	return l.numPending > 0
}

func isSchemaEmpty(ctx context.Context, db *sql.DB) bool {
	rows, err := db.QueryContext(ctx, "SELECT name FROM sqlite_master WHERE type='table' AND name='goose_db_version';") // nolint:rowserrcheck
	if err != nil {
		log.Fatal(ctx, "Database could not be opened!", err)
	}
	defer rows.Close()
	return !rows.Next()
}

type logAdapter struct {
	ctx    context.Context
	silent bool
}

func (l *logAdapter) Fatal(v ...interface{}) {
	log.Fatal(l.ctx, fmt.Sprint(v...))
}

func (l *logAdapter) Fatalf(format string, v ...interface{}) {
	log.Fatal(l.ctx, fmt.Sprintf(format, v...))
}

func (l *logAdapter) Print(v ...interface{}) {
	if !l.silent {
		log.Info(l.ctx, fmt.Sprint(v...))
	}
}

func (l *logAdapter) Println(v ...interface{}) {
	if !l.silent {
		log.Info(l.ctx, fmt.Sprintln(v...))
	}
}

func (l *logAdapter) Printf(format string, v ...interface{}) {
	if !l.silent {
		log.Info(l.ctx, fmt.Sprintf(format, v...))
	}
}
