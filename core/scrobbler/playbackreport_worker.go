package scrobbler

import (
	"context"

	"github.com/navidrome/navidrome/log"
)

func (p *playTracker) enqueuePlaybackReport(ctx context.Context, info PlaybackSession) {
	p.prMu.Lock()
	defer p.prMu.Unlock()
	ctx = context.WithoutCancel(ctx)
	p.prQueue = append(p.prQueue, playbackReportEntry{
		ctx:  ctx,
		info: info,
	})
	p.sendPlaybackReportSignal()
}

func (p *playTracker) sendPlaybackReportSignal() {
	select {
	case p.prSignal <- struct{}{}:
	default:
	}
}

func (p *playTracker) playbackReportWorker() {
	defer close(p.prWorkerDone)
	for {
		select {
		case <-p.shutdown:
			return
		case <-p.prSignal:
		}

		p.prMu.Lock()
		if len(p.prQueue) == 0 {
			p.prMu.Unlock()
			continue
		}
		entries := p.prQueue
		p.prQueue = nil
		p.prMu.Unlock()

		allScrobblers := p.getActiveScrobblers()
		for _, entry := range entries {
			p.dispatchPlaybackReport(entry.ctx, entry.info, allScrobblers)
		}
	}
}

func (p *playTracker) dispatchPlaybackReport(ctx context.Context, info PlaybackSession, allScrobblers map[string]Scrobbler) {
	for name, s := range allScrobblers {
		if !s.IsAuthorized(ctx, info.UserId) {
			continue
		}
		log.Debug(ctx, "Sending PlaybackReport", "scrobbler", name, "track", info.MediaFile.Title, "state", info.State, "positionMs", info.PositionMs)
		err := s.PlaybackReport(ctx, info)
		if err != nil {
			log.Error(ctx, "Error sending PlaybackReport", "scrobbler", name, "track", info.MediaFile.Title, "state", info.State, err)
			continue
		}
	}
}
