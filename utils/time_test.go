package utils_test

import (
	"time"

	"github.com/navidrome/navidrome/utils"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("TimeNewest", func() {
	It("returns zero time when no times are provided", func() {
		Expect(utils.TimeNewest()).To(Equal(time.Time{}))
	})

	It("returns the time when only one time is provided", func() {
		t1 := time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)
		Expect(utils.TimeNewest(t1)).To(Equal(t1))
	})

	It("returns the newest time when multiple times are provided", func() {
		t1 := time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)
		t2 := time.Date(2021, 1, 1, 0, 0, 0, 0, time.UTC)
		t3 := time.Date(2019, 1, 1, 0, 0, 0, 0, time.UTC)

		Expect(utils.TimeNewest(t1, t2, t3)).To(Equal(t2))
	})
})

var _ = Describe("ParseDuration", func() {
	DescribeTable("parses valid durations",
		func(input string, expected time.Duration) {
			d, err := utils.ParseDuration(input)
			Expect(err).ToNot(HaveOccurred())
			Expect(d).To(Equal(expected))
		},
		Entry("standard Go units", "90m", 90*time.Minute),
		Entry("hours", "12h", 12*time.Hour),
		Entry("days", "1d", 24*time.Hour),
		Entry("weeks", "1w", 7*24*time.Hour),
		Entry("multiple days", "3d", 72*time.Hour),
		Entry("mixed day and hours", "1d12h", 36*time.Hour),
		Entry("mixed week, day and hours", "1w2d3h", (7*24+2*24+3)*time.Hour),
		Entry("fractional days", "0.5d", 12*time.Hour),
	)

	DescribeTable("rejects invalid durations",
		func(input string) {
			_, err := utils.ParseDuration(input)
			Expect(err).To(HaveOccurred())
		},
		Entry("empty string", ""),
		Entry("not a duration", "tomorrow"),
		Entry("bare number", "42"),
		Entry("unit only", "d"),
		Entry("unknown unit", "5y"),
	)

	DescribeTable("rejects negative durations",
		func(input string) {
			_, err := utils.ParseDuration(input)
			Expect(err).To(MatchError(ContainSubstring("negative duration")))
		},
		Entry("negative days", "-1d"),
		Entry("negative weeks", "-0.5w"),
		Entry("negative Go units", "-30m"),
	)
})

var _ = Describe("FormatDuration", func() {
	DescribeTable("formats durations using the largest whole units",
		func(input time.Duration, expected string) {
			Expect(utils.FormatDuration(input)).To(Equal(expected))
		},
		Entry("whole weeks", 7*24*time.Hour, "1w"),
		Entry("whole days", 24*time.Hour, "1d"),
		Entry("multiple days", 72*time.Hour, "3d"),
		Entry("day and hours", 36*time.Hour, "1d12h"),
		Entry("week, day and hours", (7*24+2*24+3)*time.Hour, "1w2d3h"),
		Entry("hours only", 12*time.Hour, "12h"),
		Entry("sub-hour", 90*time.Minute, "1h30m0s"),
		Entry("zero", time.Duration(0), "0s"),
	)

	DescribeTable("round-trips through ParseDuration",
		func(input string) {
			d, err := utils.ParseDuration(input)
			Expect(err).ToNot(HaveOccurred())
			formatted := utils.FormatDuration(d)
			d2, err := utils.ParseDuration(formatted)
			Expect(err).ToNot(HaveOccurred())
			Expect(d2).To(Equal(d))
		},
		Entry("1d", "1d"),
		Entry("1w", "1w"),
		Entry("1d12h", "1d12h"),
		Entry("90m", "90m"),
	)
})
