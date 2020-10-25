//+build wireinject

package cmd

import (
	"github.com/deluan/navidrome/core"
	"github.com/deluan/navidrome/persistence"
	"github.com/deluan/navidrome/scanner"
	"github.com/deluan/navidrome/server"
	"github.com/deluan/navidrome/server/app"
	"github.com/deluan/navidrome/server/subsonic"
	"github.com/deluan/navidrome/server/subsonic/engine"
	"github.com/google/wire"
)

var allProviders = wire.NewSet(
	engine.Set,
	core.Set,
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

func CreateScanner(musicFolder string) scanner.Scanner {
	panic(wire.Build(
		allProviders,
	))
}

func CreateAppRouter() *app.Router {
	panic(wire.Build(allProviders))
}

func CreateSubsonicAPIRouter() *subsonic.Router {
	panic(wire.Build(allProviders))
}
