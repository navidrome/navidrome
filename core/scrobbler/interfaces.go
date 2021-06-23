package scrobbler

import (
	"context"
	"time"

	"github.com/navidrome/navidrome/model"
)

type Scrobble struct {
	model.MediaFile
	TimeStamp time.Time
}

type Scrobbler interface {
	IsAuthorized(ctx context.Context, userId string) bool
	NowPlaying(ctx context.Context, userId string, track *model.MediaFile) error
	Scrobble(ctx context.Context, userId string, scrobbles []Scrobble) error
}

type Constructor func(ds model.DataStore) Scrobbler
