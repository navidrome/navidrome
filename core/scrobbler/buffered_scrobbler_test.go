package scrobbler

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
	"testing/synctest"
	"time"

	"github.com/navidrome/navidrome/core/agents"
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

		// Once the service recovers, the buffered entry drains when the open
		// backoff window closes (a wake alone must not drain it early).
		flaky.succeed()
		bs.sendWakeSignal()
		synctest.Wait()
		g.Expect(buffer.Length()).To(Equal(int64(1)), "wake during backoff drained early")
		time.Sleep(80 * time.Second)
		synctest.Wait()
		g.Expect(buffer.Length()).To(Equal(int64(0)))
	})
}

func TestBufferedScrobblerBackoffWindow(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		buffer := tests.CreateMockedScrobbleBufferRepo()
		userRepo := tests.CreateMockUserRepo()
		_ = userRepo.Put(&model.User{ID: "user1", UserName: "alice"})
		ds := &tests.MockDataStore{MockedScrobbleBuffer: buffer, MockedUser: userRepo}
		scr := &fakeScrobbler{Authorized: true}
		scr.SetError(errors.Join(errors.New("boom"), ErrRetryLater))
		bs := newBufferedScrobbler(ds, scr, "test")
		defer bs.Stop()

		// First enqueue: one immediate attempt, then a 5s window opens.
		_ = bs.Scrobble(context.Background(), "user1", Scrobble{MediaFile: model.MediaFile{ID: "1"}, TimeStamp: time.Now()})
		synctest.Wait()
		if got := scr.ScrobbleAttempts(); got != 1 {
			t.Fatalf("expected 1 attempt after first enqueue, got %d", got)
		}

		// A wake inside the window must NOT trigger an early attempt.
		time.Sleep(1 * time.Second)
		_ = bs.Scrobble(context.Background(), "user1", Scrobble{MediaFile: model.MediaFile{ID: "2"}, TimeStamp: time.Now()})
		synctest.Wait()
		if got := scr.ScrobbleAttempts(); got != 1 {
			t.Fatalf("wake during backoff drained early: %d attempts", got)
		}

		// When the 5s window closes, the retry happens.
		time.Sleep(4100 * time.Millisecond)
		synctest.Wait()
		if got := scr.ScrobbleAttempts(); got != 2 {
			t.Fatalf("expected retry after window, got %d attempts", got)
		}
	})
}

func TestBufferedScrobblerHonorsServerDelay(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		buffer := tests.CreateMockedScrobbleBufferRepo()
		userRepo := tests.CreateMockUserRepo()
		_ = userRepo.Put(&model.User{ID: "user1", UserName: "alice"})
		ds := &tests.MockDataStore{MockedScrobbleBuffer: buffer, MockedUser: userRepo}
		scr := &fakeScrobbler{Authorized: true}
		scr.SetError(errors.Join(errors.New("429"), &agents.RetryLaterError{RetryIn: 30 * time.Second}))
		bs := newBufferedScrobbler(ds, scr, "test")
		defer bs.Stop()

		_ = bs.Scrobble(context.Background(), "user1", Scrobble{MediaFile: model.MediaFile{ID: "1"}, TimeStamp: time.Now()})
		synctest.Wait()
		if got := scr.ScrobbleAttempts(); got != 1 {
			t.Fatalf("expected 1 attempt, got %d", got)
		}

		// The 5s exponential floor is overridden by the 30s server delay.
		time.Sleep(20 * time.Second)
		synctest.Wait()
		if got := scr.ScrobbleAttempts(); got != 1 {
			t.Fatalf("retried before server delay elapsed: %d attempts", got)
		}
		time.Sleep(10100 * time.Millisecond)
		synctest.Wait()
		if got := scr.ScrobbleAttempts(); got != 2 {
			t.Fatalf("expected retry after server delay, got %d attempts", got)
		}
	})
}

// The drain visits users in an arbitrary order, so the longest delay must win regardless
// of which user was seen last.
func TestBufferedScrobblerTakesTheLongestServerDelayAcrossUsers(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		buffer := tests.CreateMockedScrobbleBufferRepo()
		userRepo := tests.CreateMockUserRepo()
		_ = userRepo.Put(&model.User{ID: "user1", UserName: "alice"})
		_ = userRepo.Put(&model.User{ID: "user2", UserName: "bob"})
		ds := &tests.MockDataStore{MockedScrobbleBuffer: buffer, MockedUser: userRepo}
		scr := &recoveringScrobbler{delays: map[string]time.Duration{
			"user1": 10 * time.Second,
			"user2": 45 * time.Second,
		}}
		// Both are buffered before the drain goroutine exists: it drains once on startup, and
		// seeing only one user there would park it on that user's delay, ignoring the other.
		_ = buffer.Enqueue("test", "user1", "1", time.Now())
		_ = buffer.Enqueue("test", "user2", "2", time.Now())
		bs := newBufferedScrobbler(ds, scr, "test")
		defer bs.Stop()

		synctest.Wait()
		if got := scr.count.Load(); got != 2 {
			t.Fatalf("expected both users drained, got %d attempts", got)
		}

		time.Sleep(30 * time.Second)
		synctest.Wait()
		if got := scr.count.Load(); got != 2 {
			t.Fatalf("retried on the shorter delay: %d attempts", got)
		}
		time.Sleep(15100 * time.Millisecond)
		synctest.Wait()
		if got := scr.count.Load(); got != 4 {
			t.Fatalf("expected a retry after the longest delay, got %d attempts", got)
		}
	})
}

// recoveringScrobbler is a race-safe Scrobbler whose error can be toggled while
// the buffered scrobbler's goroutine is draining, to exercise retry then recovery.
// With delays set, it instead fails every scrobble asking for that user's delay.
type recoveringScrobbler struct {
	err    atomic.Pointer[error]
	count  atomic.Int32
	delays map[string]time.Duration
}

func (f *recoveringScrobbler) fail(err error) { f.err.Store(&err) }
func (f *recoveringScrobbler) succeed()       { f.err.Store(nil) }

func (f *recoveringScrobbler) IsAuthorized(context.Context, string) bool { return true }

func (f *recoveringScrobbler) NowPlaying(context.Context, string, *model.MediaFile, int) error {
	return nil
}

func (f *recoveringScrobbler) Scrobble(_ context.Context, userId string, _ Scrobble) error {
	f.count.Add(1)
	if f.delays != nil {
		return errors.Join(errors.New("429"), &agents.RetryLaterError{RetryIn: f.delays[userId]})
	}
	if e := f.err.Load(); e != nil {
		return *e
	}
	return nil
}

func (f *recoveringScrobbler) PlaybackReport(context.Context, PlaybackSession) error { return nil }
