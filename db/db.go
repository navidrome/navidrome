package db

import (
	"context"
	"database/sql"
	"embed"
	"fmt"
	"runtime"

	"github.com/navidrome/navidrome/conf"
	"github.com/navidrome/navidrome/consts"
	"github.com/navidrome/navidrome/db/dialect"
	_ "github.com/navidrome/navidrome/db/migrations"
	"github.com/navidrome/navidrome/log"
	"github.com/navidrome/navidrome/utils/singleton"
	"github.com/pressly/goose/v3"
)

var (
	// Dialect is the goose dialect name (for backward compatibility)
	Dialect = "sqlite3"
	Driver  = "sqlite3_custom"
	Path    string
)

func init() {
	// Initialize default dialect (SQLite) for tests and early access
	dialect.Current = dialect.NewSQLite()
}

//go:embed migrations/*.sql
var embedMigrations embed.FS

const migrationsFolder = "migrations"

func Db() *sql.DB {
	return singleton.GetInstance(func() *sql.DB {
		initDialect()

		if err := dialect.Current.RegisterDriver(); err != nil {
			log.Fatal("Error registering database driver", err)
		}

		Path = dialect.Current.DSN()
		log.Debug("Opening DataBase", "dbPath", Path, "driver", Driver, "dialect", Dialect)

		db, err := sql.Open(Driver, Path)
		if err != nil {
			log.Fatal("Error opening database", err)
		}

		db.SetMaxOpenConns(max(4, runtime.NumCPU()))

		ctx := context.Background()
		if err := dialect.Current.ConfigureConnection(ctx, db); err != nil {
			log.Fatal("Error configuring database connection", err)
		}

		return db
	})
}

func initDialect() {
	switch conf.Server.DbDriver {
	case consts.DbDriverPostgres:
		dialect.Current = dialect.NewPostgres()
	default:
		dialect.Current = dialect.NewSQLite()
	}
	Dialect = dialect.Current.GooseDialect()
	Driver = dialect.Current.Driver()
}

func isSchemaEmpty(ctx context.Context, db *sql.DB) bool {
	return dialect.Current.IsSchemaEmpty(ctx, db)
}

func IsPostgres() bool {
	return conf.Server.DbDriver == consts.DbDriverPostgres
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

	// SQLite-specific: Disable foreign_keys to allow re-creating tables in migrations
	if Dialect == "sqlite3" {
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
	}

	goose.SetBaseFS(embedMigrations)
	err := goose.SetDialect(Dialect)
	if err != nil {
		log.Fatal(ctx, "Invalid DB dialect", "dialect", Dialect, err)
	}
	schemaEmpty := dialect.Current.IsSchemaEmpty(ctx, db)
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
		if err := dialect.Current.PostSchemaChange(ctx, db); err != nil {
			log.Error(ctx, "Error running post-schema-change optimization", err)
		}
	}

	return func() {
		Close(ctx)
	}
}

func Optimize(ctx context.Context) {
	if err := dialect.Current.Optimize(ctx, Db()); err != nil {
		log.Error(ctx, "Error running database optimization", err)
	}
}

type statusLogger struct{ numPending int }

func (*statusLogger) Fatalf(format string, v ...any) { log.Fatal(fmt.Sprintf(format, v...)) }
func (l *statusLogger) Printf(format string, v ...any) {
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


type logAdapter struct {
	ctx    context.Context
	silent bool
}

func (l *logAdapter) Fatal(v ...any) {
	log.Fatal(l.ctx, fmt.Sprint(v...))
}

func (l *logAdapter) Fatalf(format string, v ...any) {
	log.Fatal(l.ctx, fmt.Sprintf(format, v...))
}

func (l *logAdapter) Print(v ...any) {
	if !l.silent {
		log.Info(l.ctx, fmt.Sprint(v...))
	}
}

func (l *logAdapter) Println(v ...any) {
	if !l.silent {
		log.Info(l.ctx, fmt.Sprintln(v...))
	}
}

func (l *logAdapter) Printf(format string, v ...any) {
	if !l.silent {
		log.Info(l.ctx, fmt.Sprintf(format, v...))
	}
}
