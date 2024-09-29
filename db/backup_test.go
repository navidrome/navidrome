package db

import (
	"context"
	"math/rand"
	"os"
	"time"

	"github.com/navidrome/navidrome/conf"
	"github.com/navidrome/navidrome/conf/configtest"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func shortTime(year int, month time.Month, day, hour, minute int) time.Time {
	return time.Date(year, month, day, hour, minute, 0, 0, time.UTC)
}

var _ = Describe("database backups", func() {
	Context("prune", func() {
		var ctx context.Context
		var timesDecreasingChronologically []time.Time
		var timesShuffled []time.Time

		BeforeEach(func() {
			name, err := os.MkdirTemp("", "navidrome_backup")
			if err != nil {
				panic(err)
			}

			DeferCleanup(func() {
				configtest.SetupConfig()
				os.RemoveAll(name)
			})

			conf.Server.Backup.Path = name
			timesDecreasingChronologically = []time.Time{
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

			timesShuffled = make([]time.Time, len(timesDecreasingChronologically))
			copy(timesShuffled, timesDecreasingChronologically)
			rand.Shuffle(len(timesShuffled), func(i, j int) {
				timesShuffled[i], timesShuffled[j] = timesShuffled[j], timesShuffled[i]
			})

			for _, time := range timesShuffled {
				path := backupPath(time)
				file, err := os.Create(path)
				Expect(err).To(BeNil())
				file.Close()
			}

			ctx = context.Background()
		})

		DescribeTable("", func(count int) {
			conf.Server.Backup.Count = count
			pruneCount, err := prune(ctx)
			Expect(err).To(BeNil())
			for idx, time := range timesDecreasingChronologically {
				_, err := os.Stat(backupPath(time))
				shouldExist := idx < conf.Server.Backup.Count
				if shouldExist {
					Expect(err).ToNot(HaveOccurred())
				} else {
					Expect(err).To(MatchError(os.ErrNotExist))
				}
			}

			if count >= len(timesDecreasingChronologically) {
				Expect(pruneCount).To(BeZero())
			} else {
				Expect(pruneCount).To(Equal(len(timesDecreasingChronologically) - count))
			}
		},
			Entry("handle 5 backups", 5),
			Entry("delete all files", 0),
			Entry("preserve all files when at length", len(timesDecreasingChronologically)),
			Entry("preserve al files when greater than count", 10000))
	})
})
