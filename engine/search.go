package engine

import (
	"context"
	"strings"

	"github.com/cloudsonic/sonic-server/model"
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
	resp, err := s.ds.Artist().Search(q, offset, size)
	if err != nil {
		return nil, nil
	}
	res := make(Entries, 0, len(resp))
	for _, ar := range resp {
		res = append(res, FromArtist(&ar))
	}
	return res, nil
}

func (s *search) SearchAlbum(ctx context.Context, q string, offset int, size int) (Entries, error) {
	q = sanitize.Accents(strings.ToLower(strings.TrimSuffix(q, "*")))
	resp, err := s.ds.Album().Search(q, offset, size)
	if err != nil {
		return nil, nil
	}
	res := make(Entries, 0, len(resp))
	for _, al := range resp {
		res = append(res, FromAlbum(&al))
	}
	return res, nil
}

func (s *search) SearchSong(ctx context.Context, q string, offset int, size int) (Entries, error) {
	q = sanitize.Accents(strings.ToLower(strings.TrimSuffix(q, "*")))
	resp, err := s.ds.MediaFile().Search(q, offset, size)
	if err != nil {
		return nil, nil
	}
	res := make(Entries, 0, len(resp))
	for _, mf := range resp {
		res = append(res, FromMediaFile(&mf))
	}
	return res, nil
}
