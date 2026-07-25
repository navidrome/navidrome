package scrobbler

import (
	"context"
	"sync/atomic"
	"testing"
	"testing/synctest"
	"time"

	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/tests"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("BufferedScrobbler", func() {
	var ds model.DataStore
	var scr *fakeScrobbler
	var bs *bufferedScrobbler
	var ctx context.Context
	var buffer *tests.MockedScrobbleBufferRepo

	BeforeEach(func() {
		ctx = context.Background()
		buffer = tests.CreateMockedScrobbleBufferRepo()
		userRepo := tests.CreateMockUserRepo()
		Expect(userRepo.Put(&model.User{ID: "user1", UserName: "alice"})).To(Succeed())
		ds = &tests.MockDataStore{
			MockedScrobbleBuffer: buffer,
			MockedUser:           userRepo,
		}
		scr = &fakeScrobbler{Authorized: true}
		bs = newBufferedScrobbler(ds, scr, "test")
	})

	It("forwards IsAuthorized calls", func() {
		scr.Authorized = true
		Expect(bs.IsAuthorized(ctx, "user1")).To(BeTrue())

		scr.Authorized = false
		Expect(bs.IsAuthorized(ctx, "user1")).To(BeFalse())
	})

	It("forwards NowPlaying calls", func() {
		track := &model.MediaFile{ID: "123", Title: "Test Track"}
		Expect(bs.NowPlaying(ctx, "user1", track, 0)).To(Succeed())
		Expect(scr.GetNowPlayingCalled()).To(BeTrue())
		Expect(scr.GetUserID()).To(Equal("user1"))
		Expect(scr.GetTrack()).To(Equal(track))
	})

	It("enqueues scrobbles to buffer", func() {
		track := model.MediaFile{ID: "123", Title: "Test Track"}
		now := time.Now()
		scrobble := Scrobble{MediaFile: track, TimeStamp: now}
		Expect(buffer.Length()).To(Equal(int64(0)))
		Expect(scr.ScrobbleCalled.Load()).To(BeFalse())

		Expect(bs.Scrobble(ctx, "user1", scrobble)).To(Succeed())

		// Wait for the background goroutine to process the scrobble.
		// We don't check buffer.Length() here because the background goroutine
		// may dequeue the entry before we can observe it.
		Eventually(scr.ScrobbleCalled.Load).Should(BeTrue())

		lastScrobble := scr.LastScrobble.Load()
		Expect(lastScrobble.MediaFile.ID).To(Equal("123"))
		Expect(lastScrobble.TimeStamp).To(BeTemporally("==", now))
	})

	It("restores the user in the context when draining buffered scrobbles", func() {
		track := model.MediaFile{ID: "123", Title: "Test Track", Artist: "Test Artist"}
		scrobble := Scrobble{MediaFile: track, TimeStamp: time.Now()}

		Expect(bs.Scrobble(ctx, "user1", scrobble)).To(Succeed())

		Eventually(scr.ScrobbleCalled.Load).Should(BeTrue())
		Expect(scr.GetUsername()).To(Equal("alice"))
	})

	It("stops the background goroutine when Stop is called", func() {
		// Replace the real run method with one that signals when it exits
		done := make(chan struct{})

		// Start our instrumented run function that will signal when it exits
		go func() {
			defer close(done)
			bs.run(bs.ctx)
		}()

		// Wait a bit to ensure the goroutine is running
		time.Sleep(10 * time.Millisecond)

		// Call the real Stop method
		bs.Stop()

		// Wait for the goroutine to exit or timeout
		select {
		case <-done:
			// Success, goroutine exited
		case <-time.After(100 * time.Millisecond):
			Fail("Goroutine did not exit in time after Stop was called")
		}
	})
})

var _ = Describe("backoffDelay", func() {
	DescribeTable("computes the exponential backoff curve clamped to the ceiling",
		func(failures int, expected time.Duration) {
			Expect(backoffDelay(failures)).To(Equal(expected))
		},
		Entry("first failure", 0, 5*time.Second),
		Entry("second failure", 1, 10*time.Second),
		Entry("third failure", 2, 20*time.Second),
		Entry("fourth failure", 3, 40*time.Second),
		Entry("fifth failure", 4, 80*time.Second),
		Entry("sixth failure", 5, 160*time.Second),
		Entry("reaches the ceiling", 6, 4*time.Minute),
		Entry("stays clamped past the ceiling", 7, 4*time.Minute),
		Entry("stays clamped for large values", 1000, 4*time.Minute),
		Entry("negative is treated as zero", -1, 5*time.Second),
	)
})

// Drives the real run loop and asserts the exact retry schedule + recovery. Plain
// test: testing/synctest's fake clock needs a *testing.T, which Ginkgo doesn't give.
func TestBufferedScrobblerBackoffSchedule(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		g := NewWithT(t)
		buffer := tests.CreateMockedScrobbleBufferRepo()
		userRepo := tests.CreateMockUserRepo()
		g.Expect(userRepo.Put(&model.User{ID: "user1", UserName: "alice"})).To(Succeed())
		ds := &tests.MockDataStore{MockedScrobbleBuffer: buffer, MockedUser: userRepo}

		flaky := &recoveringScrobbler{}
		flaky.fail(ErrRetryLater)
		bs := newBufferedScrobbler(ds, flaky, "flaky")
		defer func() { bs.Stop(); synctest.Wait() }()

		// Let the loop settle on the empty buffer, then enqueue a scrobble.
		synctest.Wait()
		track := model.MediaFile{ID: "123", Title: "Test Track", Artist: "Test Artist"}
		g.Expect(bs.Scrobble(context.Background(), "user1", Scrobble{MediaFile: track, TimeStamp: time.Now()})).To(Succeed())

		// First attempt fires immediately on the enqueue wake and is left buffered.
		synctest.Wait()
		g.Expect(flaky.count.Load()).To(Equal(int32(1)))
		g.Expect(buffer.Length()).To(Equal(int64(1)))

		// Each subsequent retry waits exactly double the previous: 5s, 10s, 20s, 40s.
		for i, gap := range []time.Duration{5 * time.Second, 10 * time.Second, 20 * time.Second, 40 * time.Second} {
			want := int32(i + 2)
			time.Sleep(gap - time.Nanosecond)
			synctest.Wait()
			g.Expect(flaky.count.Load()).To(Equal(want-1), "retry fired before the %s backoff", gap)
			time.Sleep(time.Nanosecond)
			synctest.Wait()
			g.Expect(flaky.count.Load()).To(Equal(want), "retry did not fire after the %s backoff", gap)
		}

		// Once the service recovers, waking the loop drains the buffered entry.
		flaky.succeed()
		bs.sendWakeSignal()
		synctest.Wait()
		g.Expect(buffer.Length()).To(Equal(int64(0)))
	})
}

// recoveringScrobbler is a race-safe Scrobbler whose error can be toggled while
// the buffered scrobbler's goroutine is draining, to exercise retry then recovery.
type recoveringScrobbler struct {
	err   atomic.Pointer[error]
	count atomic.Int32
}

func (f *recoveringScrobbler) fail(err error) { f.err.Store(&err) }
func (f *recoveringScrobbler) succeed()       { f.err.Store(nil) }

func (f *recoveringScrobbler) IsAuthorized(context.Context, string) bool { return true }

func (f *recoveringScrobbler) NowPlaying(context.Context, string, *model.MediaFile, int) error {
	return nil
}

func (f *recoveringScrobbler) Scrobble(_ context.Context, _ string, _ Scrobble) error {
	f.count.Add(1)
	if e := f.err.Load(); e != nil {
		return *e
	}
	return nil
}

func (f *recoveringScrobbler) PlaybackReport(context.Context, PlaybackSession) error { return nil }
