package cmd

import (
	"context"
	"database/sql"
	"os"
	"path/filepath"
	"strings"

	"github.com/navidrome/navidrome/db"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("doctor", func() {
	var (
		ctx      context.Context
		dbPath   string
		database *sql.DB
		out      *strings.Builder
		reopen   func()
	)

	// A file-backed DB so specs can corrupt raw pages; a table named like a real FTS
	// search table so IsFTSCorruptionOnly matches, plus a parent/child pair for FK checks.
	BeforeEach(func() {
		ctx = context.Background()
		dbPath = filepath.Join(GinkgoT().TempDir(), "doctor.db")
		reopen = func() {
			var err error
			database, err = sql.Open(db.Dialect, dbPath)
			Expect(err).ToNot(HaveOccurred())
			database.SetMaxOpenConns(1)
		}
		reopen()
		DeferCleanup(func() { _ = database.Close() })

		for _, stmt := range []string{
			`create virtual table media_file_fts using fts5(title, content='', content_rowid='rowid')`,
			`insert into media_file_fts(rowid, title) values (1, 'teenage lobotomy'), (2, 'rockaway beach')`,
			`create table library(id integer primary key)`,
			`create table media_file(id integer primary key, library_id integer references library(id))`,
		} {
			_, err := database.ExecContext(ctx, stmt)
			Expect(err).ToNot(HaveOccurred())
		}
		out = &strings.Builder{}
	})

	It("reports a healthy database", func() {
		Expect(doctor(ctx, database, out)).To(BeTrue())
		Expect(out.String()).To(ContainSubstring("Database is healthy."))
	})

	It("points to 'search rebuild' when corruption is limited to the search index", func() {
		_, err := database.ExecContext(ctx,
			`update media_file_fts_data set block = x'deadbeefdeadbeef' where id > 1`)
		Expect(err).ToNot(HaveOccurred())

		Expect(doctor(ctx, database, out)).To(BeFalse())
		Expect(out.String()).To(ContainSubstring("navidrome search rebuild"))
	})

	It("points to a backup restore when corruption is not limited to the search index", func() {
		_, err := database.ExecContext(ctx,
			`insert into library(id)
			 with recursive s(x) as (select 1 union all select x+1 from s where x < 200)
			 select x from s`)
		Expect(err).ToNot(HaveOccurred())
		var rootPage, pageSize int64
		Expect(database.QueryRowContext(ctx,
			`select rootpage from sqlite_master where name = 'library'`).Scan(&rootPage)).To(Succeed())
		Expect(database.QueryRowContext(ctx, `pragma page_size`).Scan(&pageSize)).To(Succeed())
		Expect(database.Close()).To(Succeed())
		f, err := os.OpenFile(dbPath, os.O_WRONLY, 0600)
		Expect(err).ToNot(HaveOccurred())
		_, err = f.WriteAt([]byte{0xde, 0xad, 0xbe, 0xef, 0xde, 0xad, 0xbe, 0xef}, (rootPage-1)*pageSize+40)
		Expect(err).ToNot(HaveOccurred())
		Expect(f.Close()).To(Succeed())
		reopen()

		Expect(doctor(ctx, database, out)).To(BeFalse())
		Expect(out.String()).To(ContainSubstring("backup restore"))
		Expect(out.String()).ToNot(ContainSubstring("search rebuild"))
	})

	It("reports foreign key violations", func() {
		_, err := database.ExecContext(ctx, `pragma foreign_keys = off`)
		Expect(err).ToNot(HaveOccurred())
		_, err = database.ExecContext(ctx, `insert into media_file(id, library_id) values (1, 999)`)
		Expect(err).ToNot(HaveOccurred())

		Expect(doctor(ctx, database, out)).To(BeFalse())
		Expect(out.String()).To(ContainSubstring("Foreign key check reported"))
		Expect(out.String()).To(ContainSubstring("media_file"))
		Expect(out.String()).To(ContainSubstring("navidrome scan -f"))
		// GC never touches player, share or playqueue, so don't promise a full cleanup.
		Expect(out.String()).To(ContainSubstring("removed by hand"))
	})

	// Every issue names an FTS-like index, so IsFTSCorruptionOnly alone would send the
	// user to 'search rebuild', but the pragma stopped at its limit without saying so.
	It("does not blame the search index when the issue list is truncated", func() {
		for _, stmt := range []string{
			`create table t(a, b)`,
			`with recursive s(x) as (select 1 union all select x+1 from s where x < 300)
			 insert into t select x, x + 10000 from s`,
			`create index media_file_fts_probe on t(a)`,
			`pragma writable_schema=on`,
			`update sqlite_master set sql = 'CREATE INDEX media_file_fts_probe ON t(b)'
			 where name = 'media_file_fts_probe'`,
		} {
			_, err := database.ExecContext(ctx, stmt)
			Expect(err).ToNot(HaveOccurred())
		}
		Expect(database.Close()).To(Succeed())
		reopen()

		Expect(doctor(ctx, database, out)).To(BeFalse())
		Expect(out.String()).ToNot(ContainSubstring("search rebuild"))
		Expect(out.String()).To(ContainSubstring("backup restore"))
	})
})
