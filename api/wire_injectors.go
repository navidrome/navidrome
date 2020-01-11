//+build wireinject

package api

import (
	"github.com/cloudsonic/sonic-server/engine"
	"github.com/cloudsonic/sonic-server/itunesbridge"
	"github.com/cloudsonic/sonic-server/persistence/db_ledis"
	"github.com/cloudsonic/sonic-server/persistence/db_storm"
	"github.com/deluan/gomate"
	"github.com/deluan/gomate/ledis"
	"github.com/google/wire"
)

var allProviders = wire.NewSet(
	itunesbridge.NewItunesControl,
	//db_ledis.Set,
	db_storm.Set,
	engine.Set,
	NewSystemController,
	NewBrowsingController,
	NewAlbumListController,
	NewMediaAnnotationController,
	NewPlaylistsController,
	NewSearchingController,
	NewUsersController,
	NewMediaRetrievalController,
	NewStreamController,
	newDB,
)

func initSystemController() *SystemController {
	panic(wire.Build(allProviders))
}

func initBrowsingController() *BrowsingController {
	panic(wire.Build(allProviders))
}

func initAlbumListController() *AlbumListController {
	panic(wire.Build(allProviders))
}

func initMediaAnnotationController() *MediaAnnotationController {
	panic(wire.Build(allProviders))
}

func initPlaylistsController() *PlaylistsController {
	panic(wire.Build(allProviders))
}

func initSearchingController() *SearchingController {
	panic(wire.Build(allProviders))
}

func initUsersController() *UsersController {
	panic(wire.Build(allProviders))
}

func initMediaRetrievalController() *MediaRetrievalController {
	panic(wire.Build(allProviders))
}

func initStreamController() *StreamController {
	panic(wire.Build(allProviders))
}

func newDB() gomate.DB {
	return ledis.NewEmbeddedDB(db_ledis.Db())
}
