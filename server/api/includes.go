package api

import (
	"context"
	"sort"

	"github.com/Masterminds/squirrel"
	"github.com/navidrome/navidrome/model"
	"golang.org/x/exp/slices"
)

type includedResources struct {
	ctx       context.Context
	ds        model.DataStore
	includes  *includeSlice
	resources []IncludedResource
}

func (i *includedResources) AddAlbums(albumIds ...string) error {
	if i.includes == nil || !slices.Contains(*i.includes, "album") {
		return nil
	}
	sort.Strings(albumIds)
	slices.Compact(albumIds)
	albums, err := i.ds.Album(i.ctx).GetAll(model.QueryOptions{Filters: squirrel.Eq{"id": albumIds}})
	if err != nil {
		return err
	}
	for _, al := range albums {
		inc := &IncludedResource{}
		_ = inc.FromAlbum(toAPIAlbum(al))
		i.resources = append(i.resources, *inc)
	}
	return nil
}

func (i *includedResources) AddArtists(artistIds ...string) error {
	if i.includes == nil || !slices.Contains(*i.includes, "artist") {
		return nil
	}
	sort.Strings(artistIds)
	slices.Compact(artistIds)
	artists, err := i.ds.Artist(i.ctx).GetAll(model.QueryOptions{Filters: squirrel.Eq{"artist.id": artistIds}})
	if err != nil {
		return err
	}
	for _, ar := range artists {
		inc := &IncludedResource{}
		_ = inc.FromArtist(toAPIArtist(ar))
		i.resources = append(i.resources, *inc)
	}
	return nil
}

func (i *includedResources) Build() *[]IncludedResource {
	if len(i.resources) == 0 {
		return nil
	}
	return &i.resources
}
