package db_test

import (
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

var _ = Describe("IsSchemaEmpty", func() {
	var database *sql.DB
	BeforeEach(func() {
		path := "file::memory:"
		database, _ = sql.Open(db.Driver, path)
	})

	It("returns false if the goose metadata table is found", func() {
		_, err := database.Exec("create table goose_db_version (id primary key);")
		Expect(err).ToNot(HaveOccurred())
		Expect(db.IsSchemaEmpty(database)).To(BeFalse())
	})

	It("returns true if the schema is brand new", func() {
		Expect(db.IsSchemaEmpty(database)).To(BeTrue())
	})
})
