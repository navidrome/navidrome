package conf

import (
	"github.com/deluan/gosonic/utils"
	"github.com/deluan/gosonic/repositories"
)

func init () {
	utils.DefineSingleton(new(repositories.ArtistIndex), repositories.NewArtistIndexRepository)
	utils.DefineSingleton(new(repositories.Property), repositories.NewPropertyRepository)
}
