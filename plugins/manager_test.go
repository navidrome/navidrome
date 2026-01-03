package plugins

import (
	"context"
	"fmt"
	"net/http"
	"sync"

	"github.com/navidrome/navidrome/core/agents"
	"github.com/navidrome/navidrome/server/events"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Manager", Ordered, func() {
	var ctx context.Context

	BeforeAll(func() {
		ctx = GinkgoT().Context()
	})

	Describe("Plugin Loading", func() {
		It("loads enabled plugins from DB on Start", func() {
			// Plugin is already loaded by testManager.Start() via loadEnabledPlugins
			names := testManager.PluginNames(string(CapabilityMetadataAgent))
			Expect(names).To(ContainElement("test-metadata-agent"))
		})
	})

	Describe("unloadPlugin", func() {
		It("removes a loaded plugin", func() {
			// Plugin is already loaded from Start
			err := testManager.unloadPlugin("test-metadata-agent")
			Expect(err).ToNot(HaveOccurred())

			names := testManager.PluginNames(string(CapabilityMetadataAgent))
			Expect(names).ToNot(ContainElement("test-metadata-agent"))
		})

		It("returns error when plugin not found", func() {
			err := testManager.unloadPlugin("nonexistent")
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("not found"))
		})
	})

	Describe("EnablePlugin", func() {
		It("enables and loads a disabled plugin", func() {
			// First disable the plugin (which also unloads it)
			err := testManager.DisablePlugin(ctx, "test-metadata-agent")
			Expect(err).ToNot(HaveOccurred())
			Expect(testManager.PluginNames(string(CapabilityMetadataAgent))).ToNot(ContainElement("test-metadata-agent"))

			// Enable it
			err = testManager.EnablePlugin(ctx, "test-metadata-agent")
			Expect(err).ToNot(HaveOccurred())

			names := testManager.PluginNames(string(CapabilityMetadataAgent))
			Expect(names).To(ContainElement("test-metadata-agent"))
		})
	})

	Describe("DisablePlugin", func() {
		It("disables and unloads an enabled plugin", func() {
			// Ensure the plugin is loaded first
			_ = testManager.EnablePlugin(ctx, "test-metadata-agent")

			err := testManager.DisablePlugin(ctx, "test-metadata-agent")
			Expect(err).ToNot(HaveOccurred())

			names := testManager.PluginNames(string(CapabilityMetadataAgent))
			Expect(names).ToNot(ContainElement("test-metadata-agent"))
		})
	})

	Describe("GetPluginInfo", func() {
		BeforeEach(func() {
			// Ensure plugin is loaded for this test
			_ = testManager.EnablePlugin(ctx, "test-metadata-agent")
		})

		It("returns information about all loaded plugins", func() {
			info := testManager.GetPluginInfo()
			Expect(info).To(HaveKey("test-metadata-agent"))
			Expect(info["test-metadata-agent"].Name).To(Equal("Test Plugin"))
			Expect(info["test-metadata-agent"].Version).To(Equal("1.0.0"))
		})
	})

	It("can call the plugin concurrently", func() {
		// Ensure plugin is loaded
		_ = testManager.EnablePlugin(ctx, "test-metadata-agent")

		const concurrency = 30
		errs := make(chan error, concurrency)
		bios := make(chan string, concurrency)

		g := sync.WaitGroup{}
		g.Add(concurrency)
		for i := range concurrency {
			go func(i int) {
				defer g.Done()
				a, ok := testManager.LoadMediaAgent("test-metadata-agent")
				Expect(ok).To(BeTrue())
				agent := a.(agents.ArtistBiographyRetriever)
				bio, err := agent.GetArtistBiography(ctx, fmt.Sprintf("artist-%d", i), fmt.Sprintf("Artist %d", i), "")
				if err != nil {
					errs <- err
					return
				}
				bios <- bio
			}(i)
		}
		g.Wait()

		// Collect results
		for range concurrency {
			select {
			case err := <-errs:
				Expect(err).ToNot(HaveOccurred())
			case bio := <-bios:
				Expect(bio).To(ContainSubstring("Biography for Artist"))
			}
		}
	})

	Describe("sendPluginRefreshEvent", func() {
		var broker *testBroker
		var manager *Manager

		BeforeEach(func() {
			broker = &testBroker{}
			manager = &Manager{
				broker: broker,
			}
		})

		It("sends refresh event with single plugin ID", func() {
			manager.sendPluginRefreshEvent(ctx, "test-plugin")

			Expect(broker.broadcastCalled).To(BeTrue())
			Expect(broker.lastEvent).ToNot(BeNil())
			Expect(broker.lastEventCtx).To(Equal(ctx))

			refreshEvent, ok := broker.lastEvent.(*events.RefreshResource)
			Expect(ok).To(BeTrue(), "event should be a RefreshResource")
			Expect(refreshEvent.Data(refreshEvent)).To(Equal(`{"plugin":["test-plugin"]}`))
		})

		It("sends refresh event with multiple plugin IDs", func() {
			manager.sendPluginRefreshEvent(ctx, "plugin-1", "plugin-2", "plugin-3")

			Expect(broker.broadcastCalled).To(BeTrue())
			refreshEvent, ok := broker.lastEvent.(*events.RefreshResource)
			Expect(ok).To(BeTrue())
			Expect(refreshEvent.Data(refreshEvent)).To(Equal(`{"plugin":["plugin-1","plugin-2","plugin-3"]}`))
		})

		It("sends refresh event with wildcard when using events.Any", func() {
			manager.sendPluginRefreshEvent(ctx, events.Any)

			Expect(broker.broadcastCalled).To(BeTrue())
			refreshEvent, ok := broker.lastEvent.(*events.RefreshResource)
			Expect(ok).To(BeTrue())
			Expect(refreshEvent.Data(refreshEvent)).To(Equal(`{"plugin":["*"]}`))
		})

		It("does not panic when broker is nil", func() {
			manager.broker = nil
			Expect(func() {
				manager.sendPluginRefreshEvent(ctx, "test-plugin")
			}).ToNot(Panic())
		})
	})
})

// testBroker is a simple mock implementation of events.Broker for testing
type testBroker struct {
	lastEvent       events.Event
	lastEventCtx    context.Context
	broadcastCalled bool
}

func (m *testBroker) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// Not used in tests
}

func (m *testBroker) SendMessage(ctx context.Context, event events.Event) {
	// Not used in tests
}

func (m *testBroker) SendBroadcastMessage(ctx context.Context, event events.Event) {
	m.lastEvent = event
	m.lastEventCtx = ctx
	m.broadcastCalled = true
}
