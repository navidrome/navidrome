package model_test

import (
	"path"

	"github.com/navidrome/navidrome/conf"
	"github.com/navidrome/navidrome/conf/configtest"
	"github.com/navidrome/navidrome/model"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Podcasts", func() {
	BeforeEach(func() {
		DeferCleanup(configtest.SetupConfig())
	})

	Describe("ExtractExternalId", func() {
		It("should strip out external prefix off of podcast and episode", func() {
			Expect(model.ExtractExternalId("pd-2345")).To(Equal("2345"))
			Expect(model.ExtractExternalId("pe-1234")).To(Equal("1234"))
		})
	})

	Context("Podcast", func() {
		p := model.Podcast{ID: "1234"}

		Describe("AbsolutePath", func() {
			It("should return path from server configuration", func() {
				conf.Server.Podcast.Path = "tmp"

				Expect(p.AbsolutePath()).To(Equal(path.Join("tmp", "1234")))
			})
		})

		Describe("ExternalId", func() {
			It("should return pd-ID for external ID", func() {
				Expect(p.ExternalId()).To(Equal("pd-1234"))
			})
		})

		Describe("IsPodcastId", func() {
			It("should return true for valid podcast id", func() {
				Expect(model.IsPodcastId("pd-1234")).To(BeTrue())
			})

			It("should return false if the id is just pd-", func() {
				Expect(model.IsPodcastId("pd-")).To(BeFalse())
			})

			It("should return false for empty string", func() {
				Expect(model.IsPodcastId("")).To(BeFalse())
			})
		})
	})

	Context("PodcastEpisode", func() {
		episode := model.PodcastEpisode{BitRate: 128, Duration: 1234, ID: "1234", PodcastId: "4321", Size: 4100512, Suffix: "mp3"}
		episode.PlayCount = 5
		episode.Rating = 3
		episode.Starred = true

		Describe("AbsolutePath", func() {
			It("should handle absolute path", func() {
				conf.Server.Podcast.Path = "tmp"
				Expect(episode.AbsolutePath()).To(Equal(path.Join("tmp", "4321", "1234.mp3")))
			})
		})

		Describe("BasePath", func() {
			It("should handle base path", func() {
				Expect(episode.BasePath()).To(Equal(path.Join("4321", "1234.mp3")))
			})
		})

		Describe("ExternalId", func() {
			It("should return correct external ID", func() {
				Expect(episode.ExternalId()).To(Equal("pe-1234"))
			})
		})

		Describe("IsPodcastEpisodeId", func() {
			It("should return true for valid podcast id", func() {
				Expect(model.IsPodcastEpisodeId("pe-1234")).To(BeTrue())
			})

			It("should return false if the id is just pe-", func() {
				Expect(model.IsPodcastEpisodeId("pe-")).To(BeFalse())
			})

			It("should return false for empty string", func() {
				Expect(model.IsPodcastEpisodeId("")).To(BeFalse())
			})
		})

		Describe("ToMediaFile", func() {
			It("should convert to mediafile with necessary fields", func() {
				conf.Server.Podcast.Path = "tmp"

				Expect(episode.ToMediaFile()).To(Equal(&model.MediaFile{
					Annotations: model.Annotations{
						PlayCount: 5,
						Rating:    3,
						Starred:   true,
					},
					ID:       "pe-1234",
					BitRate:  128,
					Duration: 1234,
					Path:     path.Join("tmp", "4321", "1234.mp3"),
					Size:     4100512,
					Suffix:   "mp3",
				}))
			})
		})
	})
})
