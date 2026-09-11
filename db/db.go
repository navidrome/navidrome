package db

import (
	"context"
	"database/sql"
	"embed"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/navidrome/navidrome/conf"
	"github.com/navidrome/navidrome/db/pglite"
	"github.com/navidrome/navidrome/log"
	"github.com/navidrome/navidrome/utils/singleton"
	"github.com/pressly/goose/v3"
)

var (
	Dialect = "postgres"
	Driver  = "pgx"
	Path    string
)

//go:embed migrations/*.sql
var embedMigrations embed.FS

const migrationsFolder = "migrations"

// embedded is the in-process PGlite instance backing Db().
var embedded *pglite.PGlite

func Db() *sql.DB {
	return singleton.GetInstance(func() *sql.DB {
		Path = conf.Server.DbPath
		dataDir, ok := strings.CutPrefix(Path, "pglite://")
		if !ok {
			log.Fatal("DbPath must be a pglite://<dir> location in this build", "dbPath", Path)
		}
		socketDir := filepath.Dir(dataDir)
		logFile, _ := os.Create(filepath.Join(socketDir, "pglite.log"))
		pg, err := pglite.New(context.Background(), pglite.Config{
			DataDir: dataDir, Stderr: logFile, SocketDir: socketDir,
			CacheDir: filepath.Join(conf.Server.CacheFolder.String(), "pglite"),
		})
		if err != nil {
			log.Fatal("Error starting embedded PGlite", err)
		}
		embedded = pg
		db, err := pg.OpenDB()
		if err != nil {
			log.Fatal("Error opening database", err)
		}
		// One PGlite backend means one PG session; the bridge serializes clients onto it.
		// Keep every connection idle rather than churning handshakes through the backend.
		db.SetMaxOpenConns(4)
		db.SetMaxIdleConns(4)
		db.SetConnMaxLifetime(0)
		abs, _ := filepath.Abs(socketDir)
		log.Info("Embedded PGlite ready", "psql", "PGPASSWORD=x PGSSLMODE=disable psql -h "+abs+" -U postgres navidrome")
		return db
	})
}

func Close(ctx context.Context) {
	// Ignore cancellations when closing the DB
	ctx = context.WithoutCancel(ctx)

	log.Info(ctx, "Closing Database")
	if err := Db().Close(); err != nil {
		log.Error(ctx, "Error closing Database", err)
	}
	if embedded != nil {
		_ = embedded.Close()
	}
}

func Init(ctx context.Context) func() {
	db := Db()

	goose.SetBaseFS(embedMigrations)
	if err := goose.SetDialect(Dialect); err != nil {
		log.Fatal(ctx, "Invalid DB driver", "driver", Driver, err)
	}
	schemaEmpty := isSchemaEmpty(ctx, db)
	hasSchemaChanges := hasPendingMigrations(ctx, db, migrationsFolder)
	if !schemaEmpty && hasSchemaChanges {
		log.Info(ctx, "Upgrading DB Schema to latest version")
	}
	goose.SetLogger(&logAdapter{ctx: ctx, silent: schemaEmpty})
	if err := goose.UpContext(ctx, db, migrationsFolder); err != nil {
		log.Fatal(ctx, "Failed to apply new migrations", err)
	}

	if hasSchemaChanges {
		log.Debug(ctx, "Running ANALYZE after schema changes")
		if err := optimizeAt(ctx, db, time.Now()); err != nil {
			log.Error(ctx, "Error running ANALYZE", err)
		}
	}

	return func() {
		Close(ctx)
	}
}

// SQLState reports the PostgreSQL error code carried by err, if any.
func SQLState(err error) (string, bool) {
	var pgErr *pgconn.PgError
	if !errors.As(err, &pgErr) {
		return "", false
	}
	return pgErr.Code, true
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

func isSchemaEmpty(ctx context.Context, db *sql.DB) bool {
	var found *string
	err := db.QueryRowContext(ctx, "SELECT to_regclass('public.goose_db_version')::text").Scan(&found)
	if err != nil {
		log.Fatal(ctx, "Database could not be opened!", err)
	}
	return found == nil
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
