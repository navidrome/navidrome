package model_test

import (
	"path"
	"time"

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
			conf.Server.Podcast.Path = "tmp"

			It("should return path from server configuration", func() {
				Expect(p.AbsolutePath()).To(Equal(path.Join("tmp", "1234")))
			})

			It("should prevent directory traversal with bad id", func() {
				bad := model.Podcast{ID: "../../root"}
				Expect(bad.AbsolutePath()).To(Equal(path.Join("tmp", "root")))

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
		baseTime := time.Date(1970, 1, 1, 0, 0, 0, 0, time.UTC)
		episode := model.PodcastEpisode{BitRate: 128, Duration: 1234, ID: "1234", PodcastId: "4321", Size: 4100512, Title: "Title", Suffix: "mp3", CreatedAt: baseTime, UpdatedAt: baseTime}
		episode.PlayDate = &time.Time{}
		episode.PlayCount = 5
		episode.Rating = 3
		episode.Starred = true
		episode.StarredAt = &baseTime
		episode.BookmarkPosition = int64(46)

		Describe("AbsolutePath", func() {
			conf.Server.Podcast.Path = "tmp"

			It("should handle absolute path", func() {
				Expect(episode.AbsolutePath()).To(Equal(path.Join("tmp", "4321", "1234.mp3")))
			})

			It("should prevent directory traversal for podcast episode id", func() {
				bad := model.PodcastEpisode{ID: "../../hi", PodcastId: "234", Suffix: "mp3"}
				Expect(bad.AbsolutePath()).To(Equal(path.Join("tmp", "234", "hi.mp3")))
			})

			It("prevent directory traversal for channel and episode id", func() {
				bad := model.PodcastEpisode{ID: "../../hi", PodcastId: "../../234", Suffix: "mp3"}
				Expect(bad.AbsolutePath()).To(Equal(path.Join("tmp", "234", "hi.mp3")))
			})
		})

		Describe("BasePath", func() {
			It("should handle base path", func() {
				Expect(episode.BasePath()).To(Equal(path.Join("4321", "1234.mp3")))
			})

			It("should prevent directory traversal", func() {
				bad := model.PodcastEpisode{ID: "../../hi", PodcastId: "../../234", Suffix: "mp3"}
				Expect(bad.BasePath()).To(Equal(path.Join("234", "hi.mp3")))
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
						PlayDate:  &time.Time{},
						PlayCount: 5,
						Rating:    3,
						Starred:   true,
						StarredAt: &baseTime,
					},
					Bookmarkable: model.Bookmarkable{
						BookmarkPosition: 46,
					},
					ID:        "pe-1234",
					AlbumID:   "pd-4321",
					BitRate:   128,
					Duration:  1234,
					Path:      path.Join("tmp", "4321", "1234.mp3"),
					Size:      4100512,
					Suffix:    "mp3",
					Title:     "Title",
					CreatedAt: baseTime,
					UpdatedAt: baseTime,
				}))
			})
		})
	})
})
