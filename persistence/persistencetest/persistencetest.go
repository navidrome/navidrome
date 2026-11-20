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
	if _, err := db.Db().Exec("VACUUM INTO ?", baselinePath); err != nil {
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

// ResetBeforeEachSpec registers a top-level BeforeEach that calls Reset. Specs in Ordered
// containers share state on purpose, so they are skipped; their BeforeAll should call Reset.
func ResetBeforeEachSpec() bool {
	return ginkgo.BeforeEach(func() {
		if !ginkgo.CurrentSpecReport().IsInOrderedContainer {
			Reset()
		}
	})
}

// Tables lists the tables a restore rewrites. FTS shadow tables are excluded: their content
// tables' triggers keep them in sync, and writing them directly corrupts the index.
func Tables() ([]string, error) {
	return tables(context.Background(), db.Db())
}

type querier interface {
	QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error)
}

func tables(ctx context.Context, q querier) ([]string, error) {
	rows, err := q.QueryContext(ctx, "SELECT name FROM main.sqlite_master WHERE type='table' AND name NOT LIKE 'sqlite_%' AND name NOT LIKE '%_fts' AND name NOT LIKE '%_fts_%'")
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

// RestoreFrom replaces every table's rows with those in the SQLite file at path.
func RestoreFrom(path string) (err error) {
	ctx := context.Background()
	// ATTACH and PRAGMA foreign_keys are per connection, so everything must run on one.
	conn, err := db.Db().Conn(ctx)
	if err != nil {
		return err
	}
	defer conn.Close()

	exec := func(query string, args ...any) {
		if err != nil {
			return
		}
		if _, execErr := conn.ExecContext(ctx, query, args...); execErr != nil {
			err = fmt.Errorf("%s: %w", query, execErr)
		}
	}

	exec("PRAGMA foreign_keys = OFF")
	exec("ATTACH DATABASE ? AS snapshot", path)
	defer func() {
		_, _ = conn.ExecContext(ctx, "DETACH DATABASE snapshot")
		_, _ = conn.ExecContext(ctx, "PRAGMA foreign_keys = ON")
	}()
	if err != nil {
		return err
	}

	names, err := tables(ctx, conn)
	if err != nil {
		return err
	}
	exec("BEGIN")
	for _, table := range names {
		// Table names come from sqlite_master, not user input.
		exec(`DELETE FROM main."` + table + `"`)
		exec(`INSERT INTO main."` + table + `" SELECT * FROM snapshot."` + table + `"`)
	}
	if err != nil {
		_, _ = conn.ExecContext(ctx, "ROLLBACK")
		return err
	}
	exec("COMMIT")
	return err
}
