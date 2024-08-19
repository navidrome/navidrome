package db

import (
	"context"
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
		var times []time.Time

		BeforeEach(func() {
			DeferCleanup(configtest.SetupConfig())

			ctx = context.Background()

			times = []time.Time{
				shortTime(2024, 11, 6, 5, 11),
				shortTime(2024, 11, 6, 5, 8),  // should always be removed
				shortTime(2024, 11, 6, 4, 32), // hourly #2
				shortTime(2024, 11, 6, 2, 4),  // hourly #3
				shortTime(2024, 11, 6, 1, 52), // hourly #4

				shortTime(2024, 11, 5, 23, 0), // daily #2
				shortTime(2024, 11, 5, 6, 4),  // would be removed if hourly < 5 && daily == 0 || hourly < 6
				shortTime(2024, 11, 4, 2, 4),  // daily #3
				shortTime(2024, 11, 3, 8, 5),  // daily #4. Or new week
				shortTime(2024, 11, 2, 5, 24), // daily #5
				shortTime(2024, 11, 1, 5, 24), // daily #6
				shortTime(2024, 10, 31, 5, 9), // daily #7
				shortTime(2024, 10, 30, 5, 9), // daily #8

				shortTime(2024, 10, 23, 14, 3), // weekly #3
				shortTime(2024, 10, 22, 3, 6),  // removed unless sufficient dailies
				shortTime(2024, 10, 11, 14, 3), // weekly #3

				shortTime(2024, 9, 21, 19, 5), // monthly #3
				shortTime(2024, 9, 3, 8, 5),   // to be removed
				shortTime(2024, 7, 5, 1, 1),   // monthly #4
				shortTime(2023, 8, 2, 19, 5),  // yearly #2

				shortTime(2021, 8, 2, 19, 5), // yearly #3
				shortTime(2020, 8, 2, 19, 5), // yearly #4

			}
		})

		doTest := func(lastUsed int, indices ...int) {
			result := pruneBackups(ctx, times)
			expectedRemoved := make([]time.Time, len(indices))
			for i, idx := range indices {
				expectedRemoved[i] = times[idx]
			}

			if lastUsed+1 < len(times) {
				expectedRemoved = append(expectedRemoved, times[lastUsed+1:]...)
			}

			Expect(result).To(Equal(expectedRemoved))
		}

		It("should handle hourly-only across days", func() {
			conf.Server.Backup.Hourly = 5
			doTest(5, 1)
		})

		It("should handle daily-only across week/month", func() {
			conf.Server.Backup.Daily = 8
			doTest(12, 1, 2, 3, 4, 6)
		})

		It("should handle weekly-only", func() {
			conf.Server.Backup.Weekly = 4
			doTest(15, 1, 2, 3, 4, 5, 6, 7, 9, 10, 11, 12, 14)
		})

		It("should handle monthly-only", func() {
			conf.Server.Backup.Monthly = 5
			doTest(19, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 12, 13, 14, 15, 17)
		})

		It("should handle mix of all", func() {
			conf.Server.Backup.Hourly = 3
			conf.Server.Backup.Daily = 4
			conf.Server.Backup.Weekly = 3
			conf.Server.Backup.Monthly = 2
			conf.Server.Backup.Yearly = 2

			doTest(20, 1, 6, 10, 11, 12, 14, 17)
		})
	})
})
