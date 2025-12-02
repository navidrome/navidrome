package cmd

import (
	"context"
	"errors"
	"fmt"

	"github.com/navidrome/navidrome/core/auth"
	"github.com/navidrome/navidrome/db"
	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/persistence"
)

func getContext() (model.DataStore, context.Context) {
	sqlDB := db.Db()
	ds := persistence.New(sqlDB)
	return ds, auth.WithAdminUser(context.Background(), ds)
}

func getUser(ctx context.Context, id string, ds model.DataStore) (*model.User, error) {
	user, err := ds.User(ctx).FindByUsername(id)

	if err != nil && !errors.Is(err, model.ErrNotFound) {
		return nil, fmt.Errorf("finding user by name: %w", err)
	}

	if errors.Is(err, model.ErrNotFound) {
		user, err = ds.User(ctx).Get(id)
		if err != nil {
			return nil, fmt.Errorf("finding user by id: %w", err)
		}
	}

	return user, nil
}
