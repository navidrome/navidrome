package cast

import (
	"context"
	"errors"
	"sync"
	"time"

	"github.com/go-chi/jwtauth/v5"
	"github.com/navidrome/navidrome/conf"
	"github.com/navidrome/navidrome/core/auth"
	"github.com/navidrome/navidrome/model"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

type fakeSession struct {
	mu             sync.Mutex
	snapshot       sessionSnapshot
	events         chan sessionEvent
	playCalls      int
	pauseCalls     int
	closeCalls     int
	setVolumeCalls []float32
	seekCalls      []fakeSeekCall
	playErr        error
	pauseErr       error
	setVolumeErr   error
	seekErr        error
	closeErr       error
}

type fakeSeekCall struct {
	Seconds     float32
	ResumeState *string
}

func newFakeSession(snapshot sessionSnapshot) *fakeSession {
	return &fakeSession{snapshot: snapshot, events: make(chan sessionEvent, 16)}
}

func (f *fakeSession) Snapshot() sessionSnapshot {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.snapshot
}

func (f *fakeSession) Events() <-chan sessionEvent { return f.events }

func (f *fakeSession) Play(context.Context) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.playCalls++
	return f.playErr
}

func (f *fakeSession) Pause(context.Context) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.pauseCalls++
	return f.pauseErr
}

func (f *fakeSession) SeekTo(_ context.Context, seconds float32, resumeState *string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.seekCalls = append(f.seekCalls, fakeSeekCall{Seconds: seconds, ResumeState: resumeState})
	return f.seekErr
}

func (f *fakeSession) SetVolume(_ context.Context, level float32) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.setVolumeCalls = append(f.setVolumeCalls, level)
	return f.setVolumeErr
}

func (f *fakeSession) Close() error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.closeCalls++
	return f.closeErr
}

func asyncPlaybackReceiver(ch <-chan bool) <-chan bool {
	received := make(chan bool, 4)
	go func() {
		for v := range ch {
			received <- v
		}
	}()
	return received
}

var _ = Describe("CastTrack", func() {
	var (
		ctx          context.Context
		cancel       context.CancelFunc
		playbackDone chan bool
		target       Target
		mf           model.MediaFile
	)

	BeforeEach(func() {
		ctx, cancel = context.WithCancel(context.Background())
		playbackDone = make(chan bool)
		target = Target{Host: "127.0.0.1", Port: 8009, Name: "Kitchen"}
		mf = model.MediaFile{ID: "track-123", Title: "Song", Suffix: "mp3", Duration: 120}
		oldBaseHost := conf.Server.BaseHost
		oldBaseScheme := conf.Server.BaseScheme
		oldPublicTokenAuth := auth.PublicTokenAuth
		conf.Server.BaseHost = "example.com"
		conf.Server.BaseScheme = "https"
		auth.PublicTokenAuth = jwtauth.New("HS256", []byte("test-secret"), nil)
		DeferCleanup(func() {
			conf.Server.BaseHost = oldBaseHost
			conf.Server.BaseScheme = oldBaseScheme
			auth.PublicTokenAuth = oldPublicTokenAuth
		})
	})

	AfterEach(func() { cancel() })

	It("generates a playback URL during construction and exposes a safe string", func() {
		var gotLoad loadSpec
		fs := newFakeSession(sessionSnapshot{Connected: true, PlayerState: "PAUSED"})
		track, err := newTrack(ctx, playbackDone, target, mf, func(_ context.Context, _ Target, load loadSpec) (session, error) {
			gotLoad = load
			return fs, nil
		})
		Expect(err).ToNot(HaveOccurred())
		DeferCleanup(track.Close)
		Expect(gotLoad.ContentID).To(ContainSubstring("/share/playback/"))
		Expect(gotLoad.ContentType).To(Equal("audio/mpeg"))
		Expect(track.String()).To(ContainSubstring("track-123"))
		Expect(track.String()).ToNot(ContainSubstring("/share/playback/"))
		Expect(track.String()).ToNot(ContainSubstring("token"))
	})

	It("reports playing only for PLAYING and BUFFERING", func() {
		playing := &CastTrack{session: newFakeSession(sessionSnapshot{Connected: true, PlayerState: "PLAYING"})}
		buffering := &CastTrack{session: newFakeSession(sessionSnapshot{Connected: true, PlayerState: "BUFFERING"})}
		paused := &CastTrack{session: newFakeSession(sessionSnapshot{Connected: true, PlayerState: "PAUSED"})}
		Expect(playing.IsPlaying()).To(BeTrue())
		Expect(buffering.IsPlaying()).To(BeTrue())
		Expect(paused.IsPlaying()).To(BeFalse())
	})

	It("uses receiver volume immediately after construction", func() {
		fs := newFakeSession(sessionSnapshot{Connected: true, PlayerState: "PAUSED"})
		track, err := newTrack(ctx, playbackDone, target, mf, func(_ context.Context, _ Target, _ loadSpec) (session, error) {
			return fs, nil
		})
		Expect(err).ToNot(HaveOccurred())
		DeferCleanup(track.Close)
		track.SetVolume(0.25)
		Expect(fs.setVolumeCalls).To(Equal([]float32{0.25}))
	})

	It("sends PLAY and PAUSE through the session", func() {
		fs := newFakeSession(sessionSnapshot{Connected: true, PlayerState: "PAUSED"})
		track := &CastTrack{session: fs, trackCtx: ctx, cancel: cancel}
		track.Unpause()
		track.Pause()
		Expect(fs.playCalls).To(Equal(1))
		Expect(fs.pauseCalls).To(Equal(1))
	})

	It("uses PLAYBACK_START only when seeking from PLAYING", func() {
		playingSession := newFakeSession(sessionSnapshot{Connected: true, PlayerState: "PLAYING"})
		track := &CastTrack{session: playingSession, trackCtx: ctx, cancel: cancel}
		Expect(track.SetPosition(42)).To(Succeed())
		Expect(playingSession.seekCalls).To(HaveLen(1))
		Expect(*playingSession.seekCalls[0].ResumeState).To(Equal("PLAYBACK_START"))

		pausedSession := newFakeSession(sessionSnapshot{Connected: true, PlayerState: "PAUSED"})
		track = &CastTrack{session: pausedSession, trackCtx: ctx, cancel: cancel}
		Expect(track.SetPosition(7)).To(Succeed())
		Expect(pausedSession.seekCalls).To(HaveLen(1))
		Expect(pausedSession.seekCalls[0].ResumeState).To(BeNil())

		neverPlayedSession := newFakeSession(sessionSnapshot{Connected: true, PlayerState: "IDLE", NeverPlayed: true})
		track = &CastTrack{session: neverPlayedSession, trackCtx: ctx, cancel: cancel}
		Expect(track.SetPosition(3)).To(Succeed())
		Expect(neverPlayedSession.seekCalls[0].ResumeState).To(BeNil())
	})

	It("extrapolates position only while the session is live and playing", func() {
		playing := &CastTrack{session: newFakeSession(sessionSnapshot{Connected: true, PlayerState: "PLAYING", CurrentTime: 10, Duration: 12, LastUpdate: time.Now().Add(-3 * time.Second)})}
		buffering := &CastTrack{session: newFakeSession(sessionSnapshot{Connected: true, PlayerState: "BUFFERING", CurrentTime: 10, LastUpdate: time.Now().Add(-3 * time.Second)})}
		disconnected := &CastTrack{session: newFakeSession(sessionSnapshot{Connected: false, Closed: true, PlayerState: "PLAYING", CurrentTime: 10, LastUpdate: time.Now().Add(-3 * time.Second)})}
		Expect(playing.Position()).To(Equal(12))
		Expect(buffering.Position()).To(Equal(10))
		Expect(disconnected.Position()).To(Equal(10))
	})

	It("emits PlaybackDone exactly once on FINISHED", func() {
		received := asyncPlaybackReceiver(playbackDone)
		fs := newFakeSession(sessionSnapshot{Connected: true, PlayerState: "PAUSED"})
		track, err := newTrack(ctx, playbackDone, target, mf, func(_ context.Context, _ Target, _ loadSpec) (session, error) {
			return fs, nil
		})
		Expect(err).ToNot(HaveOccurred())
		DeferCleanup(track.Close)

		fs.events <- sessionEvent{Type: sessionEventMediaStatus, Snapshot: sessionSnapshot{PlayerState: "IDLE", IdleReason: "FINISHED", MediaSessionID: 77}}
		fs.events <- sessionEvent{Type: sessionEventMediaStatus, Snapshot: sessionSnapshot{PlayerState: "IDLE", IdleReason: "FINISHED", MediaSessionID: 77}}

		Eventually(received).Should(Receive(BeTrue()))
		Consistently(received, 100*time.Millisecond).ShouldNot(Receive())
	})

	It("does not emit PlaybackDone for non-natural endings, close, or cancellation", func() {
		fs := newFakeSession(sessionSnapshot{Connected: true, PlayerState: "PAUSED"})
		track, err := newTrack(ctx, playbackDone, target, mf, func(_ context.Context, _ Target, _ loadSpec) (session, error) {
			return fs, nil
		})
		Expect(err).ToNot(HaveOccurred())

		fs.events <- sessionEvent{Type: sessionEventMediaStatus, Snapshot: sessionSnapshot{PlayerState: "IDLE", IdleReason: "CANCELLED", MediaSessionID: 77}}
		fs.events <- sessionEvent{Type: sessionEventMediaStatus, Snapshot: sessionSnapshot{PlayerState: "IDLE", IdleReason: "INTERRUPTED", MediaSessionID: 77}}
		track.Close()
		fs.events <- sessionEvent{Type: sessionEventMediaStatus, Snapshot: sessionSnapshot{PlayerState: "IDLE", IdleReason: "FINISHED", MediaSessionID: 77}}
		Consistently(playbackDone, 150*time.Millisecond).ShouldNot(Receive())
	})

	It("lets close win over FINISHED even when a PlaybackDone receiver is ready", func() {
		received := asyncPlaybackReceiver(playbackDone)
		fs := newFakeSession(sessionSnapshot{Connected: true, PlayerState: "PAUSED"})
		track, err := newTrack(ctx, playbackDone, target, mf, func(_ context.Context, _ Target, _ loadSpec) (session, error) {
			return fs, nil
		})
		Expect(err).ToNot(HaveOccurred())
		track.Close()
		fs.events <- sessionEvent{Type: sessionEventMediaStatus, Snapshot: sessionSnapshot{PlayerState: "IDLE", IdleReason: "FINISHED", MediaSessionID: 77}}
		Consistently(received, 150*time.Millisecond).ShouldNot(Receive())
		Expect(fs.closeCalls).To(Equal(1))
	})

	It("cancels the notifier instead of blocking forever", func() {
		fs := newFakeSession(sessionSnapshot{Connected: true, PlayerState: "PAUSED"})
		track, err := newTrack(ctx, make(chan bool), target, mf, func(_ context.Context, _ Target, _ loadSpec) (session, error) {
			return fs, nil
		})
		Expect(err).ToNot(HaveOccurred())
		fs.events <- sessionEvent{Type: sessionEventMediaStatus, Snapshot: sessionSnapshot{PlayerState: "IDLE", IdleReason: "FINISHED", MediaSessionID: 77}}
		track.cancel()
		Eventually(track.trackCtx.Done()).Should(BeClosed())
		Consistently(func() bool {
			track.mu.Lock()
			defer track.mu.Unlock()
			return track.doneDelivered
		}, 100*time.Millisecond).Should(BeFalse())
	})

	It("terminates track goroutines on remote session shutdown without PlaybackDone", func() {
		received := asyncPlaybackReceiver(playbackDone)
		fs := newFakeSession(sessionSnapshot{Connected: true, PlayerState: "PAUSED"})
		track, err := newTrack(ctx, playbackDone, target, mf, func(_ context.Context, _ Target, _ loadSpec) (session, error) {
			return fs, nil
		})
		Expect(err).ToNot(HaveOccurred())
		close(fs.events)
		Eventually(track.trackCtx.Done()).Should(BeClosed())
		Consistently(received, 150*time.Millisecond).ShouldNot(Receive())
	})

	It("surfaces construction failures", func() {
		_, err := newTrack(ctx, playbackDone, target, mf, func(_ context.Context, _ Target, _ loadSpec) (session, error) {
			return nil, errors.New("boom")
		})
		Expect(err).To(MatchError("boom"))
	})
})
