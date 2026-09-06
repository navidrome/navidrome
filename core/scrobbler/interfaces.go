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

type Constructor func(ds model.DataStore) Scrobbler
