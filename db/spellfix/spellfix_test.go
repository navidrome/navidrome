//go:build sqlite_spellfix

package spellfix_test

import (
	"database/sql"
	"testing"

	_ "github.com/mattn/go-sqlite3"
	_ "github.com/navidrome/navidrome/db/spellfix"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestSpellfix(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Spellfix Suite")
}

var _ = Describe("spellfix1", func() {
	var db *sql.DB

	BeforeEach(func() {
		var err error
		db, err = sql.Open("sqlite3", ":memory:")
		Expect(err).ToNot(HaveOccurred())
	})

	AfterEach(func() {
		_ = db.Close()
	})

	It("creates a spellfix1 virtual table", func() {
		_, err := db.Exec("CREATE VIRTUAL TABLE demo USING spellfix1")
		Expect(err).ToNot(HaveOccurred())
	})

	It("returns fuzzy matches", func() {
		_, err := db.Exec("CREATE VIRTUAL TABLE demo USING spellfix1")
		Expect(err).ToNot(HaveOccurred())

		_, err = db.Exec("INSERT INTO demo(word) VALUES ('hello'), ('world'), ('help')")
		Expect(err).ToNot(HaveOccurred())

		rows, err := db.Query("SELECT word FROM demo WHERE word MATCH 'helo' AND top=3")
		Expect(err).ToNot(HaveOccurred())
		defer rows.Close()

		var words []string
		for rows.Next() {
			var word string
			Expect(rows.Scan(&word)).To(Succeed())
			words = append(words, word)
		}
		Expect(words).To(ContainElement("hello"))
	})
})
