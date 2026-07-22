package playback

import (
	"context"

	"github.com/navidrome/navidrome/core/playback/mpv"
	"github.com/navidrome/navidrome/model"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

// fakeTrack is a minimal Track implementation used to exercise playbackDevice
// logic without spawning a real mpv process.
type fakeTrack struct {
	playing      bool
	position     int
	positionHits int
	playingHits  int
}

func (f *fakeTrack) IsPlaying() bool {
	f.playingHits++
	return f.playing
}

func (f *fakeTrack) SetVolume(float32) {}
func (f *fakeTrack) Pause()            {}
func (f *fakeTrack) Unpause()          {}

func (f *fakeTrack) Position() int {
	f.positionHits++
	return f.position
}

func (f *fakeTrack) SetPosition(int) error { return nil }
func (f *fakeTrack) Close()                {}
func (f *fakeTrack) String() string        { return "fakeTrack" }

var _ = Describe("playbackDevice", func() {
	var pd *playbackDevice

	BeforeEach(func() {
		pd = NewPlaybackDevice(context.Background(), nil, "auto", "auto")
	})

	Describe("getStatus", func() {
		It("reflects the active track's live state", func() {
			track := &fakeTrack{playing: true, position: 42}
			pd.ActiveTrack = track
			pd.PlaybackQueue.Add(model.MediaFiles{{ID: "1"}})
			pd.Gain = 0.75

			status := pd.getStatus()

			Expect(status.Playing).To(BeTrue())
			Expect(status.Position).To(Equal(42))
			Expect(status.Gain).To(Equal(float32(0.75)))
			Expect(status.CurrentIndex).To(Equal(0))
			Expect(track.positionHits).To(Equal(1))
			Expect(track.playingHits).To(Equal(1))
		})

		It("reports not-playing with no active track", func() {
			status := pd.getStatus()
			Expect(status.Playing).To(BeFalse())
			Expect(status.Position).To(Equal(0))
		})
	})

	Describe("trackSwitcherGoroutine", func() {
		var ctx context.Context
		var cancel context.CancelFunc

		BeforeEach(func() {
			ctx, cancel = context.WithCancel(context.Background())
			pd.serviceCtx = ctx
			pd.PlaybackQueue.Add(model.MediaFiles{{ID: "only-track"}})
			go pd.trackSwitcherGoroutine()
		})

		AfterEach(func() {
			cancel()
		})

		It("ignores a stale finish signal for a track that was already replaced", func() {
			staleTrack := &mpv.MpvTrack{}
			currentTrack := &mpv.MpvTrack{}

			pd.mutex.Lock()
			pd.ActiveTrack = currentTrack
			pd.mutex.Unlock()

			pd.PlaybackDone <- staleTrack

			Consistently(func() bool {
				pd.mutex.RLock()
				defer pd.mutex.RUnlock()
				return pd.ActiveTrack == currentTrack
			}).Should(BeTrue())
			Expect(staleTrack.CloseCalled).To(BeFalse())
			Expect(currentTrack.CloseCalled).To(BeFalse())
		})

		It("closes and advances past a track that legitimately finished", func() {
			finishedTrack := &mpv.MpvTrack{}

			pd.mutex.Lock()
			pd.ActiveTrack = finishedTrack
			pd.mutex.Unlock()

			pd.PlaybackDone <- finishedTrack

			Eventually(func() bool {
				return finishedTrack.CloseCalled
			}).Should(BeTrue())

			Eventually(func() Track {
				pd.mutex.RLock()
				defer pd.mutex.RUnlock()
				return pd.ActiveTrack
			}).Should(BeNil())
		})
	})
})
