package dialect

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/mattn/go-sqlite3"
	"github.com/navidrome/navidrome/conf"
	"github.com/navidrome/navidrome/log"
	"github.com/navidrome/navidrome/utils/hasher"
)

type SQLiteDialect struct {
	driverRegistered bool
}

func NewSQLite() *SQLiteDialect {
	return &SQLiteDialect{}
}

func (d *SQLiteDialect) Name() string {
	return "sqlite3"
}

func (d *SQLiteDialect) Driver() string {
	return "sqlite3_custom"
}

func (d *SQLiteDialect) DSN() string {
	path := conf.Server.DbPath
	if path == ":memory:" {
		return "file::memory:?cache=shared&_foreign_keys=on"
	}
	return path
}

func (d *SQLiteDialect) RegisterDriver() error {
	if d.driverRegistered {
		return nil
	}
	sql.Register(d.Driver(), &sqlite3.SQLiteDriver{
		ConnectHook: func(conn *sqlite3.SQLiteConn) error {
			return conn.RegisterFunc("SEEDEDRAND", hasher.HashFunc(), false)
		},
	})
	d.driverRegistered = true
	return nil
}

func (d *SQLiteDialect) ConfigureConnection(ctx context.Context, db *sql.DB) error {
	if conf.Server.DevOptimizeDB {
		_, err := db.Exec("PRAGMA optimize=0x10002")
		if err != nil {
			log.Error(ctx, "Error applying PRAGMA optimize", err)
			return err
		}
	}
	return nil
}

func (d *SQLiteDialect) Placeholder(index int) string {
	return "?"
}

func (d *SQLiteDialect) CaseInsensitiveComparison(column, value string) string {
	return fmt.Sprintf("%s LIKE %s", column, value)
}

func (d *SQLiteDialect) RandomFunc() string {
	return "random()"
}

func (d *SQLiteDialect) SeededRandomFunc(seedKey, idColumn string) string {
	return fmt.Sprintf("SEEDEDRAND('%s', %s)", seedKey, idColumn)
}

func (d *SQLiteDialect) IsSchemaEmpty(ctx context.Context, db *sql.DB) bool {
	rows, err := db.QueryContext(ctx, "SELECT name FROM sqlite_master WHERE type='table' AND name='goose_db_version';")
	if err != nil {
		log.Fatal(ctx, "Database could not be opened!", err)
	}
	defer rows.Close()
	return !rows.Next()
}

func (d *SQLiteDialect) GooseDialect() string {
	return "sqlite3"
}

func (d *SQLiteDialect) Optimize(ctx context.Context, db *sql.DB) error {
	if !conf.Server.DevOptimizeDB {
		return nil
	}
	numConns := db.Stats().OpenConnections
	if numConns == 0 {
		log.Debug(ctx, "No open connections to optimize")
		return nil
	}
	log.Debug(ctx, "Optimizing SQLite connections", "numConns", numConns)
	var conns []*sql.Conn
	for i := 0; i < numConns; i++ {
		conn, err := db.Conn(ctx)
		if err != nil {
			log.Error(ctx, "Error getting connection from pool", err)
			continue
		}
		conns = append(conns, conn)
		_, err = conn.ExecContext(ctx, "PRAGMA optimize;")
		if err != nil {
			log.Error(ctx, "Error running PRAGMA optimize", err)
		}
	}
	// Return all connections to the Connection Pool
	for _, conn := range conns {
		conn.Close()
	}
	return nil
}

func (d *SQLiteDialect) PostSchemaChange(ctx context.Context, db *sql.DB) error {
	if conf.Server.DevOptimizeDB {
		log.Debug(ctx, "Applying PRAGMA optimize after schema changes")
		_, err := db.ExecContext(ctx, "PRAGMA optimize")
		if err != nil {
			log.Error(ctx, "Error applying PRAGMA optimize", err)
			return err
		}
	}
	return nil
}

func (d *SQLiteDialect) Backup(ctx context.Context, db *sql.DB, destPath string) error {
	return sqliteBackup(ctx, db, destPath, true)
}

func (d *SQLiteDialect) Restore(ctx context.Context, db *sql.DB, sourcePath string) error {
	return sqliteBackup(ctx, db, sourcePath, false)
}

// sqliteBackup performs backup or restore operations using SQLite's native backup API.
func sqliteBackup(ctx context.Context, db *sql.DB, path string, isBackup bool) error {
	existingConn, err := db.Conn(ctx)
	if err != nil {
		return fmt.Errorf("getting existing connection: %w", err)
	}
	defer existingConn.Close()

	backupDb, err := sql.Open("sqlite3_custom", path)
	if err != nil {
		return fmt.Errorf("opening backup database in '%s': %w", path, err)
	}
	defer backupDb.Close()

	backupConn, err := backupDb.Conn(ctx)
	if err != nil {
		return fmt.Errorf("getting backup connection: %w", err)
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

			// -1 means that sqlite will hold a read lock until the operation finishes
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

	return err
}
