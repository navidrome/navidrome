package auth

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/navidrome/navidrome/log"
	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/model/id"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"
)

var ErrSetupComplete = errors.New("setup already complete")

// CreateFirstAdmin counts and inserts in one locked transaction, so racing setups cannot both win.
// then, if not nil, runs in that same transaction with the new user.
func CreateFirstAdmin(ctx context.Context, ds model.DataStore, username, password string, then func(tx model.DataStore, u *model.User) error) (*model.User, error) {
	var created *model.User
	err := ds.WithTxImmediate(func(tx model.DataStore) error {
		count, err := tx.User().CountAll(ctx)
		if err != nil {
			return fmt.Errorf("counting users: %w", err)
		}
		if count > 0 {
			return ErrSetupComplete
		}
		log.Warn(ctx, "Creating initial user", "user", username)
		u := model.User{
			ID:          id.NewRandom(),
			UserName:    username,
			Name:        cases.Title(language.Und).String(username),
			NewPassword: password,
			IsAdmin:     true,
			LastLoginAt: new(time.Now()),
		}
		if err := tx.User().Put(ctx, &u); err != nil {
			return fmt.Errorf("creating initial user: %w", err)
		}
		if created, err = tx.User().Get(ctx, u.ID); err != nil {
			return err
		}
		if then != nil {
			return then(tx, created)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return created, nil
}
