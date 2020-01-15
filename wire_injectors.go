//+build wireinject

package main

import (
	"github.com/cloudsonic/sonic-server/api"
	"github.com/cloudsonic/sonic-server/engine"
	"github.com/cloudsonic/sonic-server/itunesbridge"
	"github.com/cloudsonic/sonic-server/model"
	"github.com/cloudsonic/sonic-server/persistence"
	"github.com/cloudsonic/sonic-server/scanner_legacy"
	"github.com/cloudsonic/sonic-server/server"
	"github.com/google/wire"
)

type Repositories struct {
	AlbumRepository       model.AlbumRepository
	ArtistRepository      model.ArtistRepository
	CheckSumRepository    model.ChecksumRepository
	ArtistIndexRepository model.ArtistIndexRepository
	MediaFileRepository   model.MediaFileRepository
	MediaFolderRepository model.MediaFolderRepository
	NowPlayingRepository  model.NowPlayingRepository
	PlaylistRepository    model.PlaylistRepository
	PropertyRepository    model.PropertyRepository
}

var allProviders = wire.NewSet(
	itunesbridge.NewItunesControl,
	engine.Set,
	scanner_legacy.Set,
	api.NewRouter,
	wire.FieldsOf(new(*Repositories), "AlbumRepository", "ArtistRepository", "CheckSumRepository",
		"ArtistIndexRepository", "MediaFileRepository", "MediaFolderRepository", "NowPlayingRepository",
		"PlaylistRepository", "PropertyRepository"),
	createPersistenceProvider,
)

func CreateApp(musicFolder string) *server.Server {
	panic(wire.Build(
		server.New,
		allProviders,
	))
}

func CreateSubsonicAPIRouter() *api.Router {
	panic(wire.Build(allProviders))
}

// When implementing a different persistence layer, duplicate this function (in separated files) and use build tags
// to conditionally select which function to use
func createPersistenceProvider() *Repositories {
	panic(wire.Build(
		persistence.Set,
		wire.Struct(new(Repositories), "*"),
	))
}
