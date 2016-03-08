package conf

import (
	"github.com/deluan/gosonic/domain"
	"github.com/deluan/gosonic/engine"
	"github.com/deluan/gosonic/persistence"
	"github.com/deluan/gosonic/utils"
)

func init() {
	// Persistence
	ir := utils.DefineSingleton(new(domain.ArtistIndexRepository), persistence.NewArtistIndexRepository)
	pr := utils.DefineSingleton(new(domain.PropertyRepository), persistence.NewPropertyRepository)
	fr := utils.DefineSingleton(new(domain.MediaFolderRepository), persistence.NewMediaFolderRepository)
	ar := utils.DefineSingleton(new(domain.ArtistRepository), persistence.NewArtistRepository)
	alr := utils.DefineSingleton(new(domain.AlbumRepository), persistence.NewAlbumRepository)
	mr := utils.DefineSingleton(new(domain.MediaFileRepository), persistence.NewMediaFileRepository)

	// Engine (Use cases)
	utils.DefineSingleton(new(engine.Browser), engine.NewBrowser, pr, fr, ir, ar, alr, mr)
}
