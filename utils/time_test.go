package utils_test

import (
	"time"

	"github.com/navidrome/navidrome/utils"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("TimeNewest", func() {
	var (
		newestTime, middleTime, oldestTime time.Time
	)

	BeforeEach(func() {
		newestTime = time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)
		middleTime = time.Date(2021, 1, 1, 0, 0, 0, 0, time.UTC)
		oldestTime = time.Date(2019, 1, 1, 0, 0, 0, 0, time.UTC)
	})

	It("returns zero time when no times are provided", func() {
		Expect(utils.TimeNewest()).To(Equal(time.Time{}))
	})

	It("returns the same time when only one time is provided", func() {
		Expect(utils.TimeNewest(newestTime)).To(Equal(newestTime))
	})

	It("returns the newest time when multiple times are provided", func() {
		Expect(utils.TimeNewest(newestTime, middleTime, oldestTime)).To(Equal(middleTime))
	})

	It("returns the newest time even if the newest is first", func() {
		Expect(utils.TimeNewest(middleTime, newestTime, oldestTime)).To(Equal(middleTime))
	})

	It("returns the newest time even if there are duplicates", func() {
		Expect(utils.TimeNewest(newestTime, middleTime, middleTime, oldestTime)).To(Equal(middleTime))
	})
})
