package cmd

import (
	"context"

	"github.com/navidrome/navidrome/conf"
	"github.com/navidrome/navidrome/conf/configtest"
	"github.com/navidrome/navidrome/consts"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

type schedulerRecorder struct {
	schedule string
}

func (*schedulerRecorder) Run(context.Context) {}

func (s *schedulerRecorder) Add(schedule string, _ func()) (int, error) {
	s.schedule = schedule
	return 1, nil
}

func (*schedulerRecorder) Remove(int) {}

var _ = Describe("scheduleDBAnalyzer", func() {
	BeforeEach(func() {
		DeferCleanup(configtest.SetupConfig())
	})

	It("does not register the background check when disabled", func() {
		conf.Server.EnableScheduledDBAnalyze = false
		scheduler := &schedulerRecorder{}

		Expect(scheduleDBAnalyzerWith(context.Background(), scheduler)()).To(Succeed())
		Expect(scheduler.schedule).To(BeEmpty())
	})

	It("registers the background check when enabled", func() {
		conf.Server.EnableScheduledDBAnalyze = true
		scheduler := &schedulerRecorder{}

		Expect(scheduleDBAnalyzerWith(context.Background(), scheduler)()).To(Succeed())
		Expect(scheduler.schedule).To(Equal(consts.DBAnalyzeCheckSchedule))
	})
})
