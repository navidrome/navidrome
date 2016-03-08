package conf

import (
	"github.com/deluan/gosonic/domain"
	"github.com/deluan/gosonic/engine"
	"github.com/deluan/gosonic/persistence"
	"github.com/deluan/gosonic/utils"
)

func init() {
	// Persistence
	utils.DefineSingleton(new(domain.ArtistIndexRepository), persistence.NewArtistIndexRepository)
	utils.DefineSingleton(new(domain.MediaFolderRepository), persistence.NewMediaFolderRepository)
	utils.DefineSingleton(new(domain.ArtistRepository), persistence.NewArtistRepository)
	utils.DefineSingleton(new(domain.AlbumRepository), persistence.NewAlbumRepository)
	utils.DefineSingleton(new(domain.MediaFileRepository), persistence.NewMediaFileRepository)

	// Engine (Use cases)
	utils.DefineSingleton(new(engine.PropertyRepository), persistence.NewPropertyRepository)
	utils.DefineSingleton(new(engine.Browser), engine.NewBrowser)
}
