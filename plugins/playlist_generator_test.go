//go:build !windows

package plugins

import (
	"time"

	"github.com/navidrome/navidrome/model/id"
	"github.com/navidrome/navidrome/plugins/capabilities"
	"github.com/navidrome/navidrome/tests"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

// findOrchestrator finds the playlistGeneratorOrchestrator in a plugin's closers.
func findOrchestrator(m *Manager, pluginName string) *playlistGeneratorOrchestrator {
	m.mu.RLock()
	defer m.mu.RUnlock()
	p, ok := m.plugins[pluginName]
	if !ok {
		return nil
	}
	for _, c := range p.closers {
		if orch, ok := c.(*playlistGeneratorOrchestrator); ok {
			return orch
		}
	}
	return nil
}

var _ = Describe("PlaylistGenerator", Ordered, func() {
	var (
		pgManager   *Manager
		mockPlsRepo *tests.MockPlaylistRepo
	)

	BeforeAll(func() {
		pgManager, _ = createTestManagerWithPlugins(nil,
			"test-playlist-generator"+PackageExtension,
		)

		mockPlsRepo = pgManager.ds.(*tests.MockDataStore).MockedPlaylist.(*tests.MockPlaylistRepo)
	})

	Describe("capability detection", func() {
		It("detects the PlaylistGenerator capability", func() {
			names := pgManager.PluginNames(string(CapabilityPlaylistGenerator))
			Expect(names).To(ContainElement("test-playlist-generator"))
		})
	})

	Describe("orchestrator lifecycle", func() {
		It("creates an orchestrator for the plugin", func() {
			Expect(findOrchestrator(pgManager, "test-playlist-generator")).ToNot(BeNil())
		})

		It("discovers and syncs playlists from the plugin", func() {
			// The orchestrator runs discoverAndSync in a goroutine on Start().
			// Give it a moment to complete.
			Eventually(func() int {
				return mockPlsRepo.Len()
			}).Should(BeNumerically(">=", 2))
		})

		It("creates playlists with correct fields", func() {
			Eventually(func() bool {
				return mockPlsRepo.FindByPluginPlaylistID("daily-mix-1") != nil
			}).Should(BeTrue())

			dailyMix1 := mockPlsRepo.FindByPluginPlaylistID("daily-mix-1")
			Expect(dailyMix1.Name).To(Equal("Daily Mix 1"))
			Expect(dailyMix1.Comment).To(Equal("Your personalized daily mix"))
			Expect(dailyMix1.ExternalImageURL).To(Equal("https://example.com/cover1.jpg"))
			Expect(dailyMix1.OwnerID).To(Equal("user-1"))
			Expect(dailyMix1.PluginID).To(Equal("test-playlist-generator"))
			Expect(dailyMix1.PluginPlaylistID).To(Equal("daily-mix-1"))
			Expect(dailyMix1.Public).To(BeFalse())
		})

		It("generates deterministic playlist IDs", func() {
			expectedID := id.NewHash("test-playlist-generator", "daily-mix-1", "user-1")
			Eventually(func() bool {
				_, exists := mockPlsRepo.GetData(expectedID)
				return exists
			}).Should(BeTrue())
		})

		It("creates distinct IDs for different playlists", func() {
			id1 := id.NewHash("test-playlist-generator", "daily-mix-1", "user-1")
			id2 := id.NewHash("test-playlist-generator", "daily-mix-2", "user-1")
			Expect(id1).ToNot(Equal(id2))

			Eventually(func() bool {
				_, exists1 := mockPlsRepo.GetData(id1)
				_, exists2 := mockPlsRepo.GetData(id2)
				return exists1 && exists2
			}).Should(BeTrue())
		})
	})

	Describe("GetAvailablePlaylists error handling", func() {
		It("handles plugin errors gracefully", func() {
			errManager, _ := createTestManagerWithPlugins(map[string]map[string]string{
				"test-playlist-generator": {"error": "service unavailable"},
			}, "test-playlist-generator"+PackageExtension)

			// Should still have the orchestrator (error is logged, not fatal)
			Expect(findOrchestrator(errManager, "test-playlist-generator")).ToNot(BeNil())

			// But no playlists created
			errPlsRepo := errManager.ds.(*tests.MockDataStore).MockedPlaylist.(*tests.MockPlaylistRepo)
			// The orchestrator was started but GetAvailablePlaylists returned error,
			// so no playlists should be created
			Consistently(func() int {
				return errPlsRepo.Len()
			}, "500ms").Should(Equal(0))
		})
	})

	Describe("GetPlaylist NotFound error", func() {
		It("skips playlists when plugin returns NotFound", func() {
			notFoundManager, _ := createTestManagerWithPlugins(map[string]map[string]string{
				"test-playlist-generator": {
					"get_playlist_error":      "playlist temporarily unavailable",
					"get_playlist_error_type": string(capabilities.PlaylistGeneratorErrorNotFound),
				},
			}, "test-playlist-generator"+PackageExtension)

			// Should still have the orchestrator
			Expect(findOrchestrator(notFoundManager, "test-playlist-generator")).ToNot(BeNil())

			// No playlists should be created (all returned NotFound)
			notFoundPlsRepo := notFoundManager.ds.(*tests.MockDataStore).MockedPlaylist.(*tests.MockPlaylistRepo)
			Consistently(func() int {
				return notFoundPlsRepo.Len()
			}, "500ms").Should(Equal(0))

			// No refresh timers should be scheduled for NotFound playlists
			orch := findOrchestrator(notFoundManager, "test-playlist-generator")
			Eventually(func() int32 {
				return orch.refreshTimerCount.Load()
			}).Should(Equal(int32(0)))
		})
	})

	Describe("GetPlaylist transient error with RetryInterval", func() {
		It("stores retryInterval and schedules retry on transient errors", func() {
			retryManager, _ := createTestManagerWithPlugins(map[string]map[string]string{
				"test-playlist-generator": {
					"get_playlist_error": "temporary failure",
					"retry_interval":     "60",
				},
			}, "test-playlist-generator"+PackageExtension)

			orch := findOrchestrator(retryManager, "test-playlist-generator")
			Expect(orch).ToNot(BeNil())

			// retryInterval should be stored from the response
			Eventually(func() time.Duration {
				return time.Duration(orch.retryInterval.Load())
			}).Should(Equal(60 * time.Second))

			// No playlists should be created (GetPlaylist failed)
			retryPlsRepo := retryManager.ds.(*tests.MockDataStore).MockedPlaylist.(*tests.MockPlaylistRepo)
			Consistently(func() int {
				return retryPlsRepo.Len()
			}, "500ms").Should(Equal(0))

			// Refresh timers should be scheduled for transient errors
			Eventually(func() int32 {
				return orch.refreshTimerCount.Load()
			}).Should(BeNumerically(">=", int32(1)))
		})
	})

	Describe("stop", func() {
		It("stops the orchestrator when the manager stops", func() {
			stopManager, _ := createTestManagerWithPlugins(nil,
				"test-playlist-generator"+PackageExtension,
			)
			Expect(findOrchestrator(stopManager, "test-playlist-generator")).ToNot(BeNil())

			err := stopManager.Stop()
			Expect(err).ToNot(HaveOccurred())
			// After Stop(), the plugin is unloaded so findOrchestrator returns nil
			Expect(findOrchestrator(stopManager, "test-playlist-generator")).To(BeNil())
		})
	})
})
