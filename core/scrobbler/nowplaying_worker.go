package scrobbler

import (
	"context"
	"time"

	"github.com/navidrome/navidrome/consts"
	"github.com/navidrome/navidrome/log"
	"github.com/navidrome/navidrome/model"
)

func (p *playTracker) enqueueNowPlaying(ctx context.Context, playerId string, userId string, track *model.MediaFile, position int) {
	p.npMu.Lock()
	defer p.npMu.Unlock()
	ctx = context.WithoutCancel(ctx) // Prevent cancellation from affecting background processing
	p.npQueue[playerId] = nowPlayingEntry{
		ctx:      ctx,
		userId:   userId,
		track:    track,
		position: position,
	}
	p.sendNowPlayingSignal()
}

func (p *playTracker) sendNowPlayingSignal() {
	// Don't block if the previous signal was not read yet
	select {
	case p.npSignal <- struct{}{}:
	default:
	}
}

func (p *playTracker) nowPlayingWorker() {
	defer close(p.workerDone)
	for {
		select {
		case <-p.shutdown:
			return
		case <-time.After(time.Second):
		case <-p.npSignal:
		}

		p.npMu.Lock()
		if len(p.npQueue) == 0 {
			p.npMu.Unlock()
			continue
		}

		// Keep a copy of the entries to process and clear the queue
		entries := p.npQueue
		p.npQueue = make(map[string]nowPlayingEntry)
		p.npMu.Unlock()

		// Process entries without holding lock
		for _, entry := range entries {
			p.dispatchNowPlaying(entry.ctx, entry.userId, entry.track, entry.position)
		}
	}
}

func (p *playTracker) dispatchNowPlaying(ctx context.Context, userId string, t *model.MediaFile, position int) {
	if t.Artist == consts.UnknownArtist {
		log.Debug(ctx, "Ignoring external NowPlaying update for track with unknown artist", "track", t.Title, "artist", t.Artist)
		return
	}
	allScrobblers := p.getActiveScrobblers()
	for name, s := range allScrobblers {
		if !s.IsAuthorized(ctx, userId) {
			continue
		}
		log.Debug(ctx, "Sending NowPlaying update", "scrobbler", name, "track", t.Title, "artist", t.Artist, "position", position)
		err := s.NowPlaying(ctx, userId, t, position)
		if err != nil {
			log.Error(ctx, "Error sending PlaybackSession", "scrobbler", name, "track", t.Title, "artist", t.Artist, err)
			continue
		}
	}
}
