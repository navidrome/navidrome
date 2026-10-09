package db_test

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"testing"

	"github.com/mattn/go-sqlite3"

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

var _ = DescribeTable("IsBusy",
	func(err error, expected bool) {
		Expect(db.IsBusy(err)).To(Equal(expected))
	},
	Entry("SQLITE_BUSY", sqlite3.Error{Code: sqlite3.ErrBusy}, true),
	Entry("SQLITE_BUSY_SNAPSHOT", sqlite3.Error{Code: sqlite3.ErrBusy, ExtendedCode: sqlite3.ErrBusySnapshot}, true),
	Entry("a wrapped SQLITE_BUSY", fmt.Errorf("persisting: %w", sqlite3.Error{Code: sqlite3.ErrBusy}), true),
	Entry("another SQLite error", sqlite3.Error{Code: sqlite3.ErrConstraint}, false),
	Entry("a non-SQLite error", errors.New("database is locked"), false),
	Entry("nil", nil, false),
)

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
