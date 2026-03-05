//go:build !windows

package plugins

import (
	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/model/id"
	"github.com/navidrome/navidrome/tests"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("PlaylistGenerator", Ordered, func() {
	var (
		pgManager   *Manager
		mockPlsRepo *tests.MockPlaylistRepo
	)

	BeforeAll(func() {
		pgManager, _ = createTestManagerWithPlugins(nil,
			"test-playlist-generator"+PackageExtension,
		)

		// Pre-initialize the mock playlist repo to avoid a race with the
		// discoverAndSync goroutine that is launched during Start().
		mockDS := pgManager.ds.(*tests.MockDataStore)
		if mockDS.MockedPlaylist == nil {
			mockDS.MockedPlaylist = tests.CreateMockPlaylistRepo()
		}
		mockPlsRepo = mockDS.MockedPlaylist.(*tests.MockPlaylistRepo)
	})

	Describe("capability detection", func() {
		It("detects the PlaylistGenerator capability", func() {
			names := pgManager.PluginNames(string(CapabilityPlaylistGenerator))
			Expect(names).To(ContainElement("test-playlist-generator"))
		})
	})

	Describe("startPlaylistGenerators", func() {
		It("creates an orchestrator for the plugin", func() {
			Expect(pgManager.playlistGenerators).To(HaveKey("test-playlist-generator"))
		})

		It("discovers and syncs playlists from the plugin", func() {
			// The orchestrator runs discoverAndSync in a goroutine on Start().
			// Give it a moment to complete.
			Eventually(func() int {
				return len(mockPlsRepo.Data)
			}).Should(BeNumerically(">=", 2))
		})

		It("creates playlists with correct fields", func() {
			// Check that playlists have the correct plugin fields
			Eventually(func() bool {
				for _, pls := range mockPlsRepo.Data {
					if pls.PluginID == "test-playlist-generator" && pls.PluginPlaylistID == "daily-mix-1" {
						return true
					}
				}
				return false
			}).Should(BeTrue())

			// Find the daily-mix-1 playlist and verify its fields
			var dailyMix1 *model.Playlist
			for _, pls := range mockPlsRepo.Data {
				if pls.PluginPlaylistID == "daily-mix-1" {
					dailyMix1 = pls
					break
				}
			}
			Expect(dailyMix1).ToNot(BeNil())
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
				_, exists := mockPlsRepo.Data[expectedID]
				return exists
			}).Should(BeTrue())
		})

		It("creates distinct IDs for different playlists", func() {
			id1 := id.NewHash("test-playlist-generator", "daily-mix-1", "user-1")
			id2 := id.NewHash("test-playlist-generator", "daily-mix-2", "user-1")
			Expect(id1).ToNot(Equal(id2))

			Eventually(func() bool {
				_, exists1 := mockPlsRepo.Data[id1]
				_, exists2 := mockPlsRepo.Data[id2]
				return exists1 && exists2
			}).Should(BeTrue())
		})
	})

	Describe("GetPlaylists error handling", func() {
		It("handles plugin errors gracefully", func() {
			errManager, _ := createTestManagerWithPlugins(map[string]map[string]string{
				"test-playlist-generator": {"error": "service unavailable"},
			}, "test-playlist-generator"+PackageExtension)

			// Should still have the orchestrator (error is logged, not fatal)
			Expect(errManager.playlistGenerators).To(HaveKey("test-playlist-generator"))

			// But no playlists created
			errDS := errManager.ds.(*tests.MockDataStore)
			if errDS.MockedPlaylist == nil {
				errDS.MockedPlaylist = tests.CreateMockPlaylistRepo()
			}
			errPlsRepo := errDS.MockedPlaylist.(*tests.MockPlaylistRepo)
			// The orchestrator was started but GetPlaylists returned error,
			// so no playlists should be created
			Consistently(func() int {
				return len(errPlsRepo.Data)
			}, "500ms").Should(Equal(0))
		})
	})

	Describe("stop", func() {
		It("stops the orchestrator when the manager stops", func() {
			stopManager, _ := createTestManagerWithPlugins(nil,
				"test-playlist-generator"+PackageExtension,
			)
			Expect(stopManager.playlistGenerators).To(HaveKey("test-playlist-generator"))

			err := stopManager.Stop()
			Expect(err).ToNot(HaveOccurred())
			Expect(stopManager.playlistGenerators).To(BeEmpty())
		})
	})
})
