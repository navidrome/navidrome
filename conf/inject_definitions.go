package conf

import (
	"github.com/deluan/gosonic/utils"
	"github.com/deluan/gosonic/persistence"
	"github.com/deluan/gosonic/domain"
)

func init () {
	utils.DefineSingleton(new(domain.ArtistIndexRepository), persistence.NewArtistIndexRepository)
	utils.DefineSingleton(new(domain.PropertyRepository), persistence.NewPropertyRepository)
	utils.DefineSingleton(new(domain.MediaFolderRepository), persistence.NewMediaFolderRepository)
}
