package db_test

import (
	"context"
	"database/sql"
	"time"

	"github.com/navidrome/navidrome/consts"
	"github.com/navidrome/navidrome/db"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Optimize", func() {
	var (
		ctx      context.Context
		database *sql.DB
		now      time.Time
	)

	BeforeEach(func() {
		ctx = context.Background()
		now = time.Date(2026, time.July, 9, 12, 0, 0, 0, time.UTC)
		var err error
		database, err = sql.Open(db.Dialect, "file::memory:")
		Expect(err).ToNot(HaveOccurred())
		DeferCleanup(database.Close)

		_, err = database.Exec(`create table property(
			id varchar(255) primary key,
			value varchar(255) not null default ''
		)`)
		Expect(err).ToNot(HaveOccurred())
		_, err = database.Exec("create table analyze_probe(id integer primary key, flag int)")
		Expect(err).ToNot(HaveOccurred())
		_, err = database.Exec(`insert into analyze_probe(flag)
			with recursive s(x) as (select 1 union all select x+1 from s where x < 3000)
			select 0 from s`)
		Expect(err).ToNot(HaveOccurred())
		_, err = database.Exec("create index probe_flag on analyze_probe(flag)")
		Expect(err).ToNot(HaveOccurred())
		_, err = database.Exec("analyze")
		Expect(err).ToNot(HaveOccurred())
	})

	putProperty := func(key, value string) {
		_, err := database.Exec(`insert into property(id, value) values(?, ?)
			on conflict(id) do update set value=excluded.value`, key, value)
		Expect(err).ToNot(HaveOccurred())
	}

	getProperty := func(key string) string {
		var value string
		Expect(database.QueryRow("select value from property where id=?", key).Scan(&value)).To(Succeed())
		return value
	}

	poisonStats := func() {
		_, err := database.Exec("update sqlite_stat1 set stat='3000 50' where idx='probe_flag'")
		Expect(err).ToNot(HaveOccurred())
	}

	It("replaces poisoned planner statistics with full-quality ones", func() {
		poisonStats()
		putProperty(consts.DBAnalyzePendingKey, "1")

		Expect(db.OptimizeDBAt(ctx, database, now)).To(Succeed())

		var stat string
		err := database.QueryRow("select stat from sqlite_stat1 where idx='probe_flag'").Scan(&stat)
		Expect(err).ToNot(HaveOccurred())
		// A full ANALYZE sees all 3000 rows share one value: avg rows per key = row count.
		Expect(stat).To(Equal("3000 3000"))
		Expect(getProperty(consts.LastDBAnalyzeAtKey)).To(Equal(now.Format(time.RFC3339Nano)))
		Expect(getProperty(consts.DBAnalyzePendingKey)).To(Equal("0"))
	})

	It("runs when no previous analysis was recorded", func() {
		ran, err := db.OptimizeDBIfNeeded(ctx, database, now)
		Expect(err).ToNot(HaveOccurred())
		Expect(ran).To(BeTrue())
		Expect(getProperty(consts.LastDBAnalyzeAtKey)).To(Equal(now.Format(time.RFC3339Nano)))
	})

	It("skips a recent analysis when no refresh is pending", func() {
		lastAnalyze := now.Add(-23 * time.Hour)
		putProperty(consts.LastDBAnalyzeAtKey, lastAnalyze.Format(time.RFC3339Nano))
		putProperty(consts.DBAnalyzePendingKey, "0")
		poisonStats()

		ran, err := db.OptimizeDBIfNeeded(ctx, database, now)
		Expect(err).ToNot(HaveOccurred())
		Expect(ran).To(BeFalse())
		Expect(getProperty(consts.LastDBAnalyzeAtKey)).To(Equal(lastAnalyze.Format(time.RFC3339Nano)))

		var stat string
		Expect(database.QueryRow("select stat from sqlite_stat1 where idx='probe_flag'").Scan(&stat)).To(Succeed())
		Expect(stat).To(Equal("3000 50"))
	})

	It("runs when the previous analysis is stale", func() {
		putProperty(consts.LastDBAnalyzeAtKey, now.Add(-consts.DBAnalyzeMaxAge).Format(time.RFC3339Nano))
		putProperty(consts.DBAnalyzePendingKey, "0")

		ran, err := db.OptimizeDBIfNeeded(ctx, database, now)
		Expect(err).ToNot(HaveOccurred())
		Expect(ran).To(BeTrue())
		Expect(getProperty(consts.LastDBAnalyzeAtKey)).To(Equal(now.Format(time.RFC3339Nano)))
	})

	It("runs when a refresh is pending even if the previous analysis is recent", func() {
		putProperty(consts.LastDBAnalyzeAtKey, now.Format(time.RFC3339Nano))
		putProperty(consts.DBAnalyzePendingKey, "1")

		ran, err := db.OptimizeDBIfNeeded(ctx, database, now.Add(time.Hour))
		Expect(err).ToNot(HaveOccurred())
		Expect(ran).To(BeTrue())
		Expect(getProperty(consts.DBAnalyzePendingKey)).To(Equal("0"))
	})

	DescribeTable("backs off after consecutive analysis failures",
		func(failures string, retryDelay time.Duration) {
			putProperty(consts.DBAnalyzePendingKey, "1")
			putProperty(consts.DBAnalyzeFailureCountKey, failures)
			putProperty(consts.LastDBAnalyzeAttemptAtKey, now.Format(time.RFC3339Nano))

			ran, err := db.OptimizeDBIfNeeded(ctx, database, now.Add(retryDelay-time.Nanosecond))
			Expect(err).ToNot(HaveOccurred())
			Expect(ran).To(BeFalse())

			ran, err = db.OptimizeDBIfNeeded(ctx, database, now.Add(retryDelay))
			Expect(err).ToNot(HaveOccurred())
			Expect(ran).To(BeTrue())
			Expect(getProperty(consts.DBAnalyzeFailureCountKey)).To(Equal("0"))
			Expect(getProperty(consts.DBAnalyzePendingKey)).To(Equal("0"))
		},
		Entry("for 30 minutes after the first failure", "1", 30*time.Minute),
		Entry("for one hour after the second failure", "2", time.Hour),
		Entry("for two hours after the third failure", "3", 2*time.Hour),
		Entry("for 24 hours after the fourth failure", "4", 24*time.Hour),
	)

	It("records consecutive analysis failures", func() {
		putProperty(consts.DBAnalyzeFailureCountKey, "2")

		Expect(db.RecordAnalyzeFailure(ctx, database, now)).To(Succeed())

		Expect(getProperty(consts.DBAnalyzeFailureCountKey)).To(Equal("3"))
		Expect(getProperty(consts.LastDBAnalyzeAttemptAtKey)).To(Equal(now.Format(time.RFC3339Nano)))
		Expect(getProperty(consts.DBAnalyzePendingKey)).To(Equal("1"))
	})

	It("does not record success when analysis fails", func() {
		lastAnalyze := now.Add(-48 * time.Hour).Format(time.RFC3339Nano)
		putProperty(consts.LastDBAnalyzeAtKey, lastAnalyze)
		canceledCtx, cancel := context.WithCancel(ctx)
		cancel()

		Expect(db.OptimizeDBAt(canceledCtx, database, now)).To(MatchError(ContainSubstring("context canceled")))
		Expect(getProperty(consts.LastDBAnalyzeAtKey)).To(Equal(lastAnalyze))
	})
})
