package playback

import (
	"context"
	"errors"
	"fmt"
	"sync"

	"github.com/navidrome/navidrome/model"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

type fakeTrack struct {
	name           string
	playing        bool
	position       int
	positionErr    error
	pauseCalls     int
	unpauseCalls   int
	closeCalls     int
	setVolumeCalls []float32
}

func (t *fakeTrack) IsPlaying() bool { return t.playing }

func (t *fakeTrack) SetVolume(value float32) {
	t.setVolumeCalls = append(t.setVolumeCalls, value)
}

func (t *fakeTrack) Pause() {
	t.pauseCalls++
	t.playing = false
}

func (t *fakeTrack) Unpause() {
	t.unpauseCalls++
	t.playing = true
}

func (t *fakeTrack) Position() int { return t.position }

func (t *fakeTrack) SetPosition(offset int) error {
	if t.positionErr != nil {
		return t.positionErr
	}
	t.position = offset
	return nil
}

func (t *fakeTrack) Close() { t.closeCalls++ }

func (t *fakeTrack) String() string { return t.name }

type trackFactoryRecorder struct {
	mu      sync.Mutex
	calls   []trackFactoryCall
	tracks  []*fakeTrack
	err     error
	nextErr []error
}

type trackFactoryCall struct {
	ctx          context.Context
	playbackDone chan bool
	deviceName   string
	mf           model.MediaFile
}

func (r *trackFactoryRecorder) factory(ctx context.Context, playbackDone chan bool, deviceName string, mf model.MediaFile) (Track, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.calls = append(r.calls, trackFactoryCall{ctx: ctx, playbackDone: playbackDone, deviceName: deviceName, mf: mf})
	if len(r.nextErr) > 0 {
		err := r.nextErr[0]
		r.nextErr = r.nextErr[1:]
		if err != nil {
			return nil, err
		}
	}
	if r.err != nil {
		return nil, r.err
	}
	track := &fakeTrack{name: fmt.Sprintf("track-%s", mf.ID)}
	r.tracks = append(r.tracks, track)
	return track, nil
}

func (r *trackFactoryRecorder) trackCount() int {
	r.mu.Lock()
	defer r.mu.Unlock()
	return len(r.tracks)
}

func (r *trackFactoryRecorder) trackAt(i int) *fakeTrack {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.tracks[i]
}

type fakePlaybackServer struct {
	mediaFiles map[string]model.MediaFile
}

func (s *fakePlaybackServer) Run(context.Context) error { return nil }

func (s *fakePlaybackServer) GetDeviceForUser(string) (*playbackDevice, error) { return nil, nil }

func (s *fakePlaybackServer) GetMediaFile(id string) (*model.MediaFile, error) {
	mf, ok := s.mediaFiles[id]
	if !ok {
		return nil, model.ErrNotFound
	}
	copy := mf
	return &copy, nil
}

func makeMediaFiles(ids ...string) model.MediaFiles {
	items := make(model.MediaFiles, len(ids))
	for i, id := range ids {
		items[i] = model.MediaFile{ID: id, Path: "/music/" + id + ".mp3"}
	}
	return items
}

var _ = Describe("playbackDevice", func() {
	var (
		ctx    context.Context
		cancel context.CancelFunc
		parent *fakePlaybackServer
		rec    *trackFactoryRecorder
		device *playbackDevice
	)

	BeforeEach(func() {
		ctx, cancel = context.WithCancel(context.Background())
		parent = &fakePlaybackServer{}
		rec = &trackFactoryRecorder{}
		device = NewPlaybackDevice(ctx, parent, "jukebox", "auto", rec.factory)
	})

	AfterEach(func() {
		cancel()
	})

	It("starts with a queue and no active track", func() {
		device.PlaybackQueue.Add(makeMediaFiles("1", "2"))

		status, err := device.Start(ctx)
		Expect(err).ToNot(HaveOccurred())
		Expect(rec.calls).To(HaveLen(1))
		Expect(rec.calls[0].deviceName).To(Equal("auto"))
		Expect(rec.calls[0].playbackDone).To(BeIdenticalTo(device.PlaybackDone))
		Expect(rec.calls[0].mf.ID).To(Equal("1"))
		Expect(device.ActiveTrack).To(BeIdenticalTo(rec.tracks[0]))
		Expect(rec.tracks[0].unpauseCalls).To(Equal(1))
		Expect(rec.tracks[0].setVolumeCalls).To(Equal([]float32{DefaultGain}))
		Expect(status.CurrentIndex).To(Equal(0))
		Expect(status.Playing).To(BeTrue())
	})

	It("starts a paused existing track without creating a new one", func() {
		track := &fakeTrack{name: "existing", playing: false}
		device.ActiveTrack = track

		status, err := device.Start(ctx)
		Expect(err).ToNot(HaveOccurred())
		Expect(rec.calls).To(BeEmpty())
		Expect(track.unpauseCalls).To(Equal(1))
		Expect(status.Playing).To(BeTrue())
	})

	It("keeps an already playing track running", func() {
		track := &fakeTrack{name: "existing", playing: true}
		device.ActiveTrack = track

		status, err := device.Start(ctx)
		Expect(err).ToNot(HaveOccurred())
		Expect(rec.calls).To(BeEmpty())
		Expect(track.unpauseCalls).To(Equal(0))
		Expect(status.Playing).To(BeTrue())
	})

	It("stops by pausing the active track", func() {
		track := &fakeTrack{name: "existing", playing: true}
		device.ActiveTrack = track

		status, err := device.Stop(ctx)
		Expect(err).ToNot(HaveOccurred())
		Expect(track.pauseCalls).To(Equal(1))
		Expect(status.Playing).To(BeFalse())
	})

	It("applies gain to an active track", func() {
		track := &fakeTrack{name: "existing"}
		device.ActiveTrack = track

		status, err := device.SetGain(ctx, 0.4)
		Expect(err).ToNot(HaveOccurred())
		Expect(device.Gain).To(Equal(float32(0.4)))
		Expect(track.setVolumeCalls).To(Equal([]float32{0.4}))
		Expect(status.Gain).To(Equal(float32(0.4)))
	})

	It("applies prior gain when creating a track later", func() {
		device.PlaybackQueue.Add(makeMediaFiles("1"))
		_, err := device.SetGain(ctx, 0.6)
		Expect(err).ToNot(HaveOccurred())

		_, err = device.Start(ctx)
		Expect(err).ToNot(HaveOccurred())
		Expect(rec.tracks).To(HaveLen(1))
		Expect(rec.tracks[0].setVolumeCalls).To(Equal([]float32{0.6}))
	})

	It("seeks within the same track", func() {
		device.PlaybackQueue.Add(makeMediaFiles("1", "2"))
		track := &fakeTrack{name: "existing", playing: true, position: 10}
		device.ActiveTrack = track
		device.PlaybackQueue.Index = 0

		status, err := device.Skip(ctx, 0, 42)
		Expect(err).ToNot(HaveOccurred())
		Expect(rec.calls).To(BeEmpty())
		Expect(track.pauseCalls).To(Equal(1))
		Expect(track.position).To(Equal(42))
		Expect(track.closeCalls).To(Equal(0))
		Expect(track.unpauseCalls).To(Equal(1))
		Expect(status.CurrentIndex).To(Equal(0))
		Expect(status.Position).To(Equal(42))
		Expect(status.Playing).To(BeTrue())
	})

	It("skips to another queue index", func() {
		device.PlaybackQueue.Add(makeMediaFiles("1", "2"))
		current := &fakeTrack{name: "current", playing: true, position: 3}
		device.ActiveTrack = current
		device.PlaybackQueue.Index = 0

		status, err := device.Skip(ctx, 1, 25)
		Expect(err).ToNot(HaveOccurred())
		Expect(current.pauseCalls).To(Equal(1))
		Expect(current.closeCalls).To(Equal(1))
		Expect(rec.calls).To(HaveLen(1))
		Expect(rec.calls[0].mf.ID).To(Equal("2"))
		Expect(rec.tracks[0].position).To(Equal(25))
		Expect(rec.tracks[0].unpauseCalls).To(Equal(1))
		Expect(device.ActiveTrack).To(BeIdenticalTo(rec.tracks[0]))
		Expect(status.CurrentIndex).To(Equal(1))
		Expect(status.Position).To(Equal(25))
		Expect(status.Playing).To(BeTrue())
	})

	It("clears the queue and closes the active track", func() {
		device.PlaybackQueue.Add(makeMediaFiles("1", "2"))
		track := &fakeTrack{name: "existing", playing: true}
		device.ActiveTrack = track

		status, err := device.Clear(ctx)
		Expect(err).ToNot(HaveOccurred())
		Expect(track.pauseCalls).To(Equal(1))
		Expect(track.closeCalls).To(Equal(1))
		Expect(device.ActiveTrack).To(BeNil())
		Expect(device.PlaybackQueue.Index).To(Equal(-1))
		Expect(device.PlaybackQueue.Items).To(BeNil())
		Expect(status.CurrentIndex).To(Equal(-1))
		Expect(status.Playing).To(BeFalse())
	})

	It("removes the current playing item while preserving existing behavior", func() {
		device.PlaybackQueue.Add(makeMediaFiles("1", "2"))
		track := &fakeTrack{name: "existing", playing: true, position: 17}
		device.ActiveTrack = track
		device.PlaybackQueue.Index = 0

		status, err := device.Remove(ctx, 0)
		Expect(err).ToNot(HaveOccurred())
		Expect(track.pauseCalls).To(Equal(1))
		Expect(track.closeCalls).To(Equal(0))
		Expect(device.ActiveTrack).To(BeIdenticalTo(track))
		Expect(device.PlaybackQueue.Size()).To(Equal(1))
		Expect(device.PlaybackQueue.Index).To(Equal(-1))
		Expect(status.CurrentIndex).To(Equal(-1))
		Expect(status.Position).To(Equal(17))
		Expect(status.Playing).To(BeFalse())
	})

	It("advances to the next track on natural completion", func() {
		device.PlaybackQueue.Add(makeMediaFiles("1", "2"))

		_, err := device.Start(ctx)
		Expect(err).ToNot(HaveOccurred())
		first := rec.tracks[0]

		delivered := make(chan struct{})
		go func() {
			device.PlaybackDone <- true
			close(delivered)
		}()

		Eventually(delivered).Should(BeClosed())
		Eventually(func() int { return rec.trackCount() }).Should(Equal(2))
		Eventually(func() int { return first.closeCalls }).Should(Equal(1))
		Eventually(func() int { return device.PlaybackQueue.Index }).Should(Equal(1))
		Eventually(func() Track { return device.ActiveTrack }).Should(BeIdenticalTo(rec.trackAt(1)))
		Eventually(func() int { return rec.trackAt(1).unpauseCalls }).Should(Equal(1))
	})

	It("finishes on the final track after natural completion", func() {
		device.PlaybackQueue.Add(makeMediaFiles("1"))

		_, err := device.Start(ctx)
		Expect(err).ToNot(HaveOccurred())
		first := rec.tracks[0]

		delivered := make(chan struct{})
		go func() {
			device.PlaybackDone <- true
			close(delivered)
		}()

		Eventually(delivered).Should(BeClosed())
		Eventually(func() int { return first.closeCalls }).Should(Equal(1))
		Eventually(func() Track { return device.ActiveTrack }).Should(BeNil())
		Consistently(func() int { return rec.trackCount() }).Should(Equal(1))
		Expect(device.PlaybackQueue.Index).To(Equal(0))
	})

	It("returns a factory error when creating a track fails", func() {
		device.PlaybackQueue.Add(makeMediaFiles("1"))
		rec.err = errors.New("factory failed")

		status, err := device.Start(ctx)
		Expect(err).To(MatchError("factory failed"))
		Expect(device.ActiveTrack).To(BeNil())
		Expect(status.CurrentIndex).To(Equal(0))
		Expect(status.Playing).To(BeFalse())
	})

	It("returns a seek error from SetPosition", func() {
		seekErr := errors.New("seek failed")
		track := &fakeTrack{name: "existing", playing: true, positionErr: seekErr}
		device.PlaybackQueue.Add(makeMediaFiles("1"))
		device.ActiveTrack = track
		device.PlaybackQueue.Index = 0

		status, err := device.Skip(ctx, 0, 99)
		Expect(err).To(MatchError(seekErr))
		Expect(track.pauseCalls).To(Equal(1))
		Expect(track.unpauseCalls).To(Equal(0))
		Expect(status.Playing).To(BeFalse())
		Expect(status.Position).To(Equal(0))
	})
})
