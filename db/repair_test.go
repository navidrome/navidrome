package db_test

import (
	"context"
	"database/sql"
	"regexp"
	"strings"

	"github.com/navidrome/navidrome/db"
	"github.com/pressly/goose/v3"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

// Version immediately before the add_fts5_search migration, for the pre-FTS schema spec.
const preFTSMigration = 20260220173399

var ftsSearchTables = []string{"media_file_fts", "album_fts", "artist_fts"}

var _ = Describe("IsFTSCorruptionOnly", func() {
	It("is true when every issue mentions an FTS search table", func() {
		Expect(db.IsFTSCorruptionOnly([]string{
			`fts5: corruption found reading blob 42 from table "media_file_fts"`,
			`malformed inverted index for FTS5 table main.album_fts`,
			`fts5: corruption in "artist_fts"`,
		})).To(BeTrue())
	})

	It("is false when any issue is outside the FTS search tables", func() {
		Expect(db.IsFTSCorruptionOnly([]string{
			`fts5: corruption found reading blob 42 from table "media_file_fts"`,
			`*** in database main ***`,
		})).To(BeFalse())
	})

	It("is false when there are no issues", func() {
		Expect(db.IsFTSCorruptionOnly(nil)).To(BeFalse())
	})
})

var _ = Describe("Repair", func() {
	var (
		ctx      context.Context
		database *sql.DB
	)

	newDB := func(upTo int64) *sql.DB {
		d, err := sql.Open(db.Dialect, "file::memory:")
		Expect(err).ToNot(HaveOccurred())
		d.SetMaxOpenConns(1) // non-shared :memory: — a second conn would be an empty DB
		DeferCleanup(func() { _ = d.Close() })

		_, err = d.ExecContext(ctx, "PRAGMA foreign_keys=off")
		Expect(err).ToNot(HaveOccurred())
		goose.SetBaseFS(db.EmbedMigrations)
		goose.SetLogger(goose.NopLogger())
		DeferCleanup(func() { goose.SetBaseFS(nil) })
		Expect(goose.SetDialect(db.Dialect)).To(Succeed())
		if upTo == 0 {
			Expect(goose.UpContext(ctx, d, "migrations")).To(Succeed())
		} else {
			Expect(goose.UpToContext(ctx, d, "migrations", upTo)).To(Succeed())
		}
		return d
	}

	BeforeEach(func() {
		ctx = context.Background()
		database = newDB(0)

		for _, stmt := range []string{
			`insert into artist(id, name, search_normalized) values ('ar-1', 'Ramones', 'ramones')`,
			`insert into album(id, name, search_normalized) values ('al-1', 'Rocket to Russia', 'rocket to russia')`,
			`insert into media_file(id, title, search_normalized) values ('mf-1', 'Teenage Lobotomy', 'teenage lobotomy')`,
			`insert into media_file(id, title, search_normalized) values ('mf-2', 'Rockaway Beach', 'rockaway beach')`,
		} {
			_, err := database.ExecContext(ctx, stmt)
			Expect(err).ToNot(HaveOccurred())
		}
	})

	corruptFTS := func(table string) {
		// 8+ bytes of garbage: a 4-byte blob still parses as a valid empty structure record
		_, err := database.ExecContext(ctx, `update `+table+`_data set block = x'deadbeefdeadbeef' where id > 1`) //nolint:gosec
		Expect(err).ToNot(HaveOccurred())
	}

	searchFTS := func(table, term string) int {
		var count int
		err := database.QueryRowContext(ctx, `select count(*) from `+table+` where `+table+` match ?`, term).Scan(&count)
		Expect(err).ToNot(HaveOccurred())
		return count
	}

	// ftsSchema returns name -> whitespace-normalized DDL for the FTS tables,
	// their shadow tables, and their triggers.
	ftsSchema := func() map[string]string {
		rows, err := database.QueryContext(ctx,
			`select name, sql from sqlite_master where name like '%_fts%' and sql is not null`)
		Expect(err).ToNot(HaveOccurred())
		defer rows.Close()
		ws := regexp.MustCompile(`\s+`)
		schema := map[string]string{}
		for rows.Next() {
			var name, ddl string
			Expect(rows.Scan(&name, &ddl)).To(Succeed())
			schema[name] = ws.ReplaceAllString(ddl, " ")
		}
		Expect(rows.Err()).ToNot(HaveOccurred())
		return schema
	}

	Describe("IntegrityCheck", func() {
		It("returns no issues for a healthy database", func() {
			issues, err := db.IntegrityCheck(ctx, database)
			Expect(err).ToNot(HaveOccurred())
			Expect(issues).To(BeEmpty())
		})

		It("reports corruption in an FTS index", func() {
			corruptFTS("media_file_fts")
			issues, err := db.IntegrityCheck(ctx, database)
			Expect(err).ToNot(HaveOccurred())
			Expect(issues).ToNot(BeEmpty())
			Expect(strings.Join(issues, "\n")).To(ContainSubstring("media_file_fts"))
		})
	})

	Describe("VerifyFTS", func() {
		It("passes on a healthy index", func() {
			Expect(db.VerifyFTS(ctx, database)).To(Succeed())
		})

		It("fails on a corrupted index, naming the table", func() {
			corruptFTS("album_fts")
			Expect(db.VerifyFTS(ctx, database)).To(MatchError(ContainSubstring("album_fts")))
		})
	})

	Describe("RebuildFTS", func() {
		It("repairs a corrupted FTS index", func() {
			corruptFTS("media_file_fts")

			Expect(db.RebuildFTS(ctx, database)).To(Succeed())

			issues, err := db.IntegrityCheck(ctx, database)
			Expect(err).ToNot(HaveOccurred())
			Expect(issues).To(BeEmpty())
			Expect(db.VerifyFTS(ctx, database)).To(Succeed())
			Expect(searchFTS("media_file_fts", "lobotomy")).To(Equal(1))
			Expect(searchFTS("album_fts", "russia")).To(Equal(1))
			Expect(searchFTS("artist_fts", "ramones")).To(Equal(1))
		})

		It("recreates tables and triggers dropped by hand", func() {
			for _, table := range ftsSearchTables {
				for _, suffix := range []string{"_ai", "_ad", "_au"} {
					_, err := database.ExecContext(ctx, "drop trigger "+table+suffix)
					Expect(err).ToNot(HaveOccurred())
				}
				_, err := database.ExecContext(ctx, "drop table "+table)
				Expect(err).ToNot(HaveOccurred())
			}

			Expect(db.RebuildFTS(ctx, database)).To(Succeed())

			Expect(searchFTS("media_file_fts", "rockaway")).To(Equal(1))
		})

		It("produces the same schema as the migration", func() {
			migrated := ftsSchema()
			Expect(migrated).ToNot(BeEmpty())

			Expect(db.RebuildFTS(ctx, database)).To(Succeed())

			Expect(ftsSchema()).To(Equal(migrated))
		})

		It("refuses to run on a schema older than the FTS migration", func() {
			old := newDB(preFTSMigration)

			err := db.RebuildFTS(ctx, old)
			Expect(err).To(MatchError(ContainSubstring("migration")))
		})

		// A corrupted DB often cannot run pending migrations (the server crashes on it),
		// so repair must not demand a fully migrated schema — only the FTS migration.
		It("runs on a post-FTS schema even when newer migrations are pending", func() {
			behind := newDB(20260702152457)

			Expect(db.RebuildFTS(ctx, behind)).To(Succeed())
		})

		It("leaves working triggers behind", func() {
			Expect(db.RebuildFTS(ctx, database)).To(Succeed())

			_, err := database.ExecContext(ctx,
				`insert into artist(id, name, search_normalized) values ('ar-2', 'Blondie', 'blondie')`)
			Expect(err).ToNot(HaveOccurred())
			Expect(searchFTS("artist_fts", "blondie")).To(Equal(1))

			_, err = database.ExecContext(ctx, `delete from artist where id = 'ar-2'`)
			Expect(err).ToNot(HaveOccurred())
			Expect(searchFTS("artist_fts", "blondie")).To(BeZero())
		})
	})
})
