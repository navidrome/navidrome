package dialect_test

import (
	"context"
	"database/sql"
	"testing"

	_ "github.com/mattn/go-sqlite3"
	"github.com/navidrome/navidrome/conf"
	"github.com/navidrome/navidrome/conf/configtest"
	"github.com/navidrome/navidrome/db/dialect"
	"github.com/navidrome/navidrome/log"
	"github.com/navidrome/navidrome/tests"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestDialect(t *testing.T) {
	tests.Init(t, false)
	log.SetLevel(log.LevelFatal)
	RegisterFailHandler(Fail)
	RunSpecs(t, "Dialect Suite")
}

var _ = Describe("SQLite Dialect", func() {
	var d *dialect.SQLiteDialect

	BeforeEach(func() {
		DeferCleanup(configtest.SetupConfig())
		d = dialect.NewSQLite()
	})

	Describe("Identity", func() {
		It("returns correct name", func() {
			Expect(d.Name()).To(Equal("sqlite3"))
		})

		It("returns correct driver", func() {
			Expect(d.Driver()).To(Equal("sqlite3_custom"))
		})

		It("returns correct goose dialect", func() {
			Expect(d.GooseDialect()).To(Equal("sqlite3"))
		})
	})

	Describe("DSN", func() {
		It("returns DbPath from config", func() {
			conf.Server.DbPath = "/path/to/test.db"
			Expect(d.DSN()).To(Equal("/path/to/test.db"))
		})

		It("handles :memory: database", func() {
			conf.Server.DbPath = ":memory:"
			Expect(d.DSN()).To(Equal("file::memory:?cache=shared&_foreign_keys=on"))
		})
	})

	Describe("SQL Generation", func() {
		It("returns ? for placeholder", func() {
			Expect(d.Placeholder(1)).To(Equal("?"))
			Expect(d.Placeholder(5)).To(Equal("?"))
		})

		It("returns random() for RandomFunc", func() {
			Expect(d.RandomFunc()).To(Equal("random()"))
		})

		It("returns SEEDEDRAND for SeededRandomFunc", func() {
			result := d.SeededRandomFunc("seed_key", "table.id")
			Expect(result).To(Equal("SEEDEDRAND('seed_key', table.id)"))
		})

		It("returns LIKE for case insensitive comparison", func() {
			result := d.CaseInsensitiveComparison("name", "value")
			Expect(result).To(Equal("name LIKE value"))
		})
	})

	Describe("Schema Detection", func() {
		var db *sql.DB
		var ctx context.Context

		BeforeEach(func() {
			ctx = context.Background()
			// Use standard sqlite3 driver for testing (not the custom one)
			var err error
			db, err = sql.Open("sqlite3", ":memory:")
			Expect(err).ToNot(HaveOccurred())
		})

		AfterEach(func() {
			db.Close()
		})

		It("returns true for empty schema", func() {
			Expect(d.IsSchemaEmpty(ctx, db)).To(BeTrue())
		})

		It("returns false when goose_db_version exists", func() {
			_, err := db.Exec("CREATE TABLE goose_db_version (id INTEGER PRIMARY KEY)")
			Expect(err).ToNot(HaveOccurred())
			Expect(d.IsSchemaEmpty(ctx, db)).To(BeFalse())
		})
	})
})

var _ = Describe("PostgreSQL Dialect", func() {
	var d *dialect.PostgresDialect

	BeforeEach(func() {
		DeferCleanup(configtest.SetupConfig())
		d = dialect.NewPostgres()
	})

	Describe("Identity", func() {
		It("returns correct name", func() {
			Expect(d.Name()).To(Equal("postgres"))
		})

		It("returns correct driver", func() {
			Expect(d.Driver()).To(Equal("pgx"))
		})

		It("returns correct goose dialect", func() {
			Expect(d.GooseDialect()).To(Equal("postgres"))
		})
	})

	Describe("DSN", func() {
		It("returns DbConnectionString from config", func() {
			conf.Server.DbConnectionString = "postgres://user:pass@localhost:5432/testdb"
			Expect(d.DSN()).To(Equal("postgres://user:pass@localhost:5432/testdb"))
		})
	})

	Describe("SQL Generation", func() {
		It("returns $n for placeholder", func() {
			Expect(d.Placeholder(1)).To(Equal("$1"))
			Expect(d.Placeholder(5)).To(Equal("$5"))
		})

		It("returns random() for RandomFunc", func() {
			Expect(d.RandomFunc()).To(Equal("random()"))
		})

		It("returns seededrand function call for SeededRandomFunc", func() {
			result := d.SeededRandomFunc("seed_key", "table.id")
			Expect(result).To(Equal("seededrand('seed_key', table.id)"))
		})

		It("returns LOWER() for case insensitive comparison", func() {
			result := d.CaseInsensitiveComparison("name", "value")
			Expect(result).To(Equal("LOWER(name) = LOWER(value)"))
		})
	})
})

var _ = Describe("Dialect Current", func() {
	It("defaults to SQLite", func() {
		Expect(dialect.Current).ToNot(BeNil())
		Expect(dialect.Current.Name()).To(Equal("sqlite3"))
	})
})
