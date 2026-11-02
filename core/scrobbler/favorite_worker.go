package scrobbler

import (
	"context"
	"errors"

	"github.com/navidrome/navidrome/consts"
	"github.com/navidrome/navidrome/log"
	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/model/request"
)

const (
	// maxFavoriteSyncBatch bounds how many tracks a single star/unstar request may queue
	// for external favorite synchronization.
	maxFavoriteSyncBatch = 100
	// maxFavoriteQueueSize caps the total number of pending per-(user, track) favorite updates.
	maxFavoriteQueueSize = 500
	// maxFavoritePerUserQueueSize caps how many pending updates a single user may hold.
	// When a user exceeds their own quota, that same user's oldest pending update is evicted.
	maxFavoritePerUserQueueSize = maxFavoriteSyncBatch
	// maxFavoriteRetries bounds how many times a transient ErrRetryLater failure is retried.
	maxFavoriteRetries = 1
)

type favoriteKey struct {
	userId  string
	trackId string
}

type favoriteEntry struct {
	ctx      context.Context
	userId   string
	track    *model.MediaFile
	starred  bool
	attempts int
}

type favoriteCapable interface {
	SupportsFavorite() bool
}

func supportsFavorite(s Scrobbler) bool {
	if fc, ok := s.(favoriteCapable); ok {
		return fc.SupportsFavorite()
	}
	_, ok := s.(FavoriteScrobbler)
	return ok
}

func (p *playTracker) hasFavoriteScrobbler() bool {
	for _, s := range p.getActiveScrobblers() {
		if supportsFavorite(s) {
			return true
		}
	}
	return false
}

func (p *playTracker) SyncFavorite(ctx context.Context, starred bool, trackIDs ...string) {
	if len(trackIDs) == 0 {
		return
	}
	u, ok := request.UserFrom(ctx)
	if !ok || u.ID == "" {
		return
	}
	if len(trackIDs) > maxFavoriteSyncBatch {
		log.Warn(ctx, "Skipping external favorite sync for oversized batch", "count", len(trackIDs), "max", maxFavoriteSyncBatch, "user", u.UserName)
		return
	}
	if !p.hasFavoriteScrobbler() {
		return
	}

	for _, trackID := range trackIDs {
		mf, err := p.ds.MediaFile().GetWithParticipants(ctx, trackID)
		if err != nil || mf == nil {
			log.Warn(ctx, "Cannot find track for favorite sync", "id", trackID, "user", u.UserName, err)
			continue
		}
		if mf.Artist == consts.UnknownArtist {
			log.Debug(ctx, "Ignoring external favorite sync for track with unknown artist", "track", mf.Title)
			continue
		}
		if p.isFilteredOut(ctx, mf) {
			log.Debug(ctx, "Ignoring external favorite sync for filtered track", "track", mf.Title, "artist", mf.Artist)
			continue
		}
		p.enqueueFavorite(ctx, u.ID, mf, starred)
	}
}

func (p *playTracker) enqueueFavorite(ctx context.Context, userId string, track *model.MediaFile, starred bool) {
	p.favMu.Lock()
	defer p.favMu.Unlock()
	ctx = context.WithoutCancel(ctx)
	key := favoriteKey{userId: userId, trackId: track.ID}
	if _, exists := p.favPending[key]; !exists {
		pendingForUser := 0
		for _, pendingKey := range p.favOrder {
			if pendingKey.userId == userId {
				pendingForUser++
			}
		}
		if pendingForUser >= maxFavoritePerUserQueueSize {
			// Evict this same user's oldest pending update rather than discarding the incoming
			// one or consuming shared queue capacity needed by other users.
			evicted := p.evictOldestForUser(userId)
			log.Warn(ctx, "Per-user favorite sync quota reached, evicted oldest pending track",
				"userId", userId, "evictedTrackId", evicted.trackId, "newTrack", track.Title)
		} else if len(p.favOrder) >= maxFavoriteQueueSize {
			// Global cap reached across many users: evict the oldest pending update from the
			// user currently holding the largest backlog so no user's update is dropped due to
			// other users' traffic.
			evicted := p.evictFromHeaviestUser()
			log.Warn(ctx, "Global favorite sync queue full, evicted oldest track from heaviest backlog",
				"evictedUserId", evicted.userId, "evictedTrackId", evicted.trackId, "newTrack", track.Title, "userId", userId)
		}
		p.favOrder = append(p.favOrder, key)
	}
	p.favPending[key] = favoriteEntry{
		ctx:     ctx,
		userId:  userId,
		track:   track,
		starred: starred,
	}
	p.sendFavoriteSignal()
}

func (p *playTracker) evictOldestForUser(userId string) favoriteKey {
	for i, k := range p.favOrder {
		if k.userId == userId {
			p.favOrder = append(p.favOrder[:i], p.favOrder[i+1:]...)
			delete(p.favPending, k)
			return k
		}
	}
	return favoriteKey{}
}

func (p *playTracker) evictFromHeaviestUser() favoriteKey {
	counts := make(map[string]int)
	var heaviestUser string
	maxCount := 0
	for _, k := range p.favOrder {
		counts[k.userId]++
		if counts[k.userId] > maxCount {
			maxCount = counts[k.userId]
			heaviestUser = k.userId
		}
	}
	if heaviestUser != "" {
		return p.evictOldestForUser(heaviestUser)
	}
	return favoriteKey{}
}

func (p *playTracker) dequeueNextFavorite(lastServed map[string]uint64, seq *uint64) (favoriteEntry, bool) {
	p.favMu.Lock()
	defer p.favMu.Unlock()
	if len(p.favOrder) == 0 {
		clear(lastServed)
		return favoriteEntry{}, false
	}
	// Select the earliest queued item belonging to the least-recently-served user
	// (round-robin across users, FIFO within each user) to prevent head-of-line blocking.
	bestIdx := 0
	bestUserSeq := lastServed[p.favOrder[0].userId]
	seenUsers := map[string]bool{p.favOrder[0].userId: true}
	for i := 1; i < len(p.favOrder); i++ {
		uid := p.favOrder[i].userId
		if seenUsers[uid] {
			continue
		}
		seenUsers[uid] = true
		if userSeq := lastServed[uid]; userSeq < bestUserSeq {
			bestIdx = i
			bestUserSeq = userSeq
		}
	}
	key := p.favOrder[bestIdx]
	p.favOrder = append(p.favOrder[:bestIdx], p.favOrder[bestIdx+1:]...)
	entry, ok := p.favPending[key]
	delete(p.favPending, key)
	*seq++
	lastServed[key.userId] = *seq
	return entry, ok
}

func (p *playTracker) sendFavoriteSignal() {
	select {
	case p.favSignal <- struct{}{}:
	default:
	}
}

func (p *playTracker) favoriteWorker() {
	defer close(p.favWorkerDone)
	lastServed := make(map[string]uint64)
	var seq uint64
	for {
		select {
		case <-p.shutdown:
			return
		case <-p.favSignal:
		}

		for {
			select {
			case <-p.shutdown:
				return
			default:
			}

			entry, ok := p.dequeueNextFavorite(lastServed, &seq)
			if !ok {
				p.favMu.Lock()
				empty := len(p.favOrder) == 0
				p.favMu.Unlock()
				if empty {
					break
				}
				continue
			}

			if p.dispatchFavorite(entry) && entry.attempts < maxFavoriteRetries {
				key := favoriteKey{userId: entry.userId, trackId: entry.track.ID}
				p.favMu.Lock()
				// Only retry if no newer star/unstar for the same (userId, trackId) arrived in the meantime.
				if _, superseded := p.favPending[key]; !superseded && len(p.favOrder) < maxFavoriteQueueSize {
					entry.attempts++
					p.favPending[key] = entry
					p.favOrder = append(p.favOrder, key)
					p.sendFavoriteSignal()
				}
				p.favMu.Unlock()
			}
		}
	}
}

func (p *playTracker) dispatchFavorite(entry favoriteEntry) bool {
	allScrobblers := p.getActiveScrobblers()
	shouldRetry := false
	for name, s := range allScrobblers {
		if !supportsFavorite(s) {
			continue
		}
		fs, ok := s.(FavoriteScrobbler)
		if !ok || !s.IsAuthorized(entry.ctx, entry.userId) {
			continue
		}
		callCtx, cancel := context.WithTimeout(entry.ctx, consts.DefaultHttpClientTimeOut)
		log.Debug(callCtx, "Syncing favorite state to external service",
			"scrobbler", name, "track", entry.track.Title, "artist", entry.track.Artist, "starred", entry.starred)
		err := fs.SetFavorite(callCtx, entry.userId, entry.track, entry.starred)
		cancel()
		if err != nil {
			log.Warn(entry.ctx, "Error syncing favorite state to external service",
				"scrobbler", name, "track", entry.track.Title, "artist", entry.track.Artist, "starred", entry.starred, err)
			if errors.Is(err, ErrRetryLater) {
				shouldRetry = true
			}
		}
	}
	return shouldRetry
}
