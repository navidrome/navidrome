package scrobbler

import (
	"context"
	"errors"
	"time"

	"github.com/navidrome/navidrome/model"
)

type Scrobble struct {
	model.MediaFile
	TimeStamp time.Time
}

var (
	ErrNotAuthorized = errors.New("not authorized")
	ErrRetryLater    = errors.New("retry later")
	ErrUnrecoverable = errors.New("unrecoverable")
)

type Scrobbler interface {
	IsAuthorized(ctx context.Context, userId string) bool
	NowPlaying(ctx context.Context, userId string, track *model.MediaFile, position int) error
	Scrobble(ctx context.Context, userId string, s Scrobble) error
}

type Constructor func(ds model.DataStore) Scrobbler
