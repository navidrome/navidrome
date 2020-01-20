//+build wireinject

package main

import (
	"github.com/cloudsonic/sonic-server/engine"
	"github.com/cloudsonic/sonic-server/persistence"
	"github.com/cloudsonic/sonic-server/scanner"
	"github.com/cloudsonic/sonic-server/server"
	"github.com/cloudsonic/sonic-server/server/app"
	"github.com/cloudsonic/sonic-server/server/subsonic"
	"github.com/google/wire"
)

var allProviders = wire.NewSet(
	engine.Set,
	scanner.New,
	subsonic.New,
	app.New,
	persistence.New,
)

func CreateServer(musicFolder string) *server.Server {
	panic(wire.Build(
		server.New,
		allProviders,
	))
}

func CreateAppRouter(path string) *app.Router {
	panic(wire.Build(allProviders))
}

func CreateSubsonicAPIRouter() *subsonic.Router {
	panic(wire.Build(allProviders))
}
