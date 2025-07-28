//go:build wireinject

package cmd

import (
	"context"

	"github.com/google/wire"
	"github.com/navidrome/navidrome/core"
	"github.com/navidrome/navidrome/core/agents"
	"github.com/navidrome/navidrome/core/agents/lastfm"
	"github.com/navidrome/navidrome/core/agents/listenbrainz"
	"github.com/navidrome/navidrome/core/artwork"
	"github.com/navidrome/navidrome/core/metrics"
	"github.com/navidrome/navidrome/core/playback"
	"github.com/navidrome/navidrome/core/scrobbler"
	"github.com/navidrome/navidrome/db"
	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/persistence"
	"github.com/navidrome/navidrome/plugins"
	"github.com/navidrome/navidrome/scanner"
	"github.com/navidrome/navidrome/server"
	"github.com/navidrome/navidrome/server/events"
	"github.com/navidrome/navidrome/server/nativeapi"
	"github.com/navidrome/navidrome/server/public"
	"github.com/navidrome/navidrome/server/subsonic"
)

var allProviders = wire.NewSet(
	core.Set,
	artwork.Set,
	server.New,
	subsonic.New,
	nativeapi.New,
	public.New,
	persistence.New,
	lastfm.NewRouter,
	listenbrainz.NewRouter,
	events.GetBroker,
	scanner.New,
	scanner.GetWatcher,
	plugins.GetManager,
	metrics.GetPrometheusInstance,
	db.Db,
	wire.Bind(new(agents.PluginLoader), new(plugins.Manager)),
	wire.Bind(new(scrobbler.PluginLoader), new(plugins.Manager)),
	wire.Bind(new(metrics.PluginLoader), new(plugins.Manager)),
	wire.Bind(new(core.Scanner), new(scanner.Scanner)),
	wire.Bind(new(core.Watcher), new(scanner.Watcher)),
)

func CreateDataStore() model.DataStore {
	panic(wire.Build(
		allProviders,
	))
}

func CreateServer() *server.Server {
	panic(wire.Build(
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
	))
}

func CreatePublicRouter() *public.Router {
	panic(wire.Build(
		allProviders,
	))
}

func CreateLastFMRouter() *lastfm.Router {
	panic(wire.Build(
		allProviders,
	))
}

func CreateListenBrainzRouter() *listenbrainz.Router {
	panic(wire.Build(
		allProviders,
	))
}

func CreateInsights() metrics.Insights {
	panic(wire.Build(
		allProviders,
	))
}

func CreatePrometheus() metrics.Metrics {
	panic(wire.Build(
		allProviders,
	))
}

func CreateScanner(ctx context.Context) scanner.Scanner {
	panic(wire.Build(
		allProviders,
	))
}

func CreateScanWatcher(ctx context.Context) scanner.Watcher {
	panic(wire.Build(
		allProviders,
	))
}

func GetPlaybackServer() playback.PlaybackServer {
	panic(wire.Build(
		allProviders,
	))
}

func getPluginManager() plugins.Manager {
	panic(wire.Build(
		allProviders,
	))
}

func GetPluginManager(ctx context.Context) plugins.Manager {
	manager := getPluginManager()
	manager.SetSubsonicRouter(CreateSubsonicAPIRouter(ctx))
	return manager
}
