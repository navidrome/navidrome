package apiauth

import (
	"context"
	"crypto/subtle"
	"errors"

	"github.com/navidrome/navidrome/model"
)

type Outcome int

const (
	NotMine Outcome = iota
	Authenticated
	Rejected
	Unavailable
)

type CredentialResult struct {
	Outcome       Outcome
	User          *model.User
	Provider      string
	PasswordLocal bool
}

type CredentialChecker interface {
	Check(ctx context.Context, username, password string) (CredentialResult, error)
}

// checkCredentials asks each checker in turn; only NotMine moves on, so an owning provider's "no" is final.
func checkCredentials(ctx context.Context, checkers []CredentialChecker, username, password string) (CredentialResult, error) {
	for _, c := range checkers {
		res, err := c.Check(ctx, username, password)
		if err != nil {
			return CredentialResult{}, err
		}
		switch res.Outcome {
		case NotMine:
			continue
		case Authenticated:
			return res, nil
		case Unavailable:
			return CredentialResult{}, model.ErrNotAvailable
		default:
			return CredentialResult{}, model.ErrInvalidAuth
		}
	}
	return CredentialResult{}, model.ErrInvalidAuth
}

type dbChecker struct {
	ds model.DataStore
}

func (c dbChecker) Check(ctx context.Context, username, password string) (CredentialResult, error) {
	u, err := c.ds.User().FindByUsernameWithPassword(ctx, username)
	if errors.Is(err, model.ErrNotFound) {
		return CredentialResult{Outcome: NotMine}, nil
	}
	if err != nil {
		return CredentialResult{}, err
	}
	if subtle.ConstantTimeCompare([]byte(u.Password), []byte(password)) != 1 {
		return CredentialResult{Outcome: Rejected}, nil
	}
	return CredentialResult{Outcome: Authenticated, User: u, Provider: "password", PasswordLocal: true}, nil
}
