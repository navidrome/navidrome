package conf

import (
	"github.com/deluan/gosonic/repositories"
	"github.com/deluan/gosonic/utils"
	"github.com/karlkfi/inject"
)

func define(ptr interface{}, constructor interface{}, argPtrs ...interface{}) {
	utils.Graph.Define(ptr, inject.NewProvider(constructor, argPtrs...))
}

func init () {
	define(new(repositories.ArtistIndex), repositories.NewArtistIndexRepository)
}
