package db_test

import (
	"context"
	"database/sql"
	"math/rand"
	"os"
	"time"

	"github.com/navidrome/navidrome/conf"
	"github.com/navidrome/navidrome/conf/configtest"
	. "github.com/navidrome/navidrome/db"
	"github.com/navidrome/navidrome/tests"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func shortTime(year int, month time.Month, day, hour, minute int) time.Time {
	return time.Date(year, month, day, hour, minute, 0, 0, time.UTC)
}

var _ = Describe("database backups", func() {
	When("there are a few backup files", func() {
		var ctx context.Context
		var timesShuffled []time.Time

		timesDecreasingChronologically := []time.Time{
			shortTime(2024, 11, 6, 5, 11),
			shortTime(2024, 11, 6, 5, 8),
			shortTime(2024, 11, 6, 4, 32),
			shortTime(2024, 11, 6, 2, 4),
			shortTime(2024, 11, 6, 1, 52),

			shortTime(2024, 11, 5, 23, 0),
			shortTime(2024, 11, 5, 6, 4),
			shortTime(2024, 11, 4, 2, 4),
			shortTime(2024, 11, 3, 8, 5),
			shortTime(2024, 11, 2, 5, 24),
			shortTime(2024, 11, 1, 5, 24),

			shortTime(2024, 10, 31, 5, 9),
			shortTime(2024, 10, 30, 5, 9),
			shortTime(2024, 10, 23, 14, 3),
			shortTime(2024, 10, 22, 3, 6),
			shortTime(2024, 10, 11, 14, 3),

			shortTime(2024, 9, 21, 19, 5),
			shortTime(2024, 9, 3, 8, 5),

			shortTime(2024, 7, 5, 1, 1),

			shortTime(2023, 8, 2, 19, 5),

			shortTime(2021, 8, 2, 19, 5),
			shortTime(2020, 8, 2, 19, 5),
		}

		BeforeEach(func() {
			DeferCleanup(configtest.SetupConfig())

			tempFolder, err := os.MkdirTemp("", "navidrome_backup")
			Expect(err).ToNot(HaveOccurred())
			conf.Server.Backup.Path = tempFolder

			DeferCleanup(func() {
				_ = os.RemoveAll(tempFolder)
			})

			timesShuffled = make([]time.Time, len(timesDecreasingChronologically))
			copy(timesShuffled, timesDecreasingChronologically)
			rand.Shuffle(len(timesShuffled), func(i, j int) {
				timesShuffled[i], timesShuffled[j] = timesShuffled[j], timesShuffled[i]
			})

			for _, time := range timesShuffled {
				path := BackupPath(time)
				file, err := os.Create(path)
				Expect(err).ToNot(HaveOccurred())
				_ = file.Close()
			}

			ctx = context.Background()
		})

		DescribeTable("prune", func(count, expected int) {
			conf.Server.Backup.Count = count
			pruneCount, err := Prune(ctx)
			Expect(err).ToNot(HaveOccurred())
			for idx, time := range timesDecreasingChronologically {
				_, err := os.Stat(BackupPath(time))
				shouldExist := idx < conf.Server.Backup.Count
				if shouldExist {
					Expect(err).ToNot(HaveOccurred())
				} else {
					Expect(err).To(MatchError(os.ErrNotExist))
				}
			}

			Expect(len(timesDecreasingChronologically) - pruneCount).To(Equal(expected))
		},
			Entry("preserve latest 5 backups", 5, 5),
			Entry("delete all files", 0, 0),
			Entry("preserve all files when at length", len(timesDecreasingChronologically), len(timesDecreasingChronologically)),
			Entry("preserve all files when less than count", 10000, len(timesDecreasingChronologically)))
	})

	Describe("backup and restore", Ordered, func() {
		var ctx context.Context

		BeforeAll(func() {
			ctx = context.Background()
			DeferCleanup(configtest.SetupConfig())

			conf.Server.DbPath = "file::memory:?cache=shared&_foreign_keys=on"
			DeferCleanup(Init(ctx))
		})

		BeforeEach(func() {
			tempFolder, err := os.MkdirTemp("", "navidrome_backup")
			Expect(err).ToNot(HaveOccurred())
			conf.Server.Backup.Path = tempFolder

			DeferCleanup(func() {
				_ = os.RemoveAll(tempFolder)
			})
		})

		It("successfully backups the database", func() {
			path, err := Backup(ctx)
			Expect(err).ToNot(HaveOccurred())

			backup, err := sql.Open(Driver, path)
			Expect(err).ToNot(HaveOccurred())
			Expect(IsSchemaEmpty(ctx, backup)).To(BeFalse())
		})

		It("successfully restores the database", func() {
			path, err := Backup(ctx)
			Expect(err).ToNot(HaveOccurred())

			err = tests.ClearDB()
			Expect(err).ToNot(HaveOccurred())
			Expect(IsSchemaEmpty(ctx, Db())).To(BeTrue())

			err = Restore(ctx, path)
			Expect(err).ToNot(HaveOccurred())
			Expect(IsSchemaEmpty(ctx, Db())).To(BeFalse())
		})
	})
})
