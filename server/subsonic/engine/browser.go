package engine

import (
	"context"
	"sort"
	"strings"

	"github.com/deluan/navidrome/model"
)

type Browser interface {
	GetSong(ctx context.Context, id string) (*Entry, error)
	GetGenres(ctx context.Context) (model.Genres, error)
}

func NewBrowser(ds model.DataStore) Browser {
	return &browser{ds}
}

type browser struct {
	ds model.DataStore
}

func (b *browser) GetSong(ctx context.Context, id string) (*Entry, error) {
	mf, err := b.ds.MediaFile(ctx).Get(id)
	if err != nil {
		return nil, err
	}

	entry := FromMediaFile(mf)
	return &entry, nil
}

func (b *browser) GetGenres(ctx context.Context) (model.Genres, error) {
	genres, err := b.ds.Genre(ctx).GetAll()
	for i, g := range genres {
		if strings.TrimSpace(g.Name) == "" {
			genres[i].Name = "<Empty>"
		}
	}
	sort.Slice(genres, func(i, j int) bool {
		return genres[i].Name < genres[j].Name
	})
	return genres, err
}
