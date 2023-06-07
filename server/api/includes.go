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
	ids       map[ResourceType][]string
}

func newIncludedResources(ctx context.Context, ds model.DataStore, includes *includeSlice) *includedResources {
	i := &includedResources{
		ctx:      ctx,
		ds:       ds,
		includes: includes,
	}
	if includes != nil {
		i.ids = make(map[ResourceType][]string)
		for _, inc := range *includes {
			i.ids[ResourceType(inc)] = []string{}
		}
	}
	return i
}

func (i *includedResources) Tracks(trackIds ...string) {
	if i.ids == nil || i.ids[ResourceTypeTrack] == nil {
		return
	}
	i.ids[ResourceTypeTrack] = append(i.ids[ResourceTypeTrack], trackIds...)
}

func (i *includedResources) Albums(albumIds ...string) {
	if i.ids == nil || i.ids[ResourceTypeAlbum] == nil {
		return
	}
	i.ids[ResourceTypeAlbum] = append(i.ids[ResourceTypeAlbum], albumIds...)
}

func (i *includedResources) Artists(artistIds ...string) {
	if i.ids == nil || i.ids[ResourceTypeArtist] == nil {
		return
	}
	i.ids[ResourceTypeArtist] = append(i.ids[ResourceTypeArtist], artistIds...)
}

func (i *includedResources) addTracks(albumIds []string) error {
	tracks, err := i.ds.MediaFile(i.ctx).GetAll(model.QueryOptions{Filters: squirrel.Eq{"album_id": albumIds}})
	if err != nil {
		return err
	}
	for _, tr := range tracks {
		inc := &IncludedResource{}
		_ = inc.FromTrack(toAPITrack(tr))
		i.resources = append(i.resources, *inc)
	}
	return nil
}

func (i *includedResources) addAlbums(albumIds []string) error {
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

func (i *includedResources) addArtists(artistIds []string) error {
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

func (i *includedResources) Build() (*[]IncludedResource, error) {
	if i.includes == nil {
		return nil, nil
	}
	for _, typ := range *i.includes {
		ids := i.ids[ResourceType(typ)]
		sort.Strings(ids)
		slices.Compact(ids)
		if len(ids) == 0 {
			continue
		}
		switch ResourceType(typ) {
		case ResourceTypeAlbum:
			if err := i.addAlbums(ids); err != nil {
				return nil, err
			}
		case ResourceTypeArtist:
			if err := i.addArtists(ids); err != nil {
				return nil, err
			}
		case ResourceTypeTrack:
			if err := i.addTracks(ids); err != nil {
				return nil, err
			}
		}
	}

	return &i.resources, nil
}
