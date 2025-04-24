package agents

import (
	"context"

	"github.com/navidrome/navidrome/conf"
	"github.com/navidrome/navidrome/model"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

// MockPluginLoader implements PluginLoader for testing
type MockPluginLoader struct {
	pluginNames     []string
	loadedAgents    map[string]*MockAgent
	pluginCallCount map[string]int
}

func NewMockPluginLoader() *MockPluginLoader {
	return &MockPluginLoader{
		pluginNames:     []string{},
		loadedAgents:    make(map[string]*MockAgent),
		pluginCallCount: make(map[string]int),
	}
}

func (m *MockPluginLoader) PluginNames(serviceName string) []string {
	return m.pluginNames
}

func (m *MockPluginLoader) LoadMediaAgent(name string) (Interface, bool) {
	m.pluginCallCount[name]++
	agent, exists := m.loadedAgents[name]
	return agent, exists
}

// MockAgent is a mock agent implementation for testing
type MockAgent struct {
	name string
	mbid string
}

func (m *MockAgent) AgentName() string {
	return m.name
}

func (m *MockAgent) GetArtistMBID(ctx context.Context, id string, name string) (string, error) {
	return m.mbid, nil
}

var _ Interface = (*MockAgent)(nil)
var _ ArtistMBIDRetriever = (*MockAgent)(nil)

var _ PluginLoader = (*MockPluginLoader)(nil)

var _ = Describe("Agents with Plugin Loading", func() {
	var mockLoader *MockPluginLoader
	var agents *Agents

	BeforeEach(func() {
		mockLoader = NewMockPluginLoader()

		// Create the agents instance with our mock loader
		agents = createAgents(nil, mockLoader)
	})

	Context("Dynamic agent discovery", func() {
		It("should include ONLY local agent when no config is specified", func() {
			// Ensure no specific agents are configured
			conf.Server.Agents = ""

			// Add some plugin agents that should be ignored
			mockLoader.pluginNames = append(mockLoader.pluginNames, "plugin_agent", "another_plugin")

			// Should only include the local agent
			agentNames := agents.getEnabledAgentNames()
			Expect(agentNames).To(HaveExactElements(LocalAgentName))
		})

		It("should NOT include plugin agents when no config is specified", func() {
			// Ensure no specific agents are configured
			conf.Server.Agents = ""

			// Add a plugin agent
			mockLoader.pluginNames = append(mockLoader.pluginNames, "plugin_agent")

			// Should only include the local agent
			agentNames := agents.getEnabledAgentNames()
			Expect(agentNames).To(HaveExactElements(LocalAgentName))
			Expect(agentNames).NotTo(ContainElement("plugin_agent"))
		})

		It("should include plugin agents in the enabled agents list ONLY when explicitly configured", func() {
			// Add a plugin agent
			mockLoader.pluginNames = append(mockLoader.pluginNames, "plugin_agent")

			// With no config, should not include plugin
			conf.Server.Agents = ""
			agentNames := agents.getEnabledAgentNames()
			Expect(agentNames).To(HaveExactElements(LocalAgentName))
			Expect(agentNames).NotTo(ContainElement("plugin_agent"))

			// When explicitly configured, should include plugin
			conf.Server.Agents = "plugin_agent"
			agentNames = agents.getEnabledAgentNames()
			Expect(agentNames).To(ContainElements(LocalAgentName, "plugin_agent"))
		})

		It("should only include configured plugin agents when config is specified", func() {
			// Add two plugin agents
			mockLoader.pluginNames = append(mockLoader.pluginNames, "plugin_one", "plugin_two")

			// Configure only one of them
			conf.Server.Agents = "plugin_one"

			// Verify only the configured one is included
			agentNames := agents.getEnabledAgentNames()
			Expect(agentNames).To(ContainElement("plugin_one"))
			Expect(agentNames).NotTo(ContainElement("plugin_two"))
		})

		It("should load plugin agents on demand", func() {
			ctx := context.Background()

			// Configure to use our plugin
			conf.Server.Agents = "plugin_agent"

			// Add a plugin agent
			mockLoader.pluginNames = append(mockLoader.pluginNames, "plugin_agent")
			mockLoader.loadedAgents["plugin_agent"] = &MockAgent{
				name: "plugin_agent",
				mbid: "plugin-mbid",
			}

			// Try to get data from it
			mbid, err := agents.GetArtistMBID(ctx, "123", "Artist")

			Expect(err).ToNot(HaveOccurred())
			Expect(mbid).To(Equal("plugin-mbid"))
			Expect(mockLoader.pluginCallCount["plugin_agent"]).To(Equal(1))
		})

		It("should cache plugin agents", func() {
			ctx := context.Background()

			// Configure to use our plugin
			conf.Server.Agents = "plugin_agent"

			// Add a plugin agent
			mockLoader.pluginNames = append(mockLoader.pluginNames, "plugin_agent")
			mockLoader.loadedAgents["plugin_agent"] = &MockAgent{
				name: "plugin_agent",
				mbid: "plugin-mbid",
			}

			// Call multiple times
			_, err := agents.GetArtistMBID(ctx, "123", "Artist")
			Expect(err).ToNot(HaveOccurred())
			_, err = agents.GetArtistMBID(ctx, "123", "Artist")
			Expect(err).ToNot(HaveOccurred())
			_, err = agents.GetArtistMBID(ctx, "123", "Artist")
			Expect(err).ToNot(HaveOccurred())

			// Should only load once
			Expect(mockLoader.pluginCallCount["plugin_agent"]).To(Equal(1))
		})

		It("should try both built-in and plugin agents", func() {
			// Create a mock built-in agent
			Register("built_in", func(ds model.DataStore) Interface {
				return &MockAgent{
					name: "built_in",
					mbid: "built-in-mbid",
				}
			})
			defer func() {
				delete(Map, "built_in")
			}()

			// Configure to use both built-in and plugin
			conf.Server.Agents = "built_in,plugin_agent"

			// Add a plugin agent
			mockLoader.pluginNames = append(mockLoader.pluginNames, "plugin_agent")
			mockLoader.loadedAgents["plugin_agent"] = &MockAgent{
				name: "plugin_agent",
				mbid: "plugin-mbid",
			}

			// Verify that both are in the enabled list
			agentNames := agents.getEnabledAgentNames()
			Expect(agentNames).To(ContainElements("built_in", "plugin_agent"))
		})

		It("should respect the order specified in configuration", func() {
			// Create mock built-in agents
			Register("agent_a", func(ds model.DataStore) Interface {
				return &MockAgent{name: "agent_a"}
			})
			Register("agent_b", func(ds model.DataStore) Interface {
				return &MockAgent{name: "agent_b"}
			})
			defer func() {
				delete(Map, "agent_a")
				delete(Map, "agent_b")
			}()

			// Add plugin agents
			mockLoader.pluginNames = append(mockLoader.pluginNames, "plugin_x", "plugin_y")

			// Configure specific order - plugin first, then built-ins
			conf.Server.Agents = "plugin_y,agent_b,plugin_x,agent_a"

			// Get the agent names
			agentNames := agents.getEnabledAgentNames()

			// Verify the order matches configuration, with LocalAgentName at the end
			Expect(agentNames).To(HaveExactElements("plugin_y", "agent_b", "plugin_x", "agent_a", LocalAgentName))
		})
	})
})
