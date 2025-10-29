package db

import (
	"context"
	"database/sql"
	"embed"
	"fmt"
	"path/filepath"
	"strings"
	"time"

	embeddedpostgres "github.com/fergusstrange/embedded-postgres"
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/navidrome/navidrome/conf"
	_ "github.com/navidrome/navidrome/db/migrations"
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

var postgresInstance *embeddedpostgres.EmbeddedPostgres

func Db() *sql.DB {
	return singleton.GetInstance(func() *sql.DB {
		start := time.Now()
		log.Info("Starting Embedded Postgres...")
		postgresInstance = embeddedpostgres.NewDatabase(
			embeddedpostgres.
				DefaultConfig().
				Port(5432).
				//Password(password).
				Logger(&logAdapter{ctx: context.Background()}).
				DataPath(filepath.Join(conf.Server.DataFolder, "postgres")).
				StartParameters(map[string]string{
					"unix_socket_directories": "/tmp",
					"unix_socket_permissions": "0700",
				}).
				BinariesPath(filepath.Join(conf.Server.CacheFolder, "postgres")),
		)
		if err := postgresInstance.Start(); err != nil {
			if !strings.Contains(err.Error(), "already listening on port") {
				_ = postgresInstance.Stop()
				log.Fatal("Failed to start embedded Postgres", err)
			}
			log.Info("Server already running on port 5432, assuming it's our embedded Postgres", "elapsed", time.Since(start))
		} else {
			log.Info("Embedded Postgres started", "elapsed", time.Since(start))
		}

		// Create the navidrome database if it doesn't exist
		adminPath := "postgresql://postgres:postgres@/postgres?sslmode=disable&host=/tmp"
		adminDB, err := sql.Open(Driver, adminPath)
		if err != nil {
			_ = postgresInstance.Stop()
			log.Fatal("Error connecting to admin database", err)
		}
		defer adminDB.Close()

		// Check if navidrome database exists, create if not
		var exists bool
		err = adminDB.QueryRow("SELECT EXISTS(SELECT 1 FROM pg_database WHERE datname = 'navidrome')").Scan(&exists)
		if err != nil {
			_ = postgresInstance.Stop()
			log.Fatal("Error checking if database exists", err)
		}
		if !exists {
			log.Info("Creating navidrome database...")
			_, err = adminDB.Exec("CREATE DATABASE navidrome")
			if err != nil {
				_ = postgresInstance.Stop()
				log.Fatal("Error creating navidrome database", err)
			}
		}

		// TODO: Implement seeded random function
		//sql.Register(Driver, &sqlite3.SQLiteDriver{
		//	ConnectHook: func(conn *sqlite3.SQLiteConn) error {
		//		return conn.RegisterFunc("SEEDEDRAND", hasher.HashFunc(), false)
		//	},
		//})
		//Path = conf.Server.DbPath
		// Ensure client does not attempt TLS when connecting to the embedded Postgres
		// and avoid shadowing the package-level Path variable.
		Path = "postgresql://postgres:postgres@/navidrome?sslmode=disable&host=/tmp"
		log.Debug("Opening DataBase", "dbPath", Path, "driver", Driver)
		db, err := sql.Open(Driver, Path)
		//db.SetMaxOpenConns(max(4, runtime.NumCPU()))
		if err != nil {
			_ = postgresInstance.Stop()
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
	if postgresInstance != nil {
		err = postgresInstance.Stop()
		if err != nil {
			log.Error(ctx, "Error stopping embedded Postgres", err)
		}
	}
}

func Init(ctx context.Context) func() {
	db := Db()

	goose.SetBaseFS(embedMigrations)
	err := goose.SetDialect(Dialect)
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

	return func() {
		Close(ctx)
	}
}

type statusLogger struct{ numPending int }

func (*statusLogger) Fatalf(format string, v ...interface{}) { log.Fatal(fmt.Sprintf(format, v...)) }
func (l *statusLogger) Printf(format string, v ...interface{}) {
	// format is part of the goose logger signature; reference it to avoid linter warnings
	_ = format
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
	rows, err := db.QueryContext(ctx, "SELECT tablename FROM pg_tables WHERE schemaname = 'public' AND tablename = 'goose_db_version';") // nolint:rowserrcheck
	if err != nil {
		log.Fatal(ctx, "Database could not be opened!", err)
	}
	defer func() {
		if cerr := rows.Close(); cerr != nil {
			log.Error(ctx, "Error closing rows", cerr)
		}
	}()
	return !rows.Next()
}

type logAdapter struct {
	ctx    context.Context
	silent bool
}

func (l *logAdapter) Write(p []byte) (n int, err error) {
	log.Debug(l.ctx, string(p))
	return len(p), nil
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
