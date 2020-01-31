package engine

import (
	"context"
	"strings"

	"github.com/deluan/navidrome/model"
	"github.com/kennygrant/sanitize"
)

type Search interface {
	SearchArtist(ctx context.Context, q string, offset int, size int) (Entries, error)
	SearchAlbum(ctx context.Context, q string, offset int, size int) (Entries, error)
	SearchSong(ctx context.Context, q string, offset int, size int) (Entries, error)
}

type search struct {
	ds model.DataStore
}

func NewSearch(ds model.DataStore) Search {
	s := &search{ds}
	return s
}

func (s *search) SearchArtist(ctx context.Context, q string, offset int, size int) (Entries, error) {
	q = sanitize.Accents(strings.ToLower(strings.TrimSuffix(q, "*")))
	artists, err := s.ds.Artist(ctx).Search(q, offset, size)
	if len(artists) == 0 || err != nil {
		return nil, nil
	}

	artistIds := make([]string, len(artists))
	for i, al := range artists {
		artistIds[i] = al.ID
	}
	return FromArtists(artists), nil
}

func (s *search) SearchAlbum(ctx context.Context, q string, offset int, size int) (Entries, error) {
	q = sanitize.Accents(strings.ToLower(strings.TrimSuffix(q, "*")))
	albums, err := s.ds.Album(ctx).Search(q, offset, size)
	if len(albums) == 0 || err != nil {
		return nil, nil
	}

	albumIds := make([]string, len(albums))
	for i, al := range albums {
		albumIds[i] = al.ID
	}

	return FromAlbums(albums), nil
}

func (s *search) SearchSong(ctx context.Context, q string, offset int, size int) (Entries, error) {
	q = sanitize.Accents(strings.ToLower(strings.TrimSuffix(q, "*")))
	mediaFiles, err := s.ds.MediaFile(ctx).Search(q, offset, size)
	if len(mediaFiles) == 0 || err != nil {
		return nil, nil
	}

	trackIds := make([]string, len(mediaFiles))
	for i, mf := range mediaFiles {
		trackIds[i] = mf.ID
	}

	return FromMediaFiles(mediaFiles), nil
}
