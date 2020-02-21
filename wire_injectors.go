//+build wireinject

package main

import (
	"github.com/deluan/navidrome/engine"
	"github.com/deluan/navidrome/persistence"
	"github.com/deluan/navidrome/scanner"
	"github.com/deluan/navidrome/server"
	"github.com/deluan/navidrome/server/app"
	"github.com/deluan/navidrome/server/subsonic"
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

func CreateSubsonicAPIRouter() (*subsonic.Router, error) {
	panic(wire.Build(allProviders))
}
