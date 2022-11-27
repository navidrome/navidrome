package utils

import (
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Time Conversion", func() {
	It("converts from Date to Millis and back to Date", func() {
		date := time.Date(2002, 8, 9, 12, 11, 13, 1000000, time.Local)
		milli := ToMillis(date)
		Expect(ToTime(milli)).To(Equal(date))
	})
})
