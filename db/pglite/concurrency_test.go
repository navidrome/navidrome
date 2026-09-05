package pglite_test

import (
	"context"
	"database/sql"
	"sync"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/navidrome/navidrome/db/pglite"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("PGlite concurrent connections", func() {
	var pg *pglite.PGlite
	var db *sql.DB

	BeforeEach(func() {
		var err error
		pg, err = pglite.New(context.Background(), pglite.Config{
			DataDir: GinkgoT().TempDir(), Stderr: GinkgoWriter,
		})
		Expect(err).ToNot(HaveOccurred())
		DeferCleanup(pg.Close)

		db, err = sql.Open("pgx", pg.DSN())
		Expect(err).ToNot(HaveOccurred())
		db.SetMaxOpenConns(4)
		DeferCleanup(db.Close)

		_, err = db.Exec("CREATE TABLE t (id int PRIMARY KEY, name text)")
		Expect(err).ToNot(HaveOccurred())
		for i := range 200 {
			_, err = db.Exec("INSERT INTO t VALUES ($1, $2)", i, "row")
			Expect(err).ToNot(HaveOccurred())
		}
	})

	It("serves 4 concurrent connections with correct results", func() {
		var wg sync.WaitGroup
		errs := make(chan error, 4)
		for w := range 4 {
			wg.Add(1)
			go func() {
				defer wg.Done()
				for i := range 25 {
					id := (w*25 + i) % 200
					var name string
					if err := db.QueryRow("SELECT name FROM t WHERE id = $1", id).Scan(&name); err != nil {
						errs <- err
						return
					}
					if name != "row" {
						errs <- sql.ErrNoRows
						return
					}
				}
			}()
		}
		wg.Wait()
		close(errs)
		for err := range errs {
			Expect(err).ToNot(HaveOccurred())
		}
		GinkgoWriter.Println("bridge connections opened:", pg.Connections())
		Expect(pg.Connections()).To(BeNumerically(">", 1))
	})

	It("does not let a second connection enter an open transaction", func() {
		tx, err := db.Begin()
		Expect(err).ToNot(HaveOccurred())
		_, err = tx.Exec("INSERT INTO t VALUES (1000, 'in-tx')")
		Expect(err).ToNot(HaveOccurred())

		// A second connection must neither see the uncommitted row nor join the transaction.
		done := make(chan int, 1)
		go func() {
			var n int
			if err := db.QueryRow("SELECT count(*) FROM t WHERE id = 1000").Scan(&n); err == nil {
				done <- n
			}
		}()
		select {
		case n := <-done:
			Fail("second connection ran inside the open transaction, count=" + string(rune('0'+n)))
		case <-time.After(500 * time.Millisecond):
			// blocked, as required
		}

		Expect(tx.Commit()).To(Succeed())
		Eventually(done, 5*time.Second).Should(Receive(Equal(1)))
	})

	It("shares session state between connections (single backend)", func() {
		_, err := db.Exec("SET application_name = 'alpha'")
		Expect(err).ToNot(HaveOccurred())
		var seen string
		Expect(db.QueryRow("SELECT current_setting('application_name')").Scan(&seen)).To(Succeed())
		GinkgoWriter.Println("application_name seen by another connection:", seen)
		Expect(seen).To(Equal("alpha"), "documents the limitation: all clients share one PG session")
	})

	It("makes progress with parallel transactional workers (the scanner's pattern)", func() {
		done := make(chan error, 3)
		for w := range 3 {
			go func() {
				for i := range 20 {
					tx, err := db.Begin()
					if err != nil {
						done <- err
						return
					}
					if _, err := tx.Exec("INSERT INTO t VALUES ($1, $2)", 10000+w*100+i, "worker"); err != nil {
						_ = tx.Rollback()
						done <- err
						return
					}
					if err := tx.Commit(); err != nil {
						done <- err
						return
					}
				}
				done <- nil
			}()
		}
		for range 3 {
			Eventually(done, 60*time.Second).Should(Receive(BeNil()))
		}
		var n int
		Expect(db.QueryRow("SELECT count(*) FROM t WHERE name = 'worker'").Scan(&n)).To(Succeed())
		Expect(n).To(Equal(60))
	})

	It("serves queries over an in-process pipe, with no unix socket", func() {
		piped, err := pg.OpenDB()
		Expect(err).ToNot(HaveOccurred())
		defer piped.Close()
		piped.SetMaxOpenConns(2)

		var n int
		Expect(piped.QueryRow("SELECT count(*) FROM t").Scan(&n)).To(Succeed())
		Expect(n).To(Equal(200))

		tx, err := piped.Begin()
		Expect(err).ToNot(HaveOccurred())
		_, err = tx.Exec("INSERT INTO t VALUES ($1, $2)", 5000, "piped")
		Expect(err).ToNot(HaveOccurred())
		Expect(tx.Commit()).To(Succeed())

		var name string
		Expect(piped.QueryRow("SELECT name FROM t WHERE id = $1", 5000).Scan(&name)).To(Succeed())
		Expect(name).To(Equal("piped"))

		// An error must not kill the piped connection either.
		_, err = piped.Exec("SELECT * FROM nope")
		Expect(err).To(MatchError(ContainSubstring("does not exist")))
		Expect(piped.QueryRow("SELECT count(*) FROM t").Scan(&n)).To(Succeed())
		Expect(n).To(Equal(201))
	})

	It("measures throughput at 1 vs 4 connections", func() {
		bench := func(conns int) time.Duration {
			d, err := sql.Open("pgx", pg.DSN())
			Expect(err).ToNot(HaveOccurred())
			defer d.Close()
			d.SetMaxOpenConns(conns)
			start := time.Now()
			var wg sync.WaitGroup
			for range conns {
				wg.Add(1)
				go func() {
					defer wg.Done()
					for range 100 / conns {
						var n int
						_ = d.QueryRow("SELECT count(*) FROM t WHERE id < 150").Scan(&n)
					}
				}()
			}
			wg.Wait()
			return time.Since(start)
		}
		one, four := bench(1), bench(4)
		GinkgoWriter.Printf("100 queries: 1 conn %s, 4 conns %s\n", one.Round(time.Millisecond), four.Round(time.Millisecond))
	})
})
