//go:build !windows

package plugins

import (
	"context"

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

		It("defaults language to 'xxx' when plugin does not provide one", func() {
			manager, _ := createTestManagerWithPlugins(map[string]map[string]string{
				"test-lyrics": {"no_lang": "true"},
			}, "test-lyrics"+PackageExtension)

			p, ok := manager.LoadLyricsProvider("test-lyrics")
			Expect(ok).To(BeTrue())

			track := &model.MediaFile{ID: "track-1", Title: "Test Song", Artist: "Test Artist"}
			result, err := p.GetLyrics(GinkgoT().Context(), track)
			Expect(err).ToNot(HaveOccurred())
			Expect(result).To(HaveLen(1))
			Expect(result[0].Lang).To(Equal("xxx"))
		})

		It("blocks new calls while the per-plugin concurrency cap is saturated", func() {
			sem := provider.plugin.lyricsSem
			for range cap(sem) {
				sem <- struct{}{}
			}

			ctx := GinkgoT().Context()
			track := &model.MediaFile{ID: "track-1", Title: "Test Song", Artist: "Test Artist"}
			done := make(chan error, 1)
			go func() {
				_, err := provider.GetLyrics(ctx, track)
				done <- err
			}()

			Consistently(done, "500ms").ShouldNot(Receive())
			<-sem // free one slot; the pending call should now proceed
			Eventually(done).Should(Receive(BeNil()))
			for range cap(sem) - 1 {
				<-sem
			}
		})

		It("gives up waiting for a slot when the context is cancelled", func() {
			sem := provider.plugin.lyricsSem
			for range cap(sem) {
				sem <- struct{}{}
			}
			defer func() {
				for range cap(sem) {
					<-sem
				}
			}()

			ctx, cancel := context.WithCancel(GinkgoT().Context())
			cancel()
			_, err := provider.GetLyrics(ctx, &model.MediaFile{ID: "track-1"})
			Expect(err).To(MatchError(context.Canceled))
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

		// Each DescribeTable entry proves that the adapter's content-sniffing routes
		// the plugin's rich payload to the right parser rather than mangling it as plain text.
		DescribeTable("content-sniffs plugin responses across all supported formats",
			func(format string, wantSynced bool, wantLine string) {
				manager, _ := createTestManagerWithPlugins(map[string]map[string]string{
					"test-lyrics": {"format": format},
				}, "test-lyrics"+PackageExtension)

				p, ok := manager.LoadLyricsProvider("test-lyrics")
				Expect(ok).To(BeTrue())

				track := &model.MediaFile{ID: "track-1", Title: "Test Song", Artist: "Test Artist"}
				result, err := p.GetLyrics(GinkgoT().Context(), track)
				Expect(err).ToNot(HaveOccurred())
				Expect(result).To(HaveLen(1))
				Expect(result[0].Synced).To(Equal(wantSynced), "unexpected Synced value for format %s", format)
				Expect(result[0].Line).To(HaveLen(1))
				Expect(result[0].Line[0].Value).To(Equal(wantLine))
			},
			Entry("ttml", "ttml", true, "plugin ttml line"),
			Entry("srt", "srt", true, "plugin srt line"),
			Entry("yaml", "yaml", true, "plugin yaml line"),
			Entry("lrc", "lrc", true, "plugin lrc line"),
			Entry("plain", "plain", false, "plugin plain line"),
		)
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
