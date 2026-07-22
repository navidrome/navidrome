package artwork

import (
	"context"
	"errors"
	"testing"
	"testing/synctest"
	"time"

	"github.com/navidrome/navidrome/conf/configtest"
	"github.com/navidrome/navidrome/tests"
	. "github.com/onsi/gomega"
	"go.uber.org/goleak"
)

func TestArtworkBackoffSchedule(t *testing.T) {
	g := NewWithT(t)
	for _, c := range []struct {
		attempts int
		want     time.Duration
	}{
		{0, 5 * time.Minute},
		{1, 20 * time.Minute},
		{2, 80 * time.Minute},
		{3, 320 * time.Minute},
		{4, 1280 * time.Minute},
		{5, 48 * time.Hour},
		{6, 48 * time.Hour},
	} {
		g.Expect(backoffFor(c.attempts, 0)).To(Equal(c.want), "attempt %d", c.attempts)
	}

	base := backoffFor(2, 0)
	g.Expect(backoffFor(2, 0.2)).To(Equal(time.Duration(float64(base) * 1.2)))
	g.Expect(backoffFor(2, -0.2)).To(Equal(time.Duration(float64(base) * 0.8)))

	lo := time.Duration(float64(320*time.Minute) * 0.8)
	hi := time.Duration(float64(320*time.Minute) * 1.2)
	for range 200 {
		d := backoff(3)
		g.Expect(d).To(BeNumerically(">=", lo))
		g.Expect(d).To(BeNumerically("<=", hi))
	}
}

func TestArtworkBreakerHalfOpen(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		g := NewWithT(t)
		b := newBreaker()

		for range breakerThreshold {
			b.record(errors.New("boom"))
		}
		g.Expect(b.allow()).To(BeFalse(), "breaker opens after consecutive errors")

		time.Sleep(breakerProbeAfter - time.Nanosecond)
		g.Expect(b.allow()).To(BeFalse(), "still open before the probe interval")

		time.Sleep(time.Nanosecond)
		g.Expect(b.allow()).To(BeTrue(), "half-open: one probe is granted")
		g.Expect(b.allow()).To(BeFalse(), "only a single probe per interval")

		b.record(errors.New("boom")) // probe fails -> stay open
		time.Sleep(breakerProbeAfter)
		g.Expect(b.allow()).To(BeTrue(), "another probe after the next interval")

		b.record(nil) // probe succeeds -> close
		g.Expect(b.allow()).To(BeTrue(), "closed breaker admits freely")
		g.Expect(b.allow()).To(BeTrue())
	})
}

func TestArtworkWorkerRunNoLeak(t *testing.T) {
	ignore := goleak.IgnoreCurrent()
	defer goleak.VerifyNone(t, ignore)

	t.Cleanup(configtest.SetupConfig())
	ds := &tests.MockDataStore{MockedArtworkQueue: tests.CreateMockArtworkQueueRepo()}
	w := NewWorker(ds, NewImageStore(t.TempDir()), &fakeExternalProvider{}, tests.NewMockFFmpeg(""))

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() { done <- w.Run(ctx) }()

	time.Sleep(20 * time.Millisecond) // let the loop settle on the idle select
	cancel()
	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("Run returned error on cancel: %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("Run did not exit after context cancel")
	}
}
