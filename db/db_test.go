package db_test

import (
	"context"
	"database/sql"
	"testing"

	"github.com/navidrome/navidrome/db"
	"github.com/navidrome/navidrome/log"
	"github.com/navidrome/navidrome/tests"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestDB(t *testing.T) {
	tests.Init(t, false)
	log.SetLevel(log.LevelFatal)
	RegisterFailHandler(Fail)
	RunSpecs(t, "DB Suite")
}

var _ = Describe("Optimize", func() {
	It("replaces poisoned planner statistics with full-quality ones", func() {
		ctx := context.Background()
		database, err := sql.Open(db.Dialect, "file::memory:")
		Expect(err).ToNot(HaveOccurred())
		defer database.Close()

		// A low-cardinality index over >2000 same-valued rows: exactly the shape PRAGMA
		// optimize's budget-limited ANALYZE writes wrong stats for (the "N 2001" artifact).
		_, err = database.Exec("create table analyze_probe(id integer primary key, flag int)")
		Expect(err).ToNot(HaveOccurred())
		_, err = database.Exec(`insert into analyze_probe(flag)
			with recursive s(x) as (select 1 union all select x+1 from s where x < 3000)
			select 0 from s`)
		Expect(err).ToNot(HaveOccurred())
		_, err = database.Exec("create index probe_flag on analyze_probe(flag)")
		Expect(err).ToNot(HaveOccurred())

		// Seed a poisoned stat: claims the index narrows 3000 rows to ~50.
		_, err = database.Exec("analyze")
		Expect(err).ToNot(HaveOccurred())
		_, err = database.Exec("update sqlite_stat1 set stat='3000 50' where idx='probe_flag'")
		Expect(err).ToNot(HaveOccurred())

		db.OptimizeDB(ctx, database)

		var stat string
		err = database.QueryRow("select stat from sqlite_stat1 where idx='probe_flag'").Scan(&stat)
		Expect(err).ToNot(HaveOccurred())
		// A full ANALYZE sees all 3000 rows share one value: avg rows per key = row count.
		Expect(stat).To(Equal("3000 3000"))
	})
})

var _ = Describe("IsSchemaEmpty", func() {
	var database *sql.DB
	var ctx context.Context
	BeforeEach(func() {
		ctx = context.Background()
		path := "file::memory:"
		database, _ = sql.Open(db.Dialect, path)
	})

	It("returns false if the goose metadata table is found", func() {
		_, err := database.Exec("create table goose_db_version (id primary key);")
		Expect(err).ToNot(HaveOccurred())
		Expect(db.IsSchemaEmpty(ctx, database)).To(BeFalse())
	})

	It("returns true if the schema is brand new", func() {
		Expect(db.IsSchemaEmpty(ctx, database)).To(BeTrue())
	})
})
