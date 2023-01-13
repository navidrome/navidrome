package db

import (
	"database/sql"
	"testing"

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

var _ = Describe("isSchemaEmpty", func() {
	var db *sql.DB
	BeforeEach(func() {
		path := "file::memory:"
		db, _ = sql.Open(Driver, path)
	})

	It("returns false if the goose metadata table is found", func() {
		_, err := db.Exec("create table goose_db_version (id primary key);")
		Expect(err).ToNot(HaveOccurred())
		Expect(isSchemaEmpty(db)).To(BeFalse())
	})

	It("returns true if the schema is brand new", func() {
		Expect(isSchemaEmpty(db)).To(BeTrue())
	})
})
