package model

import (
	"context"
	"errors"
)

// TODO: Should the type be encoded in the ID?
func GetEntityByID(ctx context.Context, ds DataStore, id string) (any, error) {
	getters := []func() (any, error){
		func() (any, error) { return ds.Artist(ctx).Get(id) },
		func() (any, error) { return ds.Album(ctx).Get(id) },
		func() (any, error) { return ds.Playlist(ctx).Get(id) },
		func() (any, error) { return ds.MediaFile(ctx).Get(id) },
		func() (any, error) { return ds.Radio(ctx).Get(id) },
	}
	for _, get := range getters {
		entity, err := get()
		if err == nil {
			return entity, nil
		}
		if !errors.Is(err, ErrNotFound) {
			return nil, err
		}
	}
	return nil, ErrNotFound
}
