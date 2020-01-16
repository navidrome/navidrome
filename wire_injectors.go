//+build wireinject

package main

import (
	"github.com/cloudsonic/sonic-server/api"
	"github.com/cloudsonic/sonic-server/engine"
	"github.com/cloudsonic/sonic-server/itunesbridge"
	"github.com/cloudsonic/sonic-server/persistence"
	"github.com/cloudsonic/sonic-server/scanner"
	"github.com/cloudsonic/sonic-server/scanner_legacy"
	"github.com/cloudsonic/sonic-server/server"
	"github.com/google/wire"
)

var allProviders = wire.NewSet(
	itunesbridge.NewItunesControl,
	engine.Set,
	scanner_legacy.Set,
	scanner.New,
	api.NewRouter,
	persistence.Set,
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
