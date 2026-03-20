package scheduler

import (
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/robfig/cron/v3"
)

const starBit = 1 << 63

var _ = Describe("ParseCrontab", func() {
	Describe("passthrough (no ~ present)", func() {
		It("parses a standard 5-field expression", func() {
			sched, err := ParseCrontab("5 * * * *")
			Expect(err).ToNot(HaveOccurred())
			Expect(sched).To(BeAssignableToTypeOf(&cron.SpecSchedule{}))
		})

		It("parses a 6-field expression with seconds", func() {
			sched, err := ParseCrontab("30 5 * * * *")
			Expect(err).ToNot(HaveOccurred())
			Expect(sched).To(BeAssignableToTypeOf(&cron.SpecSchedule{}))
		})

		It("parses @every descriptor", func() {
			sched, err := ParseCrontab("@every 5m")
			Expect(err).ToNot(HaveOccurred())
			Expect(sched).To(BeAssignableToTypeOf(cron.ConstantDelaySchedule{}))
		})

		It("parses @daily descriptor", func() {
			sched, err := ParseCrontab("@daily")
			Expect(err).ToNot(HaveOccurred())
			Expect(sched).To(BeAssignableToTypeOf(&cron.SpecSchedule{}))
		})

		It("parses duration string as @every", func() {
			sched, err := ParseCrontab("5m")
			Expect(err).ToNot(HaveOccurred())
			Expect(sched).To(BeAssignableToTypeOf(cron.ConstantDelaySchedule{}))
		})

		It("parses expression with TZ= prefix", func() {
			sched, err := ParseCrontab("TZ=America/New_York 5 * * * *")
			Expect(err).ToNot(HaveOccurred())
			Expect(sched).To(BeAssignableToTypeOf(&cron.SpecSchedule{}))
		})

		It("returns error for invalid expression", func() {
			_, err := ParseCrontab("invalid")
			Expect(err).To(HaveOccurred())
		})

		It("returns error for empty string", func() {
			_, err := ParseCrontab("")
			Expect(err).To(HaveOccurred())
		})
	})

	Describe("random ~ syntax", func() {
		It("parses A~B in minute field", func() {
			sched, err := ParseCrontab("0~30 * * * *")
			Expect(err).ToNot(HaveOccurred())
			cs, ok := sched.(*CrontabSchedule)
			Expect(ok).To(BeTrue(), "expected *CrontabSchedule")
			Expect(cs.Minute.IsRandom).To(BeTrue())
			Expect(cs.Minute.Min).To(Equal(uint(0)))
			Expect(cs.Minute.Max).To(Equal(uint(30)))
			Expect(cs.Second.IsRandom).To(BeFalse())
			Expect(cs.Hour.IsRandom).To(BeFalse())
		})

		It("parses ~ alone as full range for minute field", func() {
			sched, err := ParseCrontab("~ * * * *")
			Expect(err).ToNot(HaveOccurred())
			cs := sched.(*CrontabSchedule)
			Expect(cs.Minute.IsRandom).To(BeTrue())
			Expect(cs.Minute.Min).To(Equal(uint(0)))
			Expect(cs.Minute.Max).To(Equal(uint(59)))
		})

		It("parses ~B as min~B", func() {
			sched, err := ParseCrontab("~15 * * * *")
			Expect(err).ToNot(HaveOccurred())
			cs := sched.(*CrontabSchedule)
			Expect(cs.Minute.Min).To(Equal(uint(0)))
			Expect(cs.Minute.Max).To(Equal(uint(15)))
		})

		It("parses A~ as A~max", func() {
			sched, err := ParseCrontab("15~ * * * *")
			Expect(err).ToNot(HaveOccurred())
			cs := sched.(*CrontabSchedule)
			Expect(cs.Minute.Min).To(Equal(uint(15)))
			Expect(cs.Minute.Max).To(Equal(uint(59)))
		})

		It("parses multiple random fields", func() {
			sched, err := ParseCrontab("0~30 0~12 * * *")
			Expect(err).ToNot(HaveOccurred())
			cs := sched.(*CrontabSchedule)
			Expect(cs.Minute.IsRandom).To(BeTrue())
			Expect(cs.Minute.Min).To(Equal(uint(0)))
			Expect(cs.Minute.Max).To(Equal(uint(30)))
			Expect(cs.Hour.IsRandom).To(BeTrue())
			Expect(cs.Hour.Min).To(Equal(uint(0)))
			Expect(cs.Hour.Max).To(Equal(uint(12)))
		})

		It("parses 6-field expression with random and seconds", func() {
			sched, err := ParseCrontab("0 0~30 * * * *")
			Expect(err).ToNot(HaveOccurred())
			cs := sched.(*CrontabSchedule)
			Expect(cs.Second.IsRandom).To(BeFalse())
			Expect(cs.Minute.IsRandom).To(BeTrue())
			Expect(cs.Minute.Min).To(Equal(uint(0)))
			Expect(cs.Minute.Max).To(Equal(uint(30)))
		})

		It("parses ~ in DOM field with correct bounds", func() {
			sched, err := ParseCrontab("0 0 ~ * *")
			Expect(err).ToNot(HaveOccurred())
			cs := sched.(*CrontabSchedule)
			Expect(cs.Dom.IsRandom).To(BeTrue())
			Expect(cs.Dom.Min).To(Equal(uint(1)))
			Expect(cs.Dom.Max).To(Equal(uint(31)))
		})

		It("parses ~ in month field with correct bounds", func() {
			sched, err := ParseCrontab("0 0 1 ~ *")
			Expect(err).ToNot(HaveOccurred())
			cs := sched.(*CrontabSchedule)
			Expect(cs.Month.IsRandom).To(BeTrue())
			Expect(cs.Month.Min).To(Equal(uint(1)))
			Expect(cs.Month.Max).To(Equal(uint(12)))
		})

		It("parses ~ in DOW field with correct bounds", func() {
			sched, err := ParseCrontab("0 0 * * ~")
			Expect(err).ToNot(HaveOccurred())
			cs := sched.(*CrontabSchedule)
			Expect(cs.Dow.IsRandom).To(BeTrue())
			Expect(cs.Dow.Min).To(Equal(uint(0)))
			Expect(cs.Dow.Max).To(Equal(uint(6)))
		})

		It("parses with TZ= prefix", func() {
			sched, err := ParseCrontab("TZ=America/New_York 0~30 * * * *")
			Expect(err).ToNot(HaveOccurred())
			cs := sched.(*CrontabSchedule)
			Expect(cs.Minute.IsRandom).To(BeTrue())
			nyc, _ := time.LoadLocation("America/New_York")
			Expect(cs.Location).To(Equal(nyc))
		})

		It("preserves non-random field bitmasks from base", func() {
			sched, err := ParseCrontab("0~30 10 * * *")
			Expect(err).ToNot(HaveOccurred())
			cs := sched.(*CrontabSchedule)
			Expect(cs.Hour.IsRandom).To(BeFalse())
			Expect(cs.base.Hour & (1 << 10)).ToNot(BeZero())
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

var _ = Describe("CrontabSchedule", func() {
	Describe("Next", func() {
		It("delegates to base SpecSchedule when no fields are random", func() {
			base := cron.SpecSchedule{
				Second:   1 << 0,
				Minute:   1 << 5,
				Hour:     0x00FFFFFF | starBit,
				Dom:      0xFFFFFFFE | starBit,
				Month:    0x1FFE | starBit,
				Dow:      0x7F | starBit,
				Location: time.UTC,
			}
			cs := &CrontabSchedule{
				Second:   randomField{IsRandom: false},
				Minute:   randomField{IsRandom: false},
				Hour:     randomField{IsRandom: false},
				Dom:      randomField{IsRandom: false},
				Month:    randomField{IsRandom: false},
				Dow:      randomField{IsRandom: false},
				base:     base,
				Location: time.UTC,
			}

			ref := time.Date(2025, 1, 1, 12, 0, 0, 0, time.UTC)
			result := cs.Next(ref)
			expected := base.Next(ref)
			Expect(result).To(Equal(expected))
		})

		It("does not set starBit on random DOM field", func() {
			base := cron.SpecSchedule{
				Second:   1 << 0,
				Minute:   1 << 0,
				Hour:     1 << 0,
				Dom:      1 << 1,
				Month:    0x1FFE | starBit,
				Dow:      1 << 1,
				Location: time.UTC,
			}
			cs := &CrontabSchedule{
				Second:   randomField{IsRandom: false},
				Minute:   randomField{IsRandom: false},
				Hour:     randomField{IsRandom: false},
				Dom:      randomField{IsRandom: true, Min: 1, Max: 15},
				Month:    randomField{IsRandom: false},
				Dow:      randomField{IsRandom: false},
				base:     base,
				Location: time.UTC,
			}

			ref := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
			for i := 0; i < 50; i++ {
				result := cs.Next(ref)
				day := result.Day()
				weekday := result.Weekday()
				Expect(day <= 15 || weekday == time.Monday).To(BeTrue(),
					"day=%d weekday=%s should match DOM 1-15 OR Monday", day, weekday)
			}
		})
	})
})

var _ = Describe("CrontabSchedule Next with parsed schedules", func() {
	It("produces minutes within range for 0~30 * * * *", func() {
		sched, err := ParseCrontab("0~30 * * * *")
		Expect(err).ToNot(HaveOccurred())

		ref := time.Date(2025, 1, 1, 12, 0, 0, 0, time.UTC)
		seen := make(map[int]bool)
		for i := 0; i < 100; i++ {
			result := sched.Next(ref)
			Expect(result.Minute()).To(BeNumerically(">=", 0))
			Expect(result.Minute()).To(BeNumerically("<=", 30))
			seen[result.Minute()] = true
		}
		Expect(len(seen)).To(BeNumerically(">", 1), "expected multiple distinct minutes")
	})

	It("handles wrap-around with 55~ * * * *", func() {
		sched, err := ParseCrontab("55~ * * * *")
		Expect(err).ToNot(HaveOccurred())

		ref := time.Date(2025, 1, 1, 12, 58, 0, 0, time.UTC)
		for i := 0; i < 50; i++ {
			result := sched.Next(ref)
			Expect(result.Minute()).To(BeNumerically(">=", 55))
			Expect(result.Minute()).To(BeNumerically("<=", 59))
			Expect(result.After(ref)).To(BeTrue())
		}
	})

	It("returns zero time for impossible schedule (Feb 31)", func() {
		sched, err := ParseCrontab("~ ~ 31~ 2 *")
		Expect(err).ToNot(HaveOccurred())

		ref := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
		result := sched.Next(ref)
		Expect(result.IsZero()).To(BeTrue())
	})

	It("handles ~ in DOM field with correct bounds", func() {
		sched, err := ParseCrontab("0 0 ~ * *")
		Expect(err).ToNot(HaveOccurred())

		ref := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
		seen := make(map[int]bool)
		for i := 0; i < 200; i++ {
			result := sched.Next(ref)
			Expect(result.Day()).To(BeNumerically(">=", 1))
			Expect(result.Day()).To(BeNumerically("<=", 31))
			seen[result.Day()] = true
		}
		Expect(len(seen)).To(BeNumerically(">", 1), "expected multiple distinct days")
	})
})
