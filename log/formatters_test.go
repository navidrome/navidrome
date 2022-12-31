package log

import (
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("ShortDur", func() {
	It("formats microseconds", func() {
		Expect(ShortDur(9 * time.Microsecond)).To(Equal("9µs"))
		Expect(ShortDur(2 * time.Microsecond)).To(Equal("2µs"))
	})
	It("rounds milliseconds", func() {
		Expect(ShortDur(5*time.Millisecond + 10*time.Microsecond)).To(Equal("5ms"))
		Expect(ShortDur(5*time.Millisecond + 240*time.Microsecond)).To(Equal("5.2ms"))
	})
	It("rounds seconds", func() {
		Expect(ShortDur(time.Second + 263*time.Millisecond)).To(Equal("1.26s"))
	})
	It("removes 0 secs", func() {
		Expect(ShortDur(4 * time.Minute)).To(Equal("4m"))
	})
	It("rounds to seconds", func() {
		Expect(ShortDur(4*time.Minute + 3*time.Second)).To(Equal("4m3s"))
	})
	It("removes 0 minutes", func() {
		Expect(ShortDur(4 * time.Hour)).To(Equal("4h"))
	})
	It("round big durations to the minute", func() {
		Expect(ShortDur(4*time.Hour + 2*time.Minute + 5*time.Second + 200*time.Millisecond)).
			To(Equal("4h2m"))
	})
})
