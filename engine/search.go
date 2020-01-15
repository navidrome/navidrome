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
	artistRepo model.ArtistRepository
	albumRepo  model.AlbumRepository
	mfileRepo  model.MediaFileRepository
}

func NewSearch(ar model.ArtistRepository, alr model.AlbumRepository, mr model.MediaFileRepository) Search {
	s := &search{artistRepo: ar, albumRepo: alr, mfileRepo: mr}
	return s
}

func (s *search) SearchArtist(ctx context.Context, q string, offset int, size int) (Entries, error) {
	q = sanitize.Accents(strings.ToLower(strings.TrimSuffix(q, "*")))
	resp, err := s.artistRepo.Search(q, offset, size)
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
	resp, err := s.albumRepo.Search(q, offset, size)
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
	resp, err := s.mfileRepo.Search(q, offset, size)
	if err != nil {
		return nil, nil
	}
	res := make(Entries, 0, len(resp))
	for _, mf := range resp {
		res = append(res, FromMediaFile(&mf))
	}
	return res, nil
}
