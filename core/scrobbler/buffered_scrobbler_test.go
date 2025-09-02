package scrobbler

import (
	"context"
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
		ds = &tests.MockDataStore{
			MockedScrobbleBuffer: buffer,
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
		Expect(scr.NowPlayingCalled).To(BeTrue())
		Expect(scr.UserID).To(Equal("user1"))
		Expect(scr.Track).To(Equal(track))
	})

	It("enqueues scrobbles to buffer", func() {
		track := model.MediaFile{ID: "123", Title: "Test Track"}
		now := time.Now()
		scrobble := Scrobble{MediaFile: track, TimeStamp: now}
		Expect(buffer.Length()).To(Equal(int64(0)))
		Expect(scr.ScrobbleCalled.Load()).To(BeFalse())

		Expect(bs.Scrobble(ctx, "user1", scrobble)).To(Succeed())
		Expect(buffer.Length()).To(Equal(int64(1)))

		// Wait for the scrobble to be sent
		Eventually(scr.ScrobbleCalled.Load).Should(BeTrue())

		lastScrobble := scr.LastScrobble.Load()
		Expect(lastScrobble.MediaFile.ID).To(Equal("123"))
		Expect(lastScrobble.TimeStamp).To(BeTemporally("==", now))
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
