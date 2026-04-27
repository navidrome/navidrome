//go:build !windows

package plugins

import (
	"github.com/navidrome/navidrome/core/sonic"
	"github.com/navidrome/navidrome/model"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("SonicSimilarityPlugin", Ordered, func() {
	var (
		manager  *Manager
		provider sonic.Provider
	)

	BeforeAll(func() {
		manager, _ = createTestManagerWithPlugins(nil, "test-sonic-similarity"+PackageExtension)

		var ok bool
		provider, ok = manager.LoadSonicSimilarity("test-sonic-similarity")
		Expect(ok).To(BeTrue())
	})

	Describe("PluginNames", func() {
		It("reports the sonic similarity capability", func() {
			names := manager.PluginNames(string(CapabilitySonicSimilarity))
			Expect(names).To(ContainElement("test-sonic-similarity"))
		})
	})

	Describe("GetSonicSimilarTracks", func() {
		It("returns similar tracks from the plugin", func() {
			mf := &model.MediaFile{
				ID:     "track-1",
				Title:  "Yesterday",
				Artist: "The Beatles",
			}

			results, err := provider.GetSonicSimilarTracks(GinkgoT().Context(), mf, 3)
			Expect(err).ToNot(HaveOccurred())
			Expect(results).To(HaveLen(3))
			Expect(results[0].Song.Name).To(Equal("Similar to Yesterday #1"))
			Expect(results[0].Song.Artist).To(Equal("The Beatles"))
			Expect(results[0].Similarity).To(Equal(1.0))
			Expect(results[1].Similarity).To(Equal(0.9))
			Expect(results[2].Similarity).To(Equal(0.8))
		})
	})

	Describe("FindSonicPath", func() {
		It("returns a path between two tracks from the plugin", func() {
			startMf := &model.MediaFile{
				ID:     "track-1",
				Title:  "Yesterday",
				Artist: "The Beatles",
			}

			endMf := &model.MediaFile{
				ID:     "track-2",
				Title:  "Tomorrow Never Knows",
				Artist: "The Beatles",
			}

			results, err := provider.FindSonicPath(GinkgoT().Context(), startMf, endMf, 3)
			Expect(err).ToNot(HaveOccurred())
			Expect(results).To(HaveLen(3))
			Expect(results[0].Song.Name).To(Equal("Path Yesterday to Tomorrow Never Knows #1"))
			Expect(results[0].Song.Artist).To(Equal("The Beatles"))
			Expect(results[0].Similarity).To(Equal(1.0))
			Expect(results[1].Similarity).To(Equal(0.95))
			Expect(results[2].Similarity).To(Equal(0.9))
		})
	})
})

var _ = Describe("SonicSimilarityPlugin error handling", Ordered, func() {
	var (
		errorManager  *Manager
		errorProvider sonic.Provider
	)

	BeforeAll(func() {
		errorManager, _ = createTestManagerWithPlugins(map[string]map[string]string{
			"test-sonic-similarity": {
				"error": "simulated plugin error",
			},
		}, "test-sonic-similarity"+PackageExtension)

		var ok bool
		errorProvider, ok = errorManager.LoadSonicSimilarity("test-sonic-similarity")
		Expect(ok).To(BeTrue())
	})

	It("returns error from GetSonicSimilarTracks", func() {
		mf := &model.MediaFile{ID: "track-1", Title: "Test"}
		_, err := errorProvider.GetSonicSimilarTracks(GinkgoT().Context(), mf, 3)
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("simulated plugin error"))
	})

	It("returns error from FindSonicPath", func() {
		startMf := &model.MediaFile{ID: "track-1", Title: "Start"}
		endMf := &model.MediaFile{ID: "track-2", Title: "End"}
		_, err := errorProvider.FindSonicPath(GinkgoT().Context(), startMf, endMf, 3)
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("simulated plugin error"))
	})
})
