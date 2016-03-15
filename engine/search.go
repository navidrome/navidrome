package engine

import (
	"strings"

	"github.com/deluan/gomate"
	"github.com/deluan/gosonic/domain"
	"github.com/kennygrant/sanitize"
)

type Results Entries

type Search interface {
	ClearAll() error
	IndexArtist(ar *domain.Artist) error
	IndexAlbum(al *domain.Album) error
	IndexMediaFile(mf *domain.MediaFile) error

	SearchArtist(q string, offset int, size int) (*Results, error)
	SearchAlbum(q string, offset int, size int) (*Results, error)
	SearchSong(q string, offset int, size int) (*Results, error)
}

type search struct {
	artistRepo domain.ArtistRepository
	albumRepo  domain.AlbumRepository
	mfileRepo  domain.MediaFileRepository
	idxArtist  gomate.Indexer
	idxAlbum   gomate.Indexer
	idxSong    gomate.Indexer
	sArtist    gomate.Searcher
	sAlbum     gomate.Searcher
	sSong      gomate.Searcher
}

func NewSearch(ar domain.ArtistRepository, alr domain.AlbumRepository, mr domain.MediaFileRepository, db gomate.DB) Search {
	s := search{artistRepo: ar, albumRepo: alr, mfileRepo: mr}
	s.idxArtist = gomate.NewIndexer(db, "gomate-artist-idx")
	s.sArtist = gomate.NewSearcher(db, "gomate-artist-idx")
	s.idxAlbum = gomate.NewIndexer(db, "gomate-album-idx")
	s.sAlbum = gomate.NewSearcher(db, "gomate-album-idx")
	s.idxSong = gomate.NewIndexer(db, "gomate-song-idx")
	s.sSong = gomate.NewSearcher(db, "gomate-song-idx")
	return s
}

func (s search) ClearAll() error {
	return s.idxArtist.Clear()
	return s.idxAlbum.Clear()
	return s.idxSong.Clear()
}

func (s search) IndexArtist(ar *domain.Artist) error {
	return s.idxArtist.Index(ar.Id, sanitize.Accents(strings.ToLower(ar.Name)))
}

func (s search) IndexAlbum(al *domain.Album) error {
	return s.idxAlbum.Index(al.Id, sanitize.Accents(strings.ToLower(al.Name)))
}

func (s search) IndexMediaFile(mf *domain.MediaFile) error {
	return s.idxSong.Index(mf.Id, sanitize.Accents(strings.ToLower(mf.Title)))
}

func (s search) SearchArtist(q string, offset int, size int) (*Results, error) {
	q = sanitize.Accents(strings.ToLower(strings.TrimSuffix(q, "*")))
	min := offset
	max := min + size - 1
	resp, err := s.sArtist.Search(q, min, max)
	if err != nil {
		return nil, nil
	}
	res := make(Results, len(resp))
	for i, id := range resp {
		a, err := s.artistRepo.Get(id)
		if err != nil {
			return nil, err
		}
		res[i] = Entry{Id: a.Id, Title: a.Name, IsDir: true}
	}
	return &res, nil
}

func (s search) SearchAlbum(q string, offset int, size int) (*Results, error) {
	q = sanitize.Accents(strings.ToLower(strings.TrimSuffix(q, "*")))
	min := offset
	max := min + size - 1
	resp, err := s.sAlbum.Search(q, min, max)
	if err != nil {
		return nil, nil
	}
	res := make(Results, len(resp))
	for i, id := range resp {
		al, err := s.albumRepo.Get(id)
		if err != nil {
			return nil, err
		}
		res[i] = FromAlbum(al)
	}
	return &res, nil
}

func (s search) SearchSong(q string, offset int, size int) (*Results, error) {
	q = sanitize.Accents(strings.ToLower(strings.TrimSuffix(q, "*")))
	min := offset
	max := min + size - 1
	resp, err := s.sSong.Search(q, min, max)
	if err != nil {
		return nil, nil
	}
	res := make(Results, len(resp))
	for i, id := range resp {
		mf, err := s.mfileRepo.Get(id)
		if err != nil {
			return nil, err
		}
		res[i] = FromMediaFile(mf)
	}
	return &res, nil
}
