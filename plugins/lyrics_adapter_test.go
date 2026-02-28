//go:build !windows

package plugins

import (
	"github.com/navidrome/navidrome/model"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("LyricsPlugin", Ordered, func() {
	var (
		lyricsManager *Manager
		provider      *LyricsPlugin
	)

	BeforeAll(func() {
		lyricsManager, _ = createTestManagerWithPlugins(nil,
			"test-lyrics"+PackageExtension,
			"test-metadata-agent"+PackageExtension,
		)

		p, ok := lyricsManager.LoadLyricsProvider("test-lyrics")
		Expect(ok).To(BeTrue())
		provider = p.(*LyricsPlugin)
	})

	Describe("LoadLyricsProvider", func() {
		It("returns a lyrics provider for a plugin with Lyrics capability", func() {
			Expect(provider).ToNot(BeNil())
		})

		It("returns false for a plugin without Lyrics capability", func() {
			_, ok := lyricsManager.LoadLyricsProvider("test-metadata-agent")
			Expect(ok).To(BeFalse())
		})

		It("returns false for non-existent plugin", func() {
			_, ok := lyricsManager.LoadLyricsProvider("non-existent")
			Expect(ok).To(BeFalse())
		})
	})

	Describe("GetLyrics", func() {
		It("successfully returns lyrics from the plugin", func() {
			track := &model.MediaFile{
				ID:     "track-1",
				Title:  "Test Song",
				Artist: "Test Artist",
			}

			result, err := provider.GetLyrics(GinkgoT().Context(), track)
			Expect(err).ToNot(HaveOccurred())
			Expect(result).To(HaveLen(1))
			Expect(result[0].Line).ToNot(BeEmpty())
			Expect(result[0].Line[0].Value).To(ContainSubstring("Test Song"))
		})

		It("returns error when plugin returns error", func() {
			manager, _ := createTestManagerWithPlugins(map[string]map[string]string{
				"test-lyrics": {"error": "service unavailable"},
			}, "test-lyrics"+PackageExtension)

			p, ok := manager.LoadLyricsProvider("test-lyrics")
			Expect(ok).To(BeTrue())

			track := &model.MediaFile{ID: "track-1", Title: "Test Song"}
			_, err := p.GetLyrics(GinkgoT().Context(), track)
			Expect(err).To(HaveOccurred())
		})
	})

	Describe("PluginNames", func() {
		It("returns plugin names with Lyrics capability", func() {
			names := lyricsManager.PluginNames("Lyrics")
			Expect(names).To(ContainElement("test-lyrics"))
		})

		It("does not return metadata agent plugins for Lyrics capability", func() {
			names := lyricsManager.PluginNames("Lyrics")
			Expect(names).ToNot(ContainElement("test-metadata-agent"))
		})
	})
})
