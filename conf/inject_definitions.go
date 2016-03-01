package conf

import (
	"github.com/deluan/gosonic/repositories"
	"github.com/deluan/gosonic/utils"
	"github.com/karlkfi/inject"
)

var (
	indexRepository repositories.ArtistIndex
)

func init () {
	utils.Graph.Define(&indexRepository, inject.NewProvider(repositories.NewArtistIndexRepository))
}
