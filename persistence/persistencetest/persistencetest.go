// Package persistencetest gives a test suite a real, migrated SQLite database that is restored to
// its post-migration state before each spec. It must only be imported from test code.
package persistencetest

import (
	"context"
	"database/sql"
	"fmt"
	"path/filepath"
	"strings"
	"testing"

	"github.com/navidrome/navidrome/conf"
	"github.com/navidrome/navidrome/consts"
	"github.com/navidrome/navidrome/db"
	"github.com/navidrome/navidrome/persistence"
	"github.com/navidrome/navidrome/utils/singleton"
	"github.com/onsi/ginkgo/v2"
)

var baselinePath string

// Init migrates a fresh database for the suite and snapshots it as the baseline Reset restores.
// Call it in the suite's TestXxx, before RunSpecs: the database is a process-wide singleton.
func Init(t testing.TB) {
	t.Helper()
	dir := t.TempDir()
	_, options, _ := strings.Cut(consts.DefaultDbPath, "?")
	singleton.DeleteInstance[*sql.DB]()
	conf.Update(func(c *conf.Options) { c.DbPath = filepath.Join(dir, "test.db") + "?" + options })
	db.Init(t.Context())

	baselinePath = filepath.Join(dir, "baseline.db")
	if err := Snapshot(baselinePath); err != nil {
		t.Fatalf("snapshotting baseline database: %v", err)
	}
	t.Cleanup(func() {
		db.Close(context.Background())
		singleton.DeleteInstance[*sql.DB]()
	})
}

// Reset restores the database to the baseline Init saved, including rows seeded by migrations.
func Reset() {
	ginkgo.GinkgoHelper()
	if baselinePath == "" {
		ginkgo.Fail("persistencetest.Reset called before persistencetest.Init")
	}
	if err := RestoreFrom(baselinePath); err != nil {
		ginkgo.Fail("restoring baseline database: " + err.Error())
	}
}

// ResetBeforeEachSpec registers a top-level BeforeEach that calls Reset before every spec, and
// once before each Ordered container, whose specs share state on purpose.
func ResetBeforeEachSpec() bool {
	return ginkgo.BeforeEach(ginkgo.OncePerOrdered, Reset)
}

// tables lists the tables a restore rewrites. FTS shadow tables are excluded: their content
// tables' triggers keep them in sync, and writing them directly corrupts the index.
func tables(ctx context.Context, conn *sql.Conn) ([]string, error) {
	rows, err := conn.QueryContext(ctx, "SELECT name FROM main.sqlite_master WHERE type='table' AND name NOT LIKE 'sqlite_%' AND name NOT LIKE '%_fts' AND name NOT LIKE '%_fts_%'")
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var names []string
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return nil, err
		}
		names = append(names, name)
	}
	return names, rows.Err()
}

// Snapshot writes a copy of the database to path, which must not exist, for RestoreFrom to restore.
func Snapshot(path string) error {
	_, err := db.Db().Exec("VACUUM INTO ?", path)
	return err
}

// RestoreFrom replaces every table's rows with those in the SQLite file at path.
func RestoreFrom(path string) error {
	ctx := context.Background()
	// ATTACH and PRAGMA foreign_keys are per connection and refused inside a transaction, so they
	// run on one pinned connection, before it.
	conn, err := db.Db().Conn(ctx)
	if err != nil {
		return err
	}
	defer conn.Close()
	if _, err := conn.ExecContext(ctx, "PRAGMA foreign_keys = OFF"); err != nil {
		return err
	}
	defer func() { _, _ = conn.ExecContext(ctx, "PRAGMA foreign_keys = ON") }()
	if _, err := conn.ExecContext(ctx, "ATTACH DATABASE ? AS snapshot", path); err != nil {
		return err
	}
	defer func() { _, _ = conn.ExecContext(ctx, "DETACH DATABASE snapshot") }()

	names, err := tables(ctx, conn)
	if err != nil {
		return err
	}
	// A raw BEGIN on the pinned connection skips the per-statement cost of database/sql's Tx.
	if _, err := conn.ExecContext(ctx, "BEGIN"); err != nil {
		return err
	}
	committed := false
	defer func() {
		if !committed {
			_, _ = conn.ExecContext(ctx, "ROLLBACK")
		}
	}()
	for _, table := range names {
		// Table names come from sqlite_master, not user input.
		for _, query := range []string{
			`DELETE FROM main."` + table + `"`,
			`INSERT INTO main."` + table + `" SELECT * FROM snapshot."` + table + `"`,
		} {
			if _, err := conn.ExecContext(ctx, query); err != nil {
				return fmt.Errorf("%s: %w", query, err)
			}
		}
	}
	if _, err := conn.ExecContext(ctx, "COMMIT"); err != nil {
		return err
	}
	committed = true
	persistence.ClearLibraryPathCache()
	return nil
}
