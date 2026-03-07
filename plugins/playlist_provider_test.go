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

// findSyncer finds the playlistSyncer in a plugin's closers.
func findSyncer(m *Manager, pluginName string) *playlistSyncer {
	m.mu.RLock()
	defer m.mu.RUnlock()
	p, ok := m.plugins[pluginName]
	if !ok {
		return nil
	}
	for _, c := range p.closers {
		if syncer, ok := c.(*playlistSyncer); ok {
			return syncer
		}
	}
	return nil
}

var _ = Describe("PlaylistProvider", Ordered, func() {
	var (
		pgManager   *Manager
		mockPlsRepo *tests.MockPlaylistRepo
	)

	BeforeAll(func() {
		pgManager, _ = createTestManagerWithPlugins(nil,
			"test-playlist-provider"+PackageExtension,
		)

		mockPlsRepo = pgManager.ds.(*tests.MockDataStore).MockedPlaylist.(*tests.MockPlaylistRepo)
	})

	Describe("capability detection", func() {
		It("detects the PlaylistProvider capability", func() {
			names := pgManager.PluginNames(string(CapabilityPlaylistProvider))
			Expect(names).To(ContainElement("test-playlist-provider"))
		})
	})

	Describe("syncer lifecycle", func() {
		It("creates a syncer for the plugin", func() {
			Expect(findSyncer(pgManager, "test-playlist-provider")).ToNot(BeNil())
		})

		It("discovers and syncs playlists from the plugin", func() {
			// The syncer runs discoverAndSync in a goroutine on Start().
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
			Expect(dailyMix1.PluginID).To(Equal("test-playlist-provider"))
			Expect(dailyMix1.PluginPlaylistID).To(Equal("daily-mix-1"))
			Expect(dailyMix1.Public).To(BeFalse())
			Expect(dailyMix1.ValidUntil).To(BeNil())
		})

		It("generates deterministic playlist IDs", func() {
			expectedID := id.NewHash("test-playlist-provider", "daily-mix-1", "user-1")
			Eventually(func() bool {
				_, exists := mockPlsRepo.GetData(expectedID)
				return exists
			}).Should(BeTrue())
		})

		It("creates distinct IDs for different playlists", func() {
			id1 := id.NewHash("test-playlist-provider", "daily-mix-1", "user-1")
			id2 := id.NewHash("test-playlist-provider", "daily-mix-2", "user-1")
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
				"test-playlist-provider": {"error": "service unavailable"},
			}, "test-playlist-provider"+PackageExtension)

			// Should still have the syncer (error is logged, not fatal)
			Expect(findSyncer(errManager, "test-playlist-provider")).ToNot(BeNil())

			// But no playlists created
			errPlsRepo := errManager.ds.(*tests.MockDataStore).MockedPlaylist.(*tests.MockPlaylistRepo)
			// The syncer was started but GetAvailablePlaylists returned error,
			// so no playlists should be created
			Consistently(func() int {
				return errPlsRepo.Len()
			}, "500ms").Should(Equal(0))
		})
	})

	Describe("GetPlaylist NotFound error", func() {
		It("skips playlists when plugin returns NotFound", func() {
			notFoundManager, _ := createTestManagerWithPlugins(map[string]map[string]string{
				"test-playlist-provider": {
					"get_playlist_error":      "playlist temporarily unavailable",
					"get_playlist_error_type": string(capabilities.PlaylistProviderErrorNotFound),
				},
			}, "test-playlist-provider"+PackageExtension)

			// Should still have the syncer
			Expect(findSyncer(notFoundManager, "test-playlist-provider")).ToNot(BeNil())

			// No playlists should be created (all returned NotFound)
			notFoundPlsRepo := notFoundManager.ds.(*tests.MockDataStore).MockedPlaylist.(*tests.MockPlaylistRepo)
			Consistently(func() int {
				return notFoundPlsRepo.Len()
			}, "500ms").Should(Equal(0))

			// No refresh timers should be scheduled for NotFound playlists
			syncer := findSyncer(notFoundManager, "test-playlist-provider")
			Eventually(func() int32 {
				return syncer.refreshTimerCount.Load()
			}).Should(Equal(int32(0)))
		})
	})

	Describe("GetPlaylist transient error with RetryInterval", func() {
		It("stores retryInterval and schedules retry on transient errors", func() {
			retryManager, _ := createTestManagerWithPlugins(map[string]map[string]string{
				"test-playlist-provider": {
					"get_playlist_error": "temporary failure",
					"retry_interval":     "60",
				},
			}, "test-playlist-provider"+PackageExtension)

			syncer := findSyncer(retryManager, "test-playlist-provider")
			Expect(syncer).ToNot(BeNil())

			// retryInterval should be stored from the response
			Eventually(func() time.Duration {
				return time.Duration(syncer.retryInterval.Load())
			}).Should(Equal(60 * time.Second))

			// No playlists should be created (GetPlaylist failed)
			retryPlsRepo := retryManager.ds.(*tests.MockDataStore).MockedPlaylist.(*tests.MockPlaylistRepo)
			Consistently(func() int {
				return retryPlsRepo.Len()
			}, "500ms").Should(Equal(0))

			// Refresh timers should be scheduled for transient errors
			Eventually(func() int32 {
				return syncer.refreshTimerCount.Load()
			}).Should(BeNumerically(">=", int32(1)))
		})
	})

	Describe("user permission validation", func() {
		It("skips playlists for unauthorized users when AllUsers is false", func() {
			// Create manager with restricted users — only "other-user" is allowed,
			// but the plugin returns playlists for "admin" which resolves to "user-1"
			restrictedManager, _ := createTestManagerWithPluginOverrides(nil,
				map[string]pluginOverride{
					"test-playlist-provider": {AllUsers: false, Users: `["other-user"]`},
				},
				"test-playlist-provider"+PackageExtension,
			)

			// No playlists should be created because "user-1" is not in allowed users
			restrictedPlsRepo := restrictedManager.ds.(*tests.MockDataStore).MockedPlaylist.(*tests.MockPlaylistRepo)
			Consistently(func() int {
				return restrictedPlsRepo.Len()
			}, "500ms").Should(Equal(0))
		})

		It("creates playlists for authorized users when AllUsers is false", func() {
			// Create manager with restricted users — "user-1" is allowed,
			// and the plugin returns playlists for "admin" which resolves to "user-1"
			allowedManager, _ := createTestManagerWithPluginOverrides(nil,
				map[string]pluginOverride{
					"test-playlist-provider": {AllUsers: false, Users: `["user-1"]`},
				},
				"test-playlist-provider"+PackageExtension,
			)

			// Playlists should be created because "user-1" is in allowed users
			allowedPlsRepo := allowedManager.ds.(*tests.MockDataStore).MockedPlaylist.(*tests.MockPlaylistRepo)
			Eventually(func() int {
				return allowedPlsRepo.Len()
			}).Should(BeNumerically(">=", 2))
		})
	})

	Describe("stop", func() {
		It("stops the syncer when the manager stops", func() {
			stopManager, _ := createTestManagerWithPlugins(nil,
				"test-playlist-provider"+PackageExtension,
			)
			Expect(findSyncer(stopManager, "test-playlist-provider")).ToNot(BeNil())

			err := stopManager.Stop()
			Expect(err).ToNot(HaveOccurred())
			// After Stop(), the plugin is unloaded so findSyncer returns nil
			Expect(findSyncer(stopManager, "test-playlist-provider")).To(BeNil())
		})
	})
})
