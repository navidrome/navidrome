//+build wireinject

package api

import (
	"github.com/cloudsonic/sonic-server/itunesbridge"
	"github.com/cloudsonic/sonic-server/persistence/db_ledis"
	"github.com/deluan/gomate"
	"github.com/deluan/gomate/ledis"
	"github.com/google/wire"
)

var allProviders = wire.NewSet(
	itunesbridge.NewItunesControl,
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
	wire.FieldsOf(new(*Router), "Browser", "Cover", "ListGenerator", "Playlists", "Ratings", "Scrobbler", "Search"),
)

func initSystemController(router *Router) *SystemController {
	panic(wire.Build(allProviders))
}

func initBrowsingController(router *Router) *BrowsingController {
	panic(wire.Build(allProviders))
}

func initAlbumListController(router *Router) *AlbumListController {
	panic(wire.Build(allProviders))
}

func initMediaAnnotationController(router *Router) *MediaAnnotationController {
	panic(wire.Build(allProviders))
}

func initPlaylistsController(router *Router) *PlaylistsController {
	panic(wire.Build(allProviders))
}

func initSearchingController(router *Router) *SearchingController {
	panic(wire.Build(allProviders))
}

func initUsersController(router *Router) *UsersController {
	panic(wire.Build(allProviders))
}

func initMediaRetrievalController(router *Router) *MediaRetrievalController {
	panic(wire.Build(allProviders))
}

func initStreamController(router *Router) *StreamController {
	panic(wire.Build(allProviders))
}

func newDB() gomate.DB {
	return ledis.NewEmbeddedDB(db_ledis.Db())
}
