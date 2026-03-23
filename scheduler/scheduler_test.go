package scheduler

import (
	"testing"

	"github.com/navidrome/navidrome/log"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/robfig/cron/v3"
)

func TestScheduler(t *testing.T) {
	log.SetLevel(log.LevelFatal)
	RegisterFailHandler(Fail)
	RunSpecs(t, "Scheduler Suite")
}

var _ = Describe("Scheduler", func() {
	var s *scheduler

	BeforeEach(func() {
		c := cron.New(cron.WithLogger(&logger{}))
		s = &scheduler{c: c}
		s.c.Start() // Start the scheduler for tests
	})

	AfterEach(func() {
		s.c.Stop() // Stop the scheduler after tests
	})

	It("adds and executes a job", func() {
		done := make(chan struct{})

		id, err := s.Add("@every 50ms", func() {
			close(done)
		})

		Expect(err).ToNot(HaveOccurred())
		Expect(id).ToNot(BeZero())

		Eventually(done).Should(BeClosed())
	})

	It("adds a job with random ~ syntax", func() {
		id, err := s.Add("0~59 * * * *", func() {})
		Expect(err).ToNot(HaveOccurred())
		Expect(id).ToNot(BeZero())
		s.Remove(id)
	})
})
