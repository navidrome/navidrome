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

// CreateFirstAdmin must run inside ds.WithTxImmediate, so the count and the insert cannot interleave.
func CreateFirstAdmin(ctx context.Context, tx model.DataStore, username, password string) (*model.User, error) {
	count, err := tx.User().CountAll(ctx)
	if err != nil {
		return nil, fmt.Errorf("counting users: %w", err)
	}
	if count > 0 {
		return nil, ErrSetupComplete
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
		return nil, fmt.Errorf("creating initial user: %w", err)
	}
	return tx.User().Get(ctx, u.ID)
}
