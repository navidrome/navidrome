package scrobbler

import (
	"context"
	"errors"
	"time"

	"github.com/navidrome/navidrome/core/agents"
	"github.com/navidrome/navidrome/model"
)

type Scrobble struct {
	model.MediaFile
	TimeStamp time.Time
}

var (
	ErrNotAuthorized = errors.New("not authorized")
	// ErrRetryLater is an alias of agents.ErrRetryLater so adapters and plugins share one identity.
	ErrRetryLater    = agents.ErrRetryLater
	ErrUnrecoverable = errors.New("unrecoverable")
)

type Scrobbler interface {
	IsAuthorized(ctx context.Context, userId string) bool
	NowPlaying(ctx context.Context, userId string, track *model.MediaFile, position int) error
	Scrobble(ctx context.Context, userId string, s Scrobble) error
	PlaybackReport(ctx context.Context, info PlaybackSession) error
}

// FavoriteScrobbler is an optional capability implemented by scrobblers that can sync
// track favorite/love state (e.g. Last.fm track.love / track.unlove).
type FavoriteScrobbler interface {
	SetFavorite(ctx context.Context, userId string, track *model.MediaFile, starred bool) error
}

// FavoriteSyncer is an optional interface implemented by PlayTracker to propagate
// song star/unstar events to active FavoriteScrobblers asynchronously.
type FavoriteSyncer interface {
	SyncFavorite(ctx context.Context, starred bool, trackIDs ...string)
}

type Constructor func(ds model.DataStore) Scrobbler
