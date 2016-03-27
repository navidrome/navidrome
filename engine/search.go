package engine

import (
	"strings"

	"github.com/astaxie/beego"
	"github.com/deluan/gomate"
	"github.com/deluan/gosonic/domain"
	"github.com/kennygrant/sanitize"
)

type Search interface {
	ClearAll() error
	IndexArtist(ar *domain.Artist) error
	IndexAlbum(al *domain.Album) error
	IndexMediaFile(mf *domain.MediaFile) error

	RemoveArtist(ids ...string) error
	RemoveAlbum(ids ...string) error
	RemoveMediaFile(ids ...string) error

	SearchArtist(q string, offset int, size int) (Entries, error)
	SearchAlbum(q string, offset int, size int) (Entries, error)
	SearchSong(q string, offset int, size int) (Entries, error)
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
	s := &search{artistRepo: ar, albumRepo: alr, mfileRepo: mr}
	s.idxArtist = gomate.NewIndexer(db, "gomate-artist-idx")
	s.sArtist = gomate.NewSearcher(db, "gomate-artist-idx")
	s.idxAlbum = gomate.NewIndexer(db, "gomate-album-idx")
	s.sAlbum = gomate.NewSearcher(db, "gomate-album-idx")
	s.idxSong = gomate.NewIndexer(db, "gomate-song-idx")
	s.sSong = gomate.NewSearcher(db, "gomate-song-idx")
	return s
}

func (s *search) ClearAll() error {
	if err := s.idxArtist.Clear(); err != nil {
		return err
	}
	if err := s.idxAlbum.Clear(); err != nil {
		return err
	}
	if err := s.idxSong.Clear(); err != nil {
		return err
	}
	return nil
}

func (s *search) IndexArtist(ar *domain.Artist) error {
	return s.idxArtist.Index(ar.Id, sanitize.Accents(strings.ToLower(ar.Name)))
}

func (s *search) IndexAlbum(al *domain.Album) error {
	return s.idxAlbum.Index(al.Id, sanitize.Accents(strings.ToLower(al.Name)))
}

func (s *search) IndexMediaFile(mf *domain.MediaFile) error {
	return s.idxSong.Index(mf.Id, sanitize.Accents(strings.ToLower(mf.Title)))
}

func (s *search) RemoveArtist(ids ...string) error {
	return s.idxArtist.Remove(ids...)
}

func (s *search) RemoveAlbum(ids ...string) error {
	return s.idxAlbum.Remove(ids...)
}

func (s *search) RemoveMediaFile(ids ...string) error {
	return s.idxSong.Remove(ids...)
}

func (s *search) SearchArtist(q string, offset int, size int) (Entries, error) {
	q = sanitize.Accents(strings.ToLower(strings.TrimSuffix(q, "*")))
	min := offset
	max := min + size - 1
	resp, err := s.sArtist.Search(q, min, max)
	if err != nil {
		return nil, nil
	}
	res := make(Entries, 0, len(resp))
	for _, id := range resp {
		a, err := s.artistRepo.Get(id)
		if criticalError("Artist", id, err) {
			return nil, err
		}
		if err == nil {
			res = append(res, Entry{Id: a.Id, Title: a.Name, IsDir: true})
		}
	}
	return res, nil
}

func (s *search) SearchAlbum(q string, offset int, size int) (Entries, error) {
	q = sanitize.Accents(strings.ToLower(strings.TrimSuffix(q, "*")))
	min := offset
	max := min + size - 1
	resp, err := s.sAlbum.Search(q, min, max)
	if err != nil {
		return nil, nil
	}
	res := make(Entries, 0, len(resp))
	for _, id := range resp {
		al, err := s.albumRepo.Get(id)
		if criticalError("Album", id, err) {
			return nil, err
		}
		if err == nil {
			res = append(res, FromAlbum(al))
		}
	}
	return res, nil
}

func (s *search) SearchSong(q string, offset int, size int) (Entries, error) {
	q = sanitize.Accents(strings.ToLower(strings.TrimSuffix(q, "*")))
	min := offset
	max := min + size - 1
	resp, err := s.sSong.Search(q, min, max)
	if err != nil {
		return nil, nil
	}
	res := make(Entries, 0, len(resp))
	for _, id := range resp {
		mf, err := s.mfileRepo.Get(id)
		if criticalError("Song", id, err) {
			return nil, err
		}
		if err == nil {
			res = append(res, FromMediaFile(mf))
		}
	}
	return res, nil
}

func criticalError(kind, id string, err error) bool {
	switch {
	case err != nil:
		return true
	case err == domain.ErrNotFound:
		beego.Warn(kind, "Id", id, "not in the DB. Need a reindex?")
	}
	return false
}
