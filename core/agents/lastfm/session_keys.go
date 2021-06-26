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

func (sk *sessionKeys) put(ctx context.Context, userId, sessionKey string) error {
	return sk.ds.UserProps(ctx).Put(userId, sessionKeyProperty, sessionKey)
}

func (sk *sessionKeys) get(ctx context.Context, userId string) (string, error) {
	return sk.ds.UserProps(ctx).Get(userId, sessionKeyProperty)
}

func (sk *sessionKeys) delete(ctx context.Context, userId string) error {
	return sk.ds.UserProps(ctx).Delete(userId, sessionKeyProperty)
}
