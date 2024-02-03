package log

import (
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = DescribeTable("ShortDur",
	func(d time.Duration, expected string) {
		Expect(ShortDur(d)).To(Equal(expected))
	},
	Entry("1ns", 1*time.Nanosecond, "1ns"),
	Entry("9µs", 9*time.Microsecond, "9µs"),
	Entry("2ms", 2*time.Millisecond, "2ms"),
	Entry("5ms", 5*time.Millisecond, "5ms"),
	Entry("5.2ms", 5*time.Millisecond+240*time.Microsecond, "5.2ms"),
	Entry("1s", 1*time.Second, "1s"),
	Entry("1.26s", 1*time.Second+263*time.Millisecond, "1.26s"),
	Entry("4m", 4*time.Minute, "4m"),
	Entry("4m3s", 4*time.Minute+3*time.Second, "4m3s"),
	Entry("4h", 4*time.Hour, "4h"),
	Entry("4h", 4*time.Hour+2*time.Second, "4h"),
	Entry("4h2m", 4*time.Hour+2*time.Minute+5*time.Second+200*time.Millisecond, "4h2m"),
)
