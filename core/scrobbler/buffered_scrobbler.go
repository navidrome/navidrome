package scrobbler

import (
	"context"
	"errors"
	"time"

	"github.com/navidrome/navidrome/log"
	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/utils/pl"
)

func newBufferedScrobbler(ds model.DataStore, s Scrobbler, service string) *bufferedScrobbler {
	b := &bufferedScrobbler{ds: ds, wrapped: s, service: service}
	b.starSignal = make(chan struct{}, 1)
	b.wakeSignal = make(chan struct{}, 1)

	go b.run(context.TODO())
	go b.runStar(context.TODO())
	return b
}

type bufferedScrobbler struct {
	ds         model.DataStore
	wrapped    Scrobbler
	service    string
	wakeSignal chan struct{}
	starSignal chan struct{}
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

func (b *bufferedScrobbler) sendWakeSignal() {
	// Don't block if the previous signal was not read yet
	select {
	case b.wakeSignal <- struct{}{}:
	default:
	}
}

func (b *bufferedScrobbler) CanProxyStars(ctx context.Context, userId string) bool {
	return b.wrapped.CanProxyStars(ctx, userId)
}

func (b *bufferedScrobbler) CanStar(track *model.MediaFile) bool {
	return b.wrapped.CanStar(track)
}

func (b *bufferedScrobbler) Star(ctx context.Context, userId string, isStar bool, track *model.MediaFile) error {
	err := b.ds.WithTx(func(tx model.DataStore) error {
		exists, err := tx.StarBuffer(ctx).TryUpdate(b.service, userId, track.ID, isStar)

		if err != nil {
			return err
		} else if !exists {
			err = tx.StarBuffer(ctx).Enqueue(b.service, userId, track.ID, isStar)
			if err != nil {
				return err
			}
		}
		return nil
	})

	if err != nil {
		return err
	}

	b.sendStarSignal()
	return nil
}

func (b *bufferedScrobbler) sendStarSignal() {
	// Don't block if the previous signal was not read yet
	select {
	case b.starSignal <- struct{}{}:
	default:
	}
}

func (b *bufferedScrobbler) run(ctx context.Context) {
	for {
		if !b.processScrobbleQueue(ctx) {
			time.AfterFunc(5*time.Second, func() {
				b.sendWakeSignal()
			})
		}
		<-pl.ReadOrDone(ctx, b.wakeSignal)
	}
}

func (b *bufferedScrobbler) processScrobbleQueue(ctx context.Context) bool {
	buffer := b.ds.ScrobbleBuffer(ctx)
	userIds, err := buffer.UserIDs(b.service)
	if err != nil {
		log.Error(ctx, "Error retrieving userIds from scrobble buffer", "scrobbler", b.service, err)
		return false
	}
	result := true
	for _, userId := range userIds {
		if !b.processUserScrobbleQueue(ctx, userId) {
			result = false
		}
	}
	return result
}

func (b *bufferedScrobbler) processUserScrobbleQueue(ctx context.Context, userId string) bool {
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

func (b *bufferedScrobbler) runStar(ctx context.Context) {
	for {
		if !b.processStarQueue(ctx) {
			time.AfterFunc(5*time.Second, func() {
				b.sendStarSignal()
			})
		}
		<-b.starSignal
	}
}

func (b *bufferedScrobbler) processStarQueue(ctx context.Context) bool {
	buffer := b.ds.StarBuffer(ctx)
	userIds, err := buffer.UserIDs(b.service)
	if err != nil {
		log.Error(ctx, "Error retrieving userIds from scrobble buffer", "scrobbler", b.service, err)
		return false
	}
	result := true
	for _, userId := range userIds {
		if !b.processUserStarQueue(ctx, userId) {
			result = false
		}
	}
	return result
}

func (b *bufferedScrobbler) processUserStarQueue(ctx context.Context, userId string) bool {
	buffer := b.ds.StarBuffer(ctx)
	for {
		entry, err := buffer.Next(b.service, userId)
		if err != nil {
			log.Error(ctx, "Error reading from scrobble buffer", "scrobbler", b.service, err)
			return false
		}
		if entry == nil {
			return true
		}
		log.Debug(ctx, "Sending star", "service", b.service, "track", entry.Title, "artist", entry.Artist)
		err = b.wrapped.Star(ctx, userId, entry.IsStar, &entry.MediaFile)
		if errors.Is(err, ErrRetryLater) {
			log.Warn(ctx, "Could not send star. Will be retried", "userId", entry.UserID,
				"track", entry.Title, "artist", entry.Artist, "service", b.service, err)
			return false
		}
		if err != nil {
			log.Error(ctx, "Error sending star to service. Discarding", "service", b.service,
				"userId", entry.UserID, "artist", entry.Artist, "track", entry.Title, err)
		}
		err = buffer.Dequeue(entry)
		if err != nil {
			log.Error(ctx, "Error removing entry from star buffer", "userId", entry.UserID,
				"track", entry.Title, "artist", entry.Artist, "service", b.service, err)
			return false
		}
	}
}

var _ Scrobbler = (*bufferedScrobbler)(nil)
