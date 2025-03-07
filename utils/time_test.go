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
