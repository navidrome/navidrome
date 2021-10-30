package agents

import (
	"context"

	"github.com/navidrome/navidrome/model"
)

// SessionKeys is a simple wrapper around the UserPropsRepository
type SessionKeys struct {
	model.DataStore
	KeyName string
}

func (sk *SessionKeys) Put(ctx context.Context, userId, sessionKey string) error {
	return sk.DataStore.UserProps(ctx).Put(userId, sk.KeyName, sessionKey)
}

func (sk *SessionKeys) Get(ctx context.Context, userId string) (string, error) {
	return sk.DataStore.UserProps(ctx).Get(userId, sk.KeyName)
}

func (sk *SessionKeys) Delete(ctx context.Context, userId string) error {
	return sk.DataStore.UserProps(ctx).Delete(userId, sk.KeyName)
}
