//go:build !windows

package plugins

import (
	"database/sql"

	_ "github.com/mattn/go-sqlite3"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("migrateDB", func() {
	var db *sql.DB

	BeforeEach(func() {
		var err error
		db, err = sql.Open("sqlite3", ":memory:")
		Expect(err).ToNot(HaveOccurred())
	})

	AfterEach(func() {
		if db != nil {
			db.Close()
		}
	})

	getUserVersion := func() int {
		var version int
		Expect(db.QueryRow(`PRAGMA user_version`).Scan(&version)).To(Succeed())
		return version
	}

	It("applies all migrations on a fresh database", func() {
		migrations := []string{
			`CREATE TABLE test (id INTEGER PRIMARY KEY, name TEXT)`,
			`ALTER TABLE test ADD COLUMN email TEXT`,
		}

		Expect(migrateDB(db, migrations)).To(Succeed())
		Expect(getUserVersion()).To(Equal(2))

		// Verify schema
		_, err := db.Exec(`INSERT INTO test (id, name, email) VALUES (1, 'Alice', 'alice@test.com')`)
		Expect(err).ToNot(HaveOccurred())
	})

	It("skips already applied migrations", func() {
		migrations1 := []string{
			`CREATE TABLE test (id INTEGER PRIMARY KEY, name TEXT)`,
		}
		Expect(migrateDB(db, migrations1)).To(Succeed())
		Expect(getUserVersion()).To(Equal(1))

		// Add a new migration
		migrations2 := []string{
			`CREATE TABLE test (id INTEGER PRIMARY KEY, name TEXT)`,
			`ALTER TABLE test ADD COLUMN email TEXT`,
		}
		Expect(migrateDB(db, migrations2)).To(Succeed())
		Expect(getUserVersion()).To(Equal(2))

		// Verify the new column exists
		_, err := db.Exec(`INSERT INTO test (id, name, email) VALUES (1, 'Alice', 'alice@test.com')`)
		Expect(err).ToNot(HaveOccurred())
	})

	It("is a no-op when all migrations are applied", func() {
		migrations := []string{
			`CREATE TABLE test (id INTEGER PRIMARY KEY)`,
		}
		Expect(migrateDB(db, migrations)).To(Succeed())
		Expect(migrateDB(db, migrations)).To(Succeed())
		Expect(getUserVersion()).To(Equal(1))
	})

	It("is a no-op with empty migrations slice", func() {
		Expect(migrateDB(db, nil)).To(Succeed())
		Expect(getUserVersion()).To(Equal(0))
	})

	It("rolls back on failure", func() {
		migrations := []string{
			`CREATE TABLE test (id INTEGER PRIMARY KEY)`,
			`INVALID SQL STATEMENT`,
		}

		err := migrateDB(db, migrations)
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("migration 2 failed"))

		// Version should remain 0 (rolled back)
		Expect(getUserVersion()).To(Equal(0))

		// Table should not exist (rolled back)
		_, err = db.Exec(`INSERT INTO test (id) VALUES (1)`)
		Expect(err).To(HaveOccurred())
	})
})
