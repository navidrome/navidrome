package engine

import (
	"strings"

	"github.com/deluan/gomate"
	"github.com/deluan/gosonic/domain"
)

type Search interface {
	IndexArtist(ar *domain.Artist) error
	//IndexAlbum(al domain.Album) error
	//IndexMediaFile(mf domain.MediaFile) error
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

func (s search) IndexArtist(ar *domain.Artist) error {
	return s.indexer.Index(ar.Id, strings.ToLower(ar.Name))
}
