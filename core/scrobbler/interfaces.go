package scrobbler

import (
	"context"
	"time"

	"github.com/navidrome/navidrome/model"
)

type Scrobble struct {
	Track     *model.MediaFile
	TimeStamp *time.Time
}

type Scrobbler interface {
	NowPlaying(context.Context, *model.MediaFile) error
	Scrobble(context.Context, []Scrobble) error
}

type Constructor func(ds model.DataStore) Scrobbler
