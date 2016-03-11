package engine

import (
	"strings"

	"github.com/deluan/gomate"
	"github.com/deluan/gosonic/domain"
)

type Search interface {
	ClearAll() error
	IndexArtist(ar *domain.Artist) error
	IndexAlbum(al *domain.Album) error
	IndexMediaFile(mf *domain.MediaFile) error

	SearchArtist(q string, offset int, size int) (*domain.Artists, error)
	//SearchAlbum(q string, offset int, size int) (*domain.Albums, error)
	//SearchSong(q string, offset int, size int) (*domain.MediaFiles, error)
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
	return s.idxArtist.Index(ar.Id, strings.ToLower(ar.Name))
}

func (s search) IndexAlbum(al *domain.Album) error {
	return s.idxAlbum.Index(al.Id, strings.ToLower(al.Name))
}

func (s search) IndexMediaFile(mf *domain.MediaFile) error {
	return s.idxSong.Index(mf.Id, strings.ToLower(mf.Title))
}

func (s search) SearchArtist(q string, offset int, size int) (*domain.Artists, error) {
	q = strings.TrimSuffix(q, "*")
	res, err := s.sArtist.Search(q)
	if err != nil {
		return nil, nil
	}
	as := make(domain.Artists, 0, len(res))
	for _, id := range res {
		a, err := s.artistRepo.Get(id)
		if err != nil {
			return nil, err
		}
		as = append(as, *a)
	}
	return &as, nil
}

//func (s search) SearchAlbum(q string, offset int, size int) (*domain.Albums, error) {
//	q := strings.TrimSuffix(q, "*")
//	return nil
//}
//
//func (s search) SearchSong(q string, offset int, size int) (*domain.MediaFiles, error) {
//	q := strings.TrimSuffix(q, "*")
//	return nil
//}
