package nativeapi

import (
	"context"
	"testing"

	_ "github.com/navidrome/navidrome/adapters/gotaglib"
	"github.com/navidrome/navidrome/conf/configtest"
	"github.com/navidrome/navidrome/core"
	"github.com/navidrome/navidrome/core/external"
	"github.com/navidrome/navidrome/core/playlists"
	_ "github.com/navidrome/navidrome/core/storage/local"
	"github.com/navidrome/navidrome/log"
	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/persistence/persistencetest"
	"github.com/navidrome/navidrome/server/events"
	"github.com/navidrome/navidrome/tests"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestNativeApi(t *testing.T) {
	tests.Init(t, false)
	log.SetLevel(log.LevelFatal)
	persistencetest.Init(t)
	RegisterFailHandler(Fail)
	RunSpecs(t, "Native RESTful API Suite")
}

var _ = configtest.RestoreAfterEachSpec()
var _ = persistencetest.ResetBeforeEachSpec()

type noopPluginUnloader struct{}

func (noopPluginUnloader) UnloadDisabledPlugins(context.Context) {}

// newTestRouter builds the native API router with the real library, user and playlist services on ds.
func newTestRouter(ds model.DataStore, pls playlists.Playlists, pluginManager PluginManager, provider external.Provider) *Router {
	if pls == nil {
		pls = playlists.NewPlaylists(ds, nil)
	}
	libs := core.NewLibrary(ds, nil, nil, events.NoopBroker(), noopPluginUnloader{})
	users := core.NewUser(ds, noopPluginUnloader{})
	return New(ds, nil, pls, nil, libs, users, nil, pluginManager, nil, provider, nil)
}

func putUsers(ctx context.Context, ds model.DataStore, users ...*model.User) {
	GinkgoHelper()
	for _, u := range users {
		Expect(ds.User().Put(ctx, u)).To(Succeed())
	}
}
