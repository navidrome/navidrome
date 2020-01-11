//+build wireinject

package main

import (
	"github.com/cloudsonic/sonic-server/api"
	"github.com/cloudsonic/sonic-server/domain"
	"github.com/cloudsonic/sonic-server/engine"
	"github.com/cloudsonic/sonic-server/itunesbridge"
	"github.com/cloudsonic/sonic-server/persistence"
	"github.com/cloudsonic/sonic-server/persistence/db_ledis"
	"github.com/cloudsonic/sonic-server/persistence/db_storm"
	"github.com/cloudsonic/sonic-server/scanner"
	"github.com/deluan/gomate"
	"github.com/deluan/gomate/ledis"
	"github.com/google/wire"
)

type Provider struct {
	AlbumRepository       domain.AlbumRepository
	ArtistRepository      domain.ArtistRepository
	CheckSumRepository    scanner.CheckSumRepository
	ArtistIndexRepository domain.ArtistIndexRepository
	MediaFileRepository   domain.MediaFileRepository
	MediaFolderRepository domain.MediaFolderRepository
	NowPlayingRepository  domain.NowPlayingRepository
	PlaylistRepository    domain.PlaylistRepository
	PropertyRepository    domain.PropertyRepository
}

var allProviders = wire.NewSet(
	itunesbridge.NewItunesControl,
	engine.Set,
	scanner.Set,
	newDB,
	api.NewRouter,
	wire.FieldsOf(new(*Provider), "AlbumRepository", "ArtistRepository", "CheckSumRepository",
		"ArtistIndexRepository", "MediaFileRepository", "MediaFolderRepository", "NowPlayingRepository",
		"PlaylistRepository", "PropertyRepository"),
	createPersistenceProvider,
)

func CreateApp(musicFolder string, p persistence.ProviderIdentifier) *App {
	panic(wire.Build(
		NewApp,
		allProviders,
	))
}

func CreateSubsonicAPIRouter(p persistence.ProviderIdentifier) *api.Router {
	panic(wire.Build(allProviders))
}

func createPersistenceProvider(provider persistence.ProviderIdentifier) *Provider {
	switch provider {
	case "storm":
		return createStormProvider()
	default:
		return createLedisDBProvider()
	}
}

func createLedisDBProvider() *Provider {
	panic(wire.Build(
		db_ledis.Set,
		wire.Struct(new(Provider), "*"),
	))
}

func createStormProvider() *Provider {
	panic(wire.Build(
		db_storm.Set,
		wire.Struct(new(Provider), "*"),
	))
}

func newDB() gomate.DB {
	return ledis.NewEmbeddedDB(db_ledis.Db())
}
