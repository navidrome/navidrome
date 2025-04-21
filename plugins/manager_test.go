package plugins

import (
	"github.com/navidrome/navidrome/conf"
	"github.com/navidrome/navidrome/conf/configtest"
	"github.com/navidrome/navidrome/core/agents"
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
		mgr = GetManager()
	})

	It("should auto-register and load a plugin from the testdata folder", func() {
		Expect(mgr).NotTo(BeNil())

		// The plugin directory is 'agent', so the agent name should be 'agent'
		constructor, ok := agents.Map["agent"]
		Expect(ok).To(BeTrue(), "plugin agent should be registered")

		ds := &tests.MockDataStore{} // Use a mock DataStore
		agent := constructor(ds)
		Expect(agent).NotTo(BeNil(), "plugin agent should be constructible")
		Expect(agent.AgentName()).To(Equal("agent"))
	})
})
