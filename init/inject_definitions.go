package init

import (
	"github.com/cloudsonic/sonic-server/domain"
	"github.com/cloudsonic/sonic-server/engine"
	"github.com/cloudsonic/sonic-server/itunesbridge"
	"github.com/cloudsonic/sonic-server/persistence"
	"github.com/cloudsonic/sonic-server/scanner"
	"github.com/cloudsonic/sonic-server/utils"
	"github.com/deluan/gomate"
	"github.com/deluan/gomate/ledis"
)

func init() {
	// Persistence
	utils.DefineSingleton(new(domain.ArtistIndexRepository), persistence.NewArtistIndexRepository)
	utils.DefineSingleton(new(domain.MediaFolderRepository), persistence.NewMediaFolderRepository)
	utils.DefineSingleton(new(domain.ArtistRepository), persistence.NewArtistRepository)
	utils.DefineSingleton(new(domain.AlbumRepository), persistence.NewAlbumRepository)
	utils.DefineSingleton(new(domain.MediaFileRepository), persistence.NewMediaFileRepository)
	utils.DefineSingleton(new(domain.PlaylistRepository), persistence.NewPlaylistRepository)

	// Engine (Use cases)
	utils.DefineSingleton(new(engine.PropertyRepository), persistence.NewPropertyRepository)
	utils.DefineSingleton(new(engine.NowPlayingRepository), persistence.NewNowPlayingRepository)
	utils.DefineSingleton(new(engine.Browser), engine.NewBrowser)
	utils.DefineSingleton(new(engine.ListGenerator), engine.NewListGenerator)
	utils.DefineSingleton(new(engine.Cover), engine.NewCover)
	utils.DefineSingleton(new(engine.Playlists), engine.NewPlaylists)
	utils.DefineSingleton(new(engine.Search), engine.NewSearch)
	utils.DefineSingleton(new(engine.Scrobbler), engine.NewScrobbler)
	utils.DefineSingleton(new(engine.Ratings), engine.NewRatings)

	utils.DefineSingleton(new(scanner.CheckSumRepository), persistence.NewCheckSumRepository)
	utils.DefineSingleton(new(scanner.Scanner), scanner.NewItunesScanner)

	// Other dependencies
	utils.DefineSingleton(new(itunesbridge.ItunesControl), itunesbridge.NewItunesControl)
	utils.DefineSingleton(new(gomate.DB), func() gomate.DB {
		return ledis.NewEmbeddedDB(persistence.Db())
	})
}
