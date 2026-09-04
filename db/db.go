package db

import (
	"cmp"
	"context"
	"database/sql"
	"embed"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/mattn/go-sqlite3"
	"github.com/navidrome/navidrome/conf"
	_ "github.com/navidrome/navidrome/db/migrations"
	"github.com/navidrome/navidrome/db/pglite"
	"github.com/navidrome/navidrome/log"
	"github.com/navidrome/navidrome/utils/hasher"
	"github.com/navidrome/navidrome/utils/natural"
	"github.com/navidrome/navidrome/utils/singleton"
	"github.com/pressly/goose/v3"
)

// NaturalCollation sorts embedded numbers by value. It is registered on every
// connection, but only referenced when conf.Server.EnableNaturalSorting is on.
const NaturalCollation = "NATSORT"

var (
	Dialect = "sqlite3"
	Driver  = Dialect + "_custom"
	Path    string
)

//go:embed migrations/*.sql
var embedMigrations embed.FS

const migrationsFolder = "migrations"

// sql.Register panics if called twice, so guard it: the singleton instance can be reset
// (tests/benchmarks) and rebuilt, but the driver is process-global and registers only once.
var registerDriverOnce sync.Once

// embedded is the in-process PGlite instance when DbPath uses the pglite:// scheme (spike).
var embedded *pglite.PGlite

func Db() *sql.DB {
	return singleton.GetInstance(func() *sql.DB {
		registerDriverOnce.Do(func() {
			sql.Register(Driver, &sqlite3.SQLiteDriver{
				ConnectHook: func(conn *sqlite3.SQLiteConn) error {
					if err := conn.RegisterFunc("SEEDEDRAND", hasher.HashFunc(), false); err != nil {
						return err
					}
					return conn.RegisterCollation(NaturalCollation, natural.CompareFold)
				},
			})
		})
		Path = conf.Server.DbPath
		if dataDir, ok := strings.CutPrefix(Path, "pglite://"); ok {
			Dialect = "postgres"
			Driver = "pgx"
			tarball := cmp.Or(os.Getenv("ND_PGLITE_TARBALL"), "tmp/pglite-wasi-O2-fix.tar.gz")
			logFile, _ := os.Create(filepath.Join(dataDir, "..", "pglite.log"))
			socketDir := filepath.Dir(dataDir)
			pg, err := pglite.New(context.Background(), pglite.Config{
				DataDir: dataDir, Tarball: tarball, Stderr: logFile, SocketDir: socketDir,
			})
			if err != nil {
				log.Fatal("Error starting embedded PGlite", err)
			}
			embedded = pg
			log.Debug("Opening DataBase", "dbPath", Path, "driver", Driver, "dsn", pg.DSN())
			abs, _ := filepath.Abs(socketDir)
			log.Info("PGlite spike: connect with psql", "cmd", "PGSSLMODE=disable psql -h "+abs+" -U postgres postgres")
			db, err := sql.Open(Driver, pg.DSN())
			if err != nil {
				log.Fatal("Error opening database", err)
			}
			db.SetMaxOpenConns(1)
			return db
		}
		if isPostgres(Path) {
			Dialect = "postgres"
			Driver = "pgx"
			log.Debug("Opening DataBase", "dbPath", Path, "driver", Driver)
			db, err := sql.Open(Driver, Path)
			if err != nil {
				log.Fatal("Error opening database", err)
			}
			// One PGlite backend means one PG session; the bridge serializes clients onto it.
			db.SetMaxOpenConns(4)
			return db
		}
		if Path == ":memory:" {
			Path = "file::memory:?cache=shared&_foreign_keys=on"
			conf.Server.DbPath = Path
		} else {
			conf.Server.DataFolder.MustPath()
		}
		log.Debug("Opening DataBase", "dbPath", Path, "driver", Driver)
		db, err := sql.Open(Driver, Path)
		maxConns := conf.MaxOpenConns()
		if v, err := strconv.Atoi(os.Getenv("ND_TEST_MAXCONNS")); err == nil && v > 0 {
			maxConns = v // spike only: simulate the PGlite bridge's serialized session
		}
		db.SetMaxOpenConns(maxConns)
		if err != nil {
			log.Fatal("Error opening database", err)
		}
		return db
	})
}

func Close(ctx context.Context) {
	// Ignore cancellations when closing the DB
	ctx = context.WithoutCancel(ctx)

	log.Info(ctx, "Closing Database")
	err := Db().Close()
	if err != nil {
		log.Error(ctx, "Error closing Database", err)
	}
	if embedded != nil {
		_ = embedded.Close()
	}
}

// applySpikeSchema loads a crudely translated SQLite schema, one statement at a time, tolerating failures.
func applySpikeSchema(ctx context.Context, db *sql.DB, path string) {
	data, err := os.ReadFile(path)
	if err != nil {
		log.Fatal(ctx, "PGlite spike: cannot read schema", err)
	}
	var ok, failed int
	for _, stmt := range strings.Split(string(data), ";\n") {
		if strings.TrimSpace(stmt) == "" {
			continue
		}
		if _, err := db.ExecContext(ctx, stmt); err != nil {
			failed++
			log.Warn(ctx, "PGlite spike: schema statement failed", "stmt", strings.SplitN(strings.TrimSpace(stmt), "\n", 2)[0], err)
			continue
		}
		ok++
	}
	log.Warn(ctx, "PGlite spike: schema applied", "ok", ok, "failed", failed)
}

// isPostgres reports whether the DbPath is a Postgres connection URL (PGlite spike).
func isPostgres(path string) bool {
	return strings.HasPrefix(path, "postgres://") || strings.HasPrefix(path, "postgresql://")
}

func Init(ctx context.Context) func() {
	db := Db()

	if Dialect == "postgres" {
		var version string
		if err := db.QueryRowContext(ctx, "SELECT version()").Scan(&version); err != nil {
			log.Fatal(ctx, "PGlite spike: cannot reach database", err)
		}
		log.Warn(ctx, "PGlite spike: connected; migrations are SQLite-only and were SKIPPED", "version", version)
		if _, err := db.ExecContext(ctx, "CREATE TABLE IF NOT EXISTS property (id text PRIMARY KEY, value text)"); err != nil {
			log.Fatal(ctx, "PGlite spike: cannot create property table", err)
		}
		if schema := os.Getenv("ND_PGLITE_SCHEMA"); schema != "" {
			applySpikeSchema(ctx, db, schema)
		}
		return func() { Close(ctx) }
	}

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
		log.Debug(ctx, "Running ANALYZE after schema changes")
		err = optimizeAt(ctx, db, time.Now())
		if err != nil {
			log.Error(ctx, "Error running ANALYZE", err)
		}
	}

	return func() {
		Close(ctx)
	}
}

// ErrorCodes reports the SQLite result code and extended result code carried by err.
// The extended code is what distinguishes errors that share a message: "database is locked"
// is both SQLITE_BUSY, which busy_timeout retries, and SQLITE_BUSY_SNAPSHOT, which it never can.
func ErrorCodes(err error) (code, extended int, ok bool) {
	var se sqlite3.Error
	if !errors.As(err, &se) {
		return 0, 0, false
	}
	return int(se.Code), int(se.ExtendedCode), true
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
