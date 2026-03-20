package scheduler

import (
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/robfig/cron/v3"
)

var _ = Describe("ParseCrontab", func() {
	Describe("standard expressions", func() {
		It("parses a 5-field expression", func() {
			sched, err := ParseCrontab("5 * * * *")
			Expect(err).ToNot(HaveOccurred())
			Expect(sched).To(BeAssignableToTypeOf(&cron.SpecSchedule{}))
		})

		It("parses a 6-field expression with seconds", func() {
			sched, err := ParseCrontab("30 5 * * * *")
			Expect(err).ToNot(HaveOccurred())
			Expect(sched).To(BeAssignableToTypeOf(&cron.SpecSchedule{}))
		})

		It("converts duration string to @every", func() {
			sched, err := ParseCrontab("5m")
			Expect(err).ToNot(HaveOccurred())
			Expect(sched).To(BeAssignableToTypeOf(cron.ConstantDelaySchedule{}))
		})

		It("returns error for empty string", func() {
			_, err := ParseCrontab("")
			Expect(err).To(HaveOccurred())
		})
	})

	Describe("random ~ syntax", func() {
		It("resolves A~B to a value within range", func() {
			sched, err := ParseCrontab("0~30 * * * *")
			Expect(err).ToNot(HaveOccurred())
			spec := sched.(*cron.SpecSchedule)
			minute := findSetBit(spec.Minute)
			Expect(minute).To(BeNumerically(">=", 0))
			Expect(minute).To(BeNumerically("<=", 30))
		})

		It("resolves ~ alone to full field range", func() {
			sched, err := ParseCrontab("~ * * * *")
			Expect(err).ToNot(HaveOccurred())
			spec := sched.(*cron.SpecSchedule)
			minute := findSetBit(spec.Minute)
			Expect(minute).To(BeNumerically(">=", 0))
			Expect(minute).To(BeNumerically("<=", 59))
		})

		It("resolves ~B as min~B", func() {
			sched, err := ParseCrontab("~15 * * * *")
			Expect(err).ToNot(HaveOccurred())
			spec := sched.(*cron.SpecSchedule)
			minute := findSetBit(spec.Minute)
			Expect(minute).To(BeNumerically(">=", 0))
			Expect(minute).To(BeNumerically("<=", 15))
		})

		It("resolves A~ as A~max", func() {
			sched, err := ParseCrontab("15~ * * * *")
			Expect(err).ToNot(HaveOccurred())
			spec := sched.(*cron.SpecSchedule)
			minute := findSetBit(spec.Minute)
			Expect(minute).To(BeNumerically(">=", 15))
			Expect(minute).To(BeNumerically("<=", 59))
		})

		It("resolves multiple random fields independently", func() {
			sched, err := ParseCrontab("0~30 0~12 * * *")
			Expect(err).ToNot(HaveOccurred())
			spec := sched.(*cron.SpecSchedule)
			Expect(findSetBit(spec.Minute)).To(BeNumerically("<=", 30))
			Expect(findSetBit(spec.Hour)).To(BeNumerically("<=", 12))
		})

		It("resolves ~ in DOM field with correct bounds", func() {
			sched, err := ParseCrontab("0 0 ~ * *")
			Expect(err).ToNot(HaveOccurred())
			spec := sched.(*cron.SpecSchedule)
			dom := findSetBit(spec.Dom)
			Expect(dom).To(BeNumerically(">=", 1))
			Expect(dom).To(BeNumerically("<=", 31))
		})

		It("resolves ~ in month field with correct bounds", func() {
			sched, err := ParseCrontab("0 0 1 ~ *")
			Expect(err).ToNot(HaveOccurred())
			spec := sched.(*cron.SpecSchedule)
			month := findSetBit(spec.Month)
			Expect(month).To(BeNumerically(">=", 1))
			Expect(month).To(BeNumerically("<=", 12))
		})

		It("resolves ~ in DOW field with correct bounds", func() {
			sched, err := ParseCrontab("0 0 * * ~")
			Expect(err).ToNot(HaveOccurred())
			spec := sched.(*cron.SpecSchedule)
			dow := findSetBit(spec.Dow)
			Expect(dow).To(BeNumerically(">=", 0))
			Expect(dow).To(BeNumerically("<=", 6))
		})

		It("preserves TZ= prefix through resolution", func() {
			sched, err := ParseCrontab("TZ=America/New_York 0~30 * * * *")
			Expect(err).ToNot(HaveOccurred())
			spec := sched.(*cron.SpecSchedule)
			nyc, _ := time.LoadLocation("America/New_York")
			Expect(spec.Location).To(Equal(nyc))
		})

		It("preserves non-random fields", func() {
			sched, err := ParseCrontab("0~30 10 * * *")
			Expect(err).ToNot(HaveOccurred())
			spec := sched.(*cron.SpecSchedule)
			Expect(spec.Hour & (1 << 10)).ToNot(BeZero())
		})

		It("resolves to a stable value across repeated Next calls", func() {
			sched, err := ParseCrontab("0~30 * * * *")
			Expect(err).ToNot(HaveOccurred())

			ref := time.Date(2025, 1, 1, 12, 0, 0, 0, time.UTC)
			first := sched.Next(ref)
			for range 50 {
				Expect(sched.Next(ref)).To(Equal(first))
			}
		})
	})

	Describe("error cases", func() {
		It("rejects min > max", func() {
			_, err := ParseCrontab("30~0 * * * *")
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("beyond end"))
		})

		It("rejects value above field maximum", func() {
			_, err := ParseCrontab("0~60 * * * *")
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("above maximum"))
		})

		It("rejects value below field minimum", func() {
			_, err := ParseCrontab("0 0 0~15 * *")
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("below minimum"))
		})

		It("rejects ~ mixed with comma (list)", func() {
			_, err := ParseCrontab("0~30,45 * * * *")
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("cannot be combined"))
		})

		It("rejects ~ mixed with slash (step)", func() {
			_, err := ParseCrontab("0~30/5 * * * *")
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("cannot be combined"))
		})

		It("rejects @ descriptor with ~", func() {
			_, err := ParseCrontab("@every 0~30m")
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("descriptor"))
		})

		It("rejects wrong number of fields", func() {
			_, err := ParseCrontab("0~30 * *")
			Expect(err).To(HaveOccurred())
		})

		It("rejects non-numeric range values", func() {
			_, err := ParseCrontab("a~b * * * *")
			Expect(err).To(HaveOccurred())
		})
	})
})

// findSetBit returns the lowest bit position set in v, ignoring the starBit (bit 63).
func findSetBit(v uint64) int {
	v &^= 1 << 63 // clear starBit
	for i := 0; i < 63; i++ {
		if v&(1<<uint(i)) != 0 {
			return i
		}
	}
	return -1
}
