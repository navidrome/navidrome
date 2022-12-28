package scrobbler

import (
	"context"
	"errors"
	"time"

	"github.com/navidrome/navidrome/log"
	"github.com/navidrome/navidrome/model"
)

func newBufferedScrobbler(ds model.DataStore, s Scrobbler, service string) *bufferedScrobbler {
	b := &bufferedScrobbler{ds: ds, wrapped: s, service: service}
	b.wakeSignal = make(chan struct{}, 1)
	b.starChan = make(chan bufferedStar, 1)
	go b.runQueue()
	go b.runStar()
	return b
}

type bufferedStar struct {
	userId string
	star   bool
	tracks *Stars
}

type bufferedScrobbler struct {
	ds         model.DataStore
	wrapped    Scrobbler
	service    string
	wakeSignal chan struct{}
	starChan   chan bufferedStar
}

func (b *bufferedScrobbler) IsAuthorized(ctx context.Context, userId string) bool {
	return b.wrapped.IsAuthorized(ctx, userId)
}

func (b *bufferedScrobbler) NowPlaying(ctx context.Context, userId string, track *model.MediaFile) error {
	return b.wrapped.NowPlaying(ctx, userId, track)
}

func (b *bufferedScrobbler) Scrobble(ctx context.Context, userId string, s Scrobble) error {
	err := b.ds.ScrobbleBuffer(ctx).Enqueue(b.service, userId, s.ID, s.TimeStamp)
	if err != nil {
		return err
	}

	b.sendWakeSignal()
	return nil
}

func (b *bufferedScrobbler) CanProxyStars(ctx context.Context, userId string) bool {
	return b.wrapped.CanProxyStars(ctx, userId)
}

func (b *bufferedScrobbler) Star(ctx context.Context, userId string, star bool, tracks *Stars) error {
	// We don't want this to block other operations
	go func() {
		b.starChan <- bufferedStar{
			userId: userId,
			star:   star,
			tracks: tracks,
		}
	}()

	return nil
}

func (b *bufferedScrobbler) sendWakeSignal() {
	// Don't block if the previous signal was not read yet
	select {
	case b.wakeSignal <- struct{}{}:
	default:
	}
}

func (b *bufferedScrobbler) runQueue() {
	ctx := context.Background()
	for {
		if !b.processQueue(ctx) {
			time.AfterFunc(5*time.Second, func() {
				b.sendWakeSignal()
			})
		}
		<-b.wakeSignal
	}
}

func (b *bufferedScrobbler) runStar() {
	ctx := context.Background()

	for star := range b.starChan {
		err := b.wrapped.Star(ctx, star.userId, star.star, star.tracks)
		if err != nil {
			log.Error(ctx, "Error starring", "error", err)
		}
	}
}

func (b *bufferedScrobbler) processQueue(ctx context.Context) bool {
	buffer := b.ds.ScrobbleBuffer(ctx)
	userIds, err := buffer.UserIDs(b.service)
	if err != nil {
		log.Error(ctx, "Error retrieving userIds from scrobble buffer", "scrobbler", b.service, err)
		return false
	}
	result := true
	for _, userId := range userIds {
		if !b.processUserQueue(ctx, userId) {
			result = false
		}
	}
	return result
}

func (b *bufferedScrobbler) processUserQueue(ctx context.Context, userId string) bool {
	buffer := b.ds.ScrobbleBuffer(ctx)
	for {
		entry, err := buffer.Next(b.service, userId)
		if err != nil {
			log.Error(ctx, "Error reading from scrobble buffer", "scrobbler", b.service, err)
			return false
		}
		if entry == nil {
			return true
		}
		log.Debug(ctx, "Sending scrobble", "scrobbler", b.service, "track", entry.Title, "artist", entry.Artist)
		err = b.wrapped.Scrobble(ctx, entry.UserID, Scrobble{
			MediaFile: entry.MediaFile,
			TimeStamp: entry.PlayTime,
		})
		if errors.Is(err, ErrRetryLater) {
			log.Warn(ctx, "Could not send scrobble. Will be retried", "userId", entry.UserID,
				"track", entry.Title, "artist", entry.Artist, "scrobbler", b.service, err)
			return false
		}
		if err != nil {
			log.Error(ctx, "Error sending scrobble to service. Discarding", "scrobbler", b.service,
				"userId", entry.UserID, "artist", entry.Artist, "track", entry.Title, err)
		}
		err = buffer.Dequeue(entry)
		if err != nil {
			log.Error(ctx, "Error removing entry from scrobble buffer", "userId", entry.UserID,
				"track", entry.Title, "artist", entry.Artist, "scrobbler", b.service, err)
			return false
		}
	}
}

var _ Scrobbler = (*bufferedScrobbler)(nil)
