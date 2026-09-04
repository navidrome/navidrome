package pglite_test

import (
	"context"
	"database/sql"
	"os"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/navidrome/navidrome/db/pglite"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/tetratelabs/wazero/experimental"
	"github.com/tetratelabs/wazero/experimental/logging"
)

var _ = Describe("PGlite under wazero", func() {
	var pg *pglite.PGlite
	var db *sql.DB

	BeforeEach(func() {
		tarball := os.Getenv("ND_PGLITE_TARBALL")
		if tarball == "" {
			tarball = "tmp/pglite-wasi-O2-fix.tar.gz"
		}
		if _, err := os.Stat(tarball); err != nil {
			Skip("pglite tarball not found: " + tarball)
		}
		ctx := context.Background()
		if os.Getenv("ND_PGLITE_TRACE") != "" {
			ctx = experimental.WithFunctionListenerFactory(ctx,
				logging.NewHostLoggingListenerFactory(os.Stderr, logging.LogScopeFilesystem))
		}
		var err error
		pg, err = pglite.New(ctx, pglite.Config{
			DataDir: GinkgoT().TempDir(),
			Tarball: tarball,
			Stderr:  GinkgoWriter,
		})
		Expect(err).ToNot(HaveOccurred())
		DeferCleanup(pg.Close)

		db, err = sql.Open("pgx", pg.DSN())
		Expect(err).ToNot(HaveOccurred())
		db.SetMaxOpenConns(1)
		DeferCleanup(db.Close)
	})

	It("answers SELECT version()", func() {
		var version string
		Expect(db.QueryRow("SELECT version()").Scan(&version)).To(Succeed())
		GinkgoWriter.Println("version:", version)
		Expect(version).To(ContainSubstring("PostgreSQL 17"))
	})

	It("keeps the connection after a SQL error and survives a reconnect", func() {
		_, err := db.Exec("SELECT * FROM nope WHERE id = $1", 1)
		Expect(err).To(MatchError(ContainSubstring("does not exist")))
		var one int
		Expect(db.QueryRow("SELECT $1::int", 1).Scan(&one)).To(Succeed())
		GinkgoWriter.Println("connections after error:", pg.Connections())
		Expect(pg.Connections()).To(BeEquivalentTo(1))

		// The session must still be usable for DDL, DML and transactions after the error.
		_, err = db.Exec("CREATE TABLE t (id int PRIMARY KEY, name text)")
		Expect(err).ToNot(HaveOccurred())
		tx, err := db.Begin()
		Expect(err).ToNot(HaveOccurred())
		_, err = tx.Exec("INSERT INTO t VALUES ($1, $2)", 1, "one")
		Expect(err).ToNot(HaveOccurred())
		Expect(tx.Commit()).To(Succeed())
		_, err = db.Exec("INSERT INTO t VALUES ($1, $2)", 1, "dup")
		Expect(err).To(MatchError(ContainSubstring("duplicate key")))
		var name string
		Expect(db.QueryRow("SELECT name FROM t WHERE id = $1", 1).Scan(&name)).To(Succeed())
		Expect(name).To(Equal("one"))
		Expect(pg.Connections()).To(BeEquivalentTo(1))

		// Force a new connection and make sure parameterized queries still work on it.
		Expect(db.Close()).To(Succeed())
		db, err = sql.Open("pgx", pg.DSN())
		Expect(err).ToNot(HaveOccurred())
		db.SetMaxOpenConns(1)
		Expect(db.QueryRow("SELECT $1::int", 2).Scan(&one)).To(Succeed())
		Expect(one).To(Equal(2))
		Expect(pg.Connections()).To(BeEquivalentTo(2))
	})

	It("creates a table and reads rows back", func() {
		_, err := db.Exec("CREATE TABLE property (id text PRIMARY KEY, value text)")
		Expect(err).ToNot(HaveOccurred())
		_, err = db.Exec("INSERT INTO property (id, value) VALUES ($1, $2)", "JWTSecret", "abc")
		Expect(err).ToNot(HaveOccurred())

		start := time.Now()
		var value string
		Expect(db.QueryRow("SELECT value FROM property WHERE id = $1", "JWTSecret").Scan(&value)).To(Succeed())
		GinkgoWriter.Println("select took", time.Since(start))
		Expect(value).To(Equal("abc"))

		var count int
		Expect(db.QueryRow("SELECT count(*) FROM property").Scan(&count)).To(Succeed())
		Expect(count).To(Equal(1))
	})
})
