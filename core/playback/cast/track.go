package cast

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/navidrome/navidrome/core/publicurl"
	"github.com/navidrome/navidrome/log"
	"github.com/navidrome/navidrome/model"
)

const constructorTimeout = 10 * time.Second

type sessionFactory func(context.Context, Target, loadSpec) (session, error)

type CastTrack struct {
	mu sync.Mutex

	mediaFile             model.MediaFile
	target                Target
	playbackDone          chan bool
	naturalFinish         chan struct{}
	trackCtx              context.Context
	cancel                context.CancelFunc
	session               session
	receiverVolumeHandler func(float32)

	closing       bool
	finishQueued  bool
	doneDelivered bool
}

func NewTrack(ctx context.Context, playbackDone chan bool, target Target, mf model.MediaFile) (*CastTrack, error) {
	return newTrack(ctx, playbackDone, target, mf, func(ctx context.Context, target Target, load loadSpec) (session, error) {
		return newSession(ctx, target, load)
	})
}

func newTrack(ctx context.Context, playbackDone chan bool, target Target, mf model.MediaFile, makeSession sessionFactory) (*CastTrack, error) {
	trackCtx, cancel := context.WithCancel(ctx)
	readyCtx, readyCancel := boundedInitContext(trackCtx, constructorTimeout)
	defer readyCancel()

	playbackURL, err := publicurl.PlaybackURL(mf.ID)
	if err != nil {
		cancel()
		return nil, err
	}
	sess, err := makeSession(readyCtx, target, loadSpec{ContentID: playbackURL, ContentType: mf.ContentType(), Duration: mf.Duration})
	if err != nil {
		cancel()
		return nil, err
	}
	track := &CastTrack{
		mediaFile:     mf,
		target:        target,
		playbackDone:  playbackDone,
		naturalFinish: make(chan struct{}, 1),
		trackCtx:      trackCtx,
		cancel:        cancel,
		session:       sess,
	}
	go track.sessionEventLoop()
	go track.playbackDoneNotifier()
	return track, nil
}

func boundedInitContext(parent context.Context, max time.Duration) (context.Context, context.CancelFunc) {
	if deadline, ok := parent.Deadline(); ok {
		limit := time.Now().Add(max)
		if deadline.Before(limit) {
			return context.WithDeadline(parent, deadline)
		}
	}
	return context.WithTimeout(parent, max)
}

func (t *CastTrack) String() string {
	name := t.target.Name
	if name == "" {
		name = t.target.Host
	}
	return fmt.Sprintf("CastTrack[target=%s mediaID=%s title=%q]", name, t.mediaFile.ID, t.mediaFile.Title)
}

func (t *CastTrack) IsPlaying() bool {
	snap := t.session.Snapshot()
	switch snap.PlayerState {
	case "PLAYING", "BUFFERING":
		return !snap.Closed && snap.Connected
	default:
		return false
	}
}

func (t *CastTrack) SetVolume(value float32) {
	ctx, cancel := context.WithTimeout(context.Background(), requestTimeout)
	defer cancel()
	if err := t.session.SetVolume(ctx, value); err != nil {
		log.Warn("Failed to set cast volume", "track", t, err)
	}
}

func (t *CastTrack) Pause() {
	ctx, cancel := context.WithTimeout(context.Background(), requestTimeout)
	defer cancel()
	if err := t.session.Pause(ctx); err != nil {
		log.Warn("Failed to pause cast track", "track", t, err)
	}
}

func (t *CastTrack) Unpause() {
	ctx, cancel := context.WithTimeout(context.Background(), requestTimeout)
	defer cancel()
	if err := t.session.Play(ctx); err != nil {
		log.Warn("Failed to unpause cast track", "track", t, err)
	}
}

func (t *CastTrack) Position() int {
	snap := t.session.Snapshot()
	pos := snap.CurrentTime
	if snap.PlayerState == "PLAYING" && snap.Connected && !snap.Closed && !snap.LastUpdate.IsZero() {
		pos += float32(time.Since(snap.LastUpdate).Seconds())
	}
	if snap.Duration > 0 && pos > snap.Duration {
		pos = snap.Duration
	}
	if pos < 0 {
		pos = 0
	}
	return int(pos)
}

func (t *CastTrack) SetPosition(offset int) error {
	snap := t.session.Snapshot()
	var resumeState *string
	if snap.PlayerState == "PLAYING" {
		start := "PLAYBACK_START"
		resumeState = &start
	}
	return t.session.SeekTo(t.trackCtx, float32(offset), resumeState)
}

func (t *CastTrack) SetReceiverVolumeHandler(handler func(float32)) {
	t.mu.Lock()
	defer t.mu.Unlock()
	if t.closing || t.trackCtx.Err() != nil {
		return
	}
	t.receiverVolumeHandler = handler
}

// Close wins over any not-yet-delivered natural completion. Once closing is set,
// PlaybackDone is suppressed even if FINISHED was already observed.
func (t *CastTrack) Close() {
	t.mu.Lock()
	if t.closing {
		t.mu.Unlock()
		return
	}
	t.closing = true
	t.receiverVolumeHandler = nil
	t.mu.Unlock()

	t.cancel()
	if err := t.session.Close(); err != nil {
		log.Warn("Failed to close cast track", "track", t, err)
	}
}

// sessionEventLoop owns runtime session terminal handling for the track. A
// remote disconnect or explicit session close cancels the track context so the
// completion notifier cannot remain parked forever after the Cast session dies.
func (t *CastTrack) sessionEventLoop() {
	for {
		select {
		case <-t.trackCtx.Done():
			return
		case ev, ok := <-t.session.Events():
			if !ok {
				t.cancel()
				return
			}
			switch ev.Type {
			case sessionEventMediaStatus:
				if ev.Snapshot.MediaSessionID != 0 && ev.Snapshot.PlayerState == "IDLE" && ev.Snapshot.IdleReason == "FINISHED" {
					t.queueNaturalCompletion()
				}
			case sessionEventReceiverStatus:
				t.handleReceiverVolume(ev.Snapshot.ReceiverVolume)
			case sessionEventDisconnected, sessionEventClosed, sessionEventLoadFailed:
				t.cancel()
				return
			}
		}
	}
}

func (t *CastTrack) handleReceiverVolume(level float32) {
	t.mu.Lock()
	if t.closing || t.trackCtx.Err() != nil {
		t.mu.Unlock()
		return
	}
	handler := t.receiverVolumeHandler
	t.mu.Unlock()
	if handler != nil {
		handler(level)
	}
}

// queueNaturalCompletion records that the receiver reported a natural FINISHED
// transition. Delivery to PlaybackDone happens later under the same mutex so the
// Close-vs-FINISHED tie-break is deterministic.
func (t *CastTrack) queueNaturalCompletion() {
	t.mu.Lock()
	defer t.mu.Unlock()
	if t.closing || t.finishQueued || t.doneDelivered || t.trackCtx.Err() != nil {
		return
	}
	t.finishQueued = true
	select {
	case t.naturalFinish <- struct{}{}:
	default:
	}
}

func (t *CastTrack) playbackDoneNotifier() {
	select {
	case <-t.trackCtx.Done():
		return
	case <-t.naturalFinish:
	}

	ticker := time.NewTicker(10 * time.Millisecond)
	defer ticker.Stop()
	for {
		t.mu.Lock()
		if t.closing || t.doneDelivered || t.trackCtx.Err() != nil {
			t.mu.Unlock()
			return
		}
		select {
		case t.playbackDone <- true:
			t.doneDelivered = true
			t.mu.Unlock()
			return
		default:
			t.mu.Unlock()
		}
		select {
		case <-t.trackCtx.Done():
			return
		case <-ticker.C:
		}
	}
}
