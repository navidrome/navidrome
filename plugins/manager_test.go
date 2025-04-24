package plugins

import (
	"path/filepath"

	"github.com/navidrome/navidrome/conf"
	"github.com/navidrome/navidrome/conf/configtest"
	"github.com/navidrome/navidrome/core/agents"
	"github.com/navidrome/navidrome/core/scrobbler"
	"github.com/navidrome/navidrome/tests"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Plugin Manager", func() {
	var mgr *Manager
	BeforeEach(func() {
		DeferCleanup(configtest.SetupConfig())
		conf.Server.Plugins.Enabled = true
		absPath, err := filepath.Abs("./testdata")
		Expect(err).To(BeNil())
		conf.Server.Plugins.Folder = absPath
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
		// The plugin directory is 'fake_scrobbler', so the scrobbler name should be 'fake_scrobbler'
		Expect(scrobbler.IsScrobblerRegistered("fake_scrobbler")).To(BeTrue(), "plugin scrobbler should be registered")
	})
})
