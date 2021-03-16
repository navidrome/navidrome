//+build wireinject

package cmd

import (
	"sync"

	"github.com/google/wire"

	"github.com/navidrome/navidrome/core"
	"github.com/navidrome/navidrome/persistence"
	"github.com/navidrome/navidrome/scanner"
	"github.com/navidrome/navidrome/server"
	"github.com/navidrome/navidrome/server/app"
	"github.com/navidrome/navidrome/server/events"
	"github.com/navidrome/navidrome/server/subsonic"
)

var allProviders = wire.NewSet(
	core.Set,
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

func CreateAppRouter() *app.Router {
	panic(wire.Build(
		allProviders,
		GetBroker,
	))
}

func CreateSubsonicAPIRouter() *subsonic.Router {
	panic(wire.Build(
		allProviders,
		GetScanner,
	))
}

// Scanner must be a Singleton
var (
	onceScanner     sync.Once
	scannerInstance scanner.Scanner
)

func GetScanner() scanner.Scanner {
	onceScanner.Do(func() {
		scannerInstance = createScanner()
	})
	return scannerInstance
}

func createScanner() scanner.Scanner {
	panic(wire.Build(
		allProviders,
		GetBroker,
		scanner.New,
	))
}

// Broker must be a Singleton
var (
	onceBroker     sync.Once
	brokerInstance events.Broker
)

func GetBroker() events.Broker {
	onceBroker.Do(func() {
		brokerInstance = createBroker()
	})
	return brokerInstance
}

func createBroker() events.Broker {
	panic(wire.Build(
		events.NewBroker,
	))
}
