package plugins

import (
	"context"
	"time"

	"github.com/navidrome/navidrome/conf"
	"github.com/navidrome/navidrome/conf/configtest"
	"github.com/navidrome/navidrome/core/agents"
	"github.com/navidrome/navidrome/core/scrobbler"
	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/tests"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Plugin Manager", func() {
	var mgr *Manager
	BeforeEach(func() {
		DeferCleanup(configtest.SetupConfig())
		conf.Server.Plugins.Enabled = true
		conf.Server.Plugins.Folder = "plugins/testdata"
		mgr = createManager()
	})

	It("should auto-register and load a plugin from the testdata folder", func() {
		Expect(mgr).NotTo(BeNil())

		// The plugin directory is 'fake_artist_agent', so the agent name should be 'fake_artist_agent'
		constructor, ok := agents.Map["fake_artist_agent"]
		Expect(ok).To(BeTrue(), "plugin agent should be registered")

		ds := &tests.MockDataStore{} // Use a mock DataStore
		agent := constructor(ds)
		Expect(agent).NotTo(BeNil(), "plugin agent should be constructible")
		Expect(agent.AgentName()).To(Equal("fake_artist_agent"))
	})

	It("should auto-register and load a scrobbler plugin from the testdata folder", func() {
		// The plugin directory is 'fake_scrobbler_agent', so the agent name should be 'fake_scrobbler_agent'
		constructor, ok := agents.Map["fake_scrobbler_agent"]
		Expect(ok).To(BeTrue(), "plugin scrobbler should be registered")

		ds := &tests.MockDataStore{} // Use a mock DataStore
		sc := constructor(ds)
		Expect(sc).NotTo(BeNil(), "plugin scrobbler should be constructible")
		Expect(sc.AgentName()).To(Equal("fake_scrobbler_agent"))

		// Type assert to scrobbler.Scrobbler
		scrob, ok := sc.(interface {
			IsAuthorized(ctx context.Context, userId string) bool
			NowPlaying(ctx context.Context, userId string, track *model.MediaFile) error
			Scrobble(ctx context.Context, userId string, s scrobbler.Scrobble) error
		})
		Expect(ok).To(BeTrue(), "plugin scrobbler should implement Scrobbler interface")
		ctx := context.Background()
		// IsAuthorized should return true
		Expect(scrob.IsAuthorized(ctx, "user1")).To(BeTrue())
		// NowPlaying and Scrobble should not error
		track := &model.MediaFile{ID: "t1", Title: "Song", Album: "Album", Duration: 123}
		Expect(scrob.NowPlaying(ctx, "user1", track)).To(Succeed())
		Expect(scrob.Scrobble(ctx, "user1", scrobbler.Scrobble{MediaFile: *track, TimeStamp: time.Unix(123456, 0)})).To(Succeed())
	})
})
