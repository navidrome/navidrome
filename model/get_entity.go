package model

import (
	"context"
)

// TODO: Should the type be encoded in the ID?
func GetEntityByID(ctx context.Context, ds DataStore, id string) (interface{}, error) {
	ar, err := ds.Artist(ctx).Get(id)
	if err == nil {
		return ar, nil
	}
	al, err := ds.Album(ctx).Get(id)
	if err == nil {
		return al, nil
	}
	pls, err := ds.Playlist(ctx).Get(id)
	if err == nil {
		return pls, nil
	}
	mf, err := ds.MediaFile(ctx).Get(id)
	if err == nil {
		return mf, nil
	}
	return nil, err
}
