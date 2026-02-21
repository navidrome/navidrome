package migrations

import (
	"context"
	"database/sql"
	"sync"
	"testing"

	_ "github.com/mattn/go-sqlite3"
	"github.com/navidrome/navidrome/consts"
	"github.com/navidrome/navidrome/log"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestMigrations(t *testing.T) {
	log.SetLevel(log.LevelFatal)
	RegisterFailHandler(Fail)
	RunSpecs(t, "Migrations Suite")
}

var _ = Describe("Migration Helpers", func() {
	var db *sql.DB
	var tx *sql.Tx
	var ctx context.Context

	BeforeEach(func() {
		ctx = context.Background()

		var err error
		db, err = sql.Open("sqlite3", ":memory:")
		Expect(err).ToNot(HaveOccurred())

		_, err = db.Exec(`CREATE TABLE property (id TEXT PRIMARY KEY, value TEXT)`)
		Expect(err).ToNot(HaveOccurred())

		tx, err = db.Begin()
		Expect(err).ToNot(HaveOccurred())

		once = sync.Once{}
		initialized = false
	})

	AfterEach(func() {
		if tx != nil {
			tx.Rollback()
		}
		if db != nil {
			db.Close()
		}
	})

	Describe("forceFullRescan", func() {
		It("sets the full scan flag in property table", func() {
			err := forceFullRescan(tx)
			Expect(err).ToNot(HaveOccurred())

			var value string
			err = tx.QueryRow("SELECT value FROM property WHERE id = ?",
				consts.FullScanAfterMigrationFlagKey).Scan(&value)
			Expect(err).ToNot(HaveOccurred())
			Expect(value).To(Equal("1"))
		})
	})

	Describe("createExecuteFunc", func() {
		It("executes SQL statement successfully", func() {
			_, err := tx.Exec(`CREATE TABLE test_table (id INTEGER)`)
			Expect(err).ToNot(HaveOccurred())

			execFn := createExecuteFunc(ctx, tx)
			insertFn := execFn("INSERT INTO test_table (id) VALUES (1)")

			err = insertFn()
			Expect(err).ToNot(HaveOccurred())

			var count int
			err = tx.QueryRow("SELECT COUNT(*) FROM test_table").Scan(&count)
			Expect(err).ToNot(HaveOccurred())
			Expect(count).To(Equal(1))
		})

		It("returns error for invalid SQL", func() {
			execFn := createExecuteFunc(ctx, tx)
			invalidFn := execFn("INVALID SQL STATEMENT")

			err := invalidFn()
			Expect(err).To(HaveOccurred())
		})
	})

	Describe("checkCount", func() {
		It("returns count from rows", func() {
			_, err := tx.Exec(`CREATE TABLE count_test (id INTEGER)`)
			Expect(err).ToNot(HaveOccurred())
			_, err = tx.Exec(`INSERT INTO count_test VALUES (1), (2), (3)`)
			Expect(err).ToNot(HaveOccurred())

			rows, err := tx.Query("SELECT COUNT(*) FROM count_test")
			Expect(err).ToNot(HaveOccurred())
			defer rows.Close()

			count := checkCount(rows)
			Expect(count).To(Equal(3))
		})

		It("returns 0 for empty result", func() {
			_, err := tx.Exec(`CREATE TABLE empty_test (id INTEGER)`)
			Expect(err).ToNot(HaveOccurred())

			rows, err := tx.Query("SELECT COUNT(*) FROM empty_test")
			Expect(err).ToNot(HaveOccurred())
			defer rows.Close()

			count := checkCount(rows)
			Expect(count).To(Equal(0))
		})
	})
})
