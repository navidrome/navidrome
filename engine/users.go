package engine

import (
	"context"
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"strings"

	"github.com/deluan/navidrome/log"
	"github.com/deluan/navidrome/model"
)

type Users interface {
	Authenticate(ctx context.Context, username, password, token, salt string) (*model.User, error)
}

func NewUsers(ds model.DataStore) Users {
	return &users{ds}
}

type users struct {
	ds model.DataStore
}

func (u *users) Authenticate(ctx context.Context, username, pass, token, salt string) (*model.User, error) {
	user, err := u.ds.User(ctx).FindByUsername(username)
	if err == model.ErrNotFound {
		return nil, model.ErrInvalidAuth
	}
	if err != nil {
		return nil, err
	}
	valid := false

	switch {
	case pass != "":
		if strings.HasPrefix(pass, "enc:") {
			if dec, err := hex.DecodeString(pass[4:]); err == nil {
				pass = string(dec)
			}
		}
		valid = pass == user.Password
	case token != "":
		t := fmt.Sprintf("%x", md5.Sum([]byte(user.Password+salt)))
		valid = t == token
	}

	if !valid {
		return nil, model.ErrInvalidAuth
	}
	go func() {
		err := u.ds.User(ctx).UpdateLastAccessAt(user.ID)
		if err != nil {
			log.Error(ctx, "Could not update user's lastAccessAt", "user", user.UserName)
		}
	}()
	return user, nil
}
