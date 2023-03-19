package agents

import (
	"context"
	"errors"

	"github.com/navidrome/navidrome/model"
)

const (
	UserSuffix = "-user"
)

// SessionKeys is a simple wrapper around the UserPropsRepository
type SessionKeys struct {
	model.DataStore
	KeyName string
}

func (sk *SessionKeys) Put(ctx context.Context, userId, sessionKey string) error {
	return sk.DataStore.UserProps(ctx).Put(userId, sk.KeyName, sessionKey)
}

func (sk *SessionKeys) PutWithUser(ctx context.Context, userId, sessionKey, remoteUser string) error {
	return sk.WithTx(func(tx model.DataStore) error {
		err := tx.UserProps(ctx).Put(userId, sk.KeyName, sessionKey)

		if err != nil {
			return err
		}
		return tx.UserProps(ctx).Put(userId, sk.KeyName+UserSuffix, remoteUser)
	})
}

func (sk *SessionKeys) Get(ctx context.Context, userId string) (string, error) {
	return sk.DataStore.UserProps(ctx).Get(userId, sk.KeyName)
}

type KeyWithUser struct {
	Key, User string
}

var (
	ErrNoUsername = errors.New("Token with no username")
)

func (sk *SessionKeys) GetWithUser(ctx context.Context, userId string) (KeyWithUser, error) {
	result := KeyWithUser{}
	err := sk.WithTx(func(tx model.DataStore) error {
		key, err := tx.UserProps(ctx).Get(userId, sk.KeyName)

		if err != nil {
			return err
		}

		result.Key = key

		user, err := tx.UserProps(ctx).Get(userId, sk.KeyName+UserSuffix)

		if err != nil {
			return ErrNoUsername
		}

		result.User = user
		return nil
	})

	return result, err
}

func (sk *SessionKeys) Delete(ctx context.Context, userId string) error {
	return sk.DataStore.UserProps(ctx).Delete(userId, sk.KeyName)
}
