//go:build wireinject

package cmd

import (
	"context"
	"sync"

	"github.com/google/wire"
	"github.com/navidrome/navidrome/core"
	"github.com/navidrome/navidrome/core/agents/lastfm"
	"github.com/navidrome/navidrome/core/agents/listenbrainz"
	"github.com/navidrome/navidrome/core/artwork"
	"github.com/navidrome/navidrome/db"
	"github.com/navidrome/navidrome/persistence"
	"github.com/navidrome/navidrome/scanner"
	"github.com/navidrome/navidrome/scanner2"
	"github.com/navidrome/navidrome/server"
	"github.com/navidrome/navidrome/server/events"
	"github.com/navidrome/navidrome/server/nativeapi"
	"github.com/navidrome/navidrome/server/public"
	"github.com/navidrome/navidrome/server/subsonic"
)

var allProviders = wire.NewSet(
	core.Set,
	artwork.Set,
	subsonic.New,
	nativeapi.New,
	public.New,
	persistence.New,
	lastfm.NewRouter,
	listenbrainz.NewRouter,
	events.GetBroker,
	db.Db,
)

func CreateServer(ctx context.Context) *server.Server {
	panic(wire.Build(
		server.New,
		allProviders,
	))
}

func CreateNativeAPIRouter(ctx context.Context) *nativeapi.Router {
	panic(wire.Build(
		allProviders,
	))
}

func CreateSubsonicAPIRouter(ctx context.Context) *subsonic.Router {
	panic(wire.Build(
		allProviders,
		GetScanner,
	))
}

func CreatePublicRouter(ctx context.Context) *public.Router {
	panic(wire.Build(
		allProviders,
	))
}

func CreateLastFMRouter(ctx context.Context) *lastfm.Router {
	panic(wire.Build(
		allProviders,
	))
}

func CreateListenBrainzRouter(ctx context.Context) *listenbrainz.Router {
	panic(wire.Build(
		allProviders,
	))
}

// Scanner must be a Singleton
var (
	onceScanner     sync.Once
	scannerInstance scanner.Scanner
)

func GetScanner(ctx context.Context) scanner.Scanner {
	onceScanner.Do(func() {
		scannerInstance = createScanner(ctx)
	})
	return scannerInstance
}

func createScanner(ctx context.Context) scanner.Scanner {
	panic(wire.Build(
		allProviders,
		scanner2.New,
	))
}
