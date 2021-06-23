package lastfm

import (
	"context"

	"github.com/navidrome/navidrome/model"
)

const (
	sessionKeyProperty = "LastFMSessionKey"
)

// sessionKeys is a simple wrapper around the UserPropsRepository
type sessionKeys struct {
	ds model.DataStore
}

func (sk *sessionKeys) put(ctx context.Context, sessionKey string) error {
	return sk.ds.UserProps(ctx).Put(sessionKeyProperty, sessionKey)
}

func (sk *sessionKeys) get(ctx context.Context) (string, error) {
	return sk.ds.UserProps(ctx).Get(sessionKeyProperty)
}

func (sk *sessionKeys) delete(ctx context.Context) error {
	return sk.ds.UserProps(ctx).Delete(sessionKeyProperty)
}
