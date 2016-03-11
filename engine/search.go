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
}

type search struct {
	artistRepo domain.ArtistRepository
	albumRepo  domain.AlbumRepository
	mfileRepo  domain.MediaFileRepository
	indexer    gomate.Indexer
}

func NewSearch(ar domain.ArtistRepository, alr domain.AlbumRepository, mr domain.MediaFileRepository, idx gomate.Indexer) Search {
	return search{ar, alr, mr, idx}
}

func (s search) ClearAll() error {
	return s.indexer.Clear()
}

func (s search) IndexArtist(ar *domain.Artist) error {
	return s.indexer.Index("ar-"+ar.Id, strings.ToLower(ar.Name))
}

func (s search) IndexAlbum(al *domain.Album) error {
	return s.indexer.Index("al-"+al.Id, strings.ToLower(al.Name))
}

func (s search) IndexMediaFile(mf *domain.MediaFile) error {
	return s.indexer.Index("mf-"+mf.Id, strings.ToLower(mf.Title))
}
