package plugins

import (
	"context"
	"fmt"
	"sync"

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

	Describe("Plugin pre-compilation and agent factory synchronization", func() {
		var fakeAgentCtor func(pool *sync.Pool, wasmPath, pluginName string) agents.Interface

		BeforeEach(func() {
			// Always return a mock agent for these tests
			fakeAgentCtor = func(pool *sync.Pool, wasmPath, pluginName string) agents.Interface {
				return &mockAgent{}
			}
		})

		It("should create agent after successful pre-compilation", func() {
			// Simulate a plugin that has already finished compiling successfully
			pluginState := &pluginState{ready: make(chan struct{})}
			close(pluginState.ready)

			// Create a mock plugin pool and agent factory
			mockPool := newPluginPool(struct{}{}, "fake.wasm", "fake", func(ctx context.Context, loader struct{}, wasmPath string) (any, error) {
				return &mockAgent{}, nil
			})
			agentFactory := createAgentFactory(pluginState, mockPool, "fake.wasm", "fake", fakeAgentCtor)
			mockDS := &tests.MockDataStore{}

			// Agent should be created immediately
			agent := agentFactory(mockDS)
			Expect(agent).NotTo(BeNil(), "Agent should not be nil after successful compilation")
			Expect(agent.AgentName()).To(Equal("mockAgent"), "Agent should have the correct name")
		})

		It("should return nil agent if pre-compilation fails", func() {
			// Simulate a plugin that failed to compile
			pluginState := &pluginState{ready: make(chan struct{}), err: fmt.Errorf("compilation failed")}
			close(pluginState.ready)

			// Create a mock plugin pool and agent factory
			mockPool := newPluginPool(struct{}{}, "fake.wasm", "fake", func(ctx context.Context, loader struct{}, wasmPath string) (any, error) {
				return &mockAgent{}, nil
			})
			agentFactory := createAgentFactory(pluginState, mockPool, "fake.wasm", "fake", fakeAgentCtor)
			mockDS := &tests.MockDataStore{}

			// Agent factory should return nil due to compilation error
			agent := agentFactory(mockDS)
			Expect(agent).To(BeNil(), "Agent should be nil if compilation failed")
		})

		It("should block agent creation until pre-compilation completes", func() {
			// Simulate a plugin that is still compiling (ready channel not closed)
			pluginState := &pluginState{ready: make(chan struct{})}
			mockPool := newPluginPool(struct{}{}, "fake.wasm", "fake", func(ctx context.Context, loader struct{}, wasmPath string) (any, error) {
				return &mockAgent{}, nil
			})
			agentFactory := createAgentFactory(pluginState, mockPool, "fake.wasm", "fake", fakeAgentCtor)
			mockDS := &tests.MockDataStore{}

			// Start agent creation in a goroutine; it should block until compilation completes
			resultChan := make(chan agents.Interface)
			go func() {
				resultChan <- agentFactory(mockDS)
			}()

			// Assert that the factory is blocked (no result yet)
			Consistently(resultChan, "100ms").ShouldNot(Receive(), "Factory should block while compilation is pending")

			// Simulate compilation completion
			close(pluginState.ready)

			// Now the factory should unblock and return a valid agent
			var agent agents.Interface
			Eventually(resultChan, "100ms").Should(Receive(&agent), "Factory should return agent after compilation completes")
			Expect(agent).NotTo(BeNil(), "Agent should not be nil")
			Expect(agent.AgentName()).To(Equal("mockAgent"), "Agent should have the correct name")
		})

		It("should allow multiple plugins to precompile in parallel", func() {
			// Simulate two plugins, both still compiling
			pluginStates := []*pluginState{
				{ready: make(chan struct{})},
				{ready: make(chan struct{})},
			}
			mockPool := newPluginPool(struct{}{}, "fake.wasm", "fake", func(ctx context.Context, loader struct{}, wasmPath string) (any, error) {
				return &mockAgent{}, nil
			})
			factory1 := createAgentFactory(pluginStates[0], mockPool, "fake1.wasm", "fake1", fakeAgentCtor)
			factory2 := createAgentFactory(pluginStates[1], mockPool, "fake2.wasm", "fake2", fakeAgentCtor)
			mockDS := &tests.MockDataStore{}

			// Start agent creation for both plugins in parallel; both should block
			ch1 := make(chan agents.Interface)
			ch2 := make(chan agents.Interface)
			go func() { ch1 <- factory1(mockDS) }()
			go func() { ch2 <- factory2(mockDS) }()

			// Assert that both factories are blocked
			Consistently(ch1, "100ms").ShouldNot(Receive(), "First factory should block while compilation is pending")
			Consistently(ch2, "100ms").ShouldNot(Receive(), "Second factory should block while compilation is pending")

			// Simulate both compilations completing
			close(pluginStates[0].ready)
			close(pluginStates[1].ready)

			// Both factories should now return valid agents
			var agent1, agent2 agents.Interface
			Eventually(ch1, "100ms").Should(Receive(&agent1), "First factory should return agent after compilation completes")
			Eventually(ch2, "100ms").Should(Receive(&agent2), "Second factory should return agent after compilation completes")
			Expect(agent1).NotTo(BeNil(), "First agent should not be nil")
			Expect(agent2).NotTo(BeNil(), "Second agent should not be nil")
			Expect(agent1.AgentName()).To(Equal("mockAgent"), "First agent should have the correct name")
			Expect(agent2.AgentName()).To(Equal("mockAgent"), "Second agent should have the correct name")
		})
	})
})

// mockAgent implements agents.Interface for testing
type mockAgent struct{}

func (m *mockAgent) AgentName() string { return "mockAgent" }
