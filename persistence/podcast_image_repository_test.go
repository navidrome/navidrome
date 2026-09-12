package persistence

import (
	"context"

	"github.com/navidrome/navidrome/log"
	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/model/request"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("PodcastImageRepository", func() {
	var repo model.PodcastImageRepository

	BeforeEach(func() {
		ctx := log.NewContext(context.TODO())
		ctx = request.WithUser(ctx, adminUser)
		repo = NewPodcastImageRepository(ctx, GetDBXBuilder())
	})

	Describe("SaveForChannel and GetByChannel", func() {
		It("saves and retrieves channel images", func() {
			images := []model.PodcastImage{
				{URL: "https://example.com/img-3000.jpg", Width: 3000},
				{URL: "https://example.com/img-300.jpg", Width: 300},
			}
			Expect(repo.SaveForChannel("pc-1", images)).To(Succeed())

			result, err := repo.GetByChannel("pc-1")
			Expect(err).ToNot(HaveOccurred())
			Expect(result).To(HaveLen(2))

			Expect(repo.SaveForChannel("pc-1", nil)).To(Succeed())
		})

		It("persists URL and Width correctly", func() {
			Expect(repo.SaveForChannel("pc-1", []model.PodcastImage{
				{URL: "https://example.com/img-1500.jpg", Width: 1500},
			})).To(Succeed())

			result, _ := repo.GetByChannel("pc-1")
			Expect(result[0].URL).To(Equal("https://example.com/img-1500.jpg"))
			Expect(result[0].Width).To(Equal(1500))

			Expect(repo.SaveForChannel("pc-1", nil)).To(Succeed())
		})

		It("sets ChannelID and empty EpisodeID on returned items", func() {
			Expect(repo.SaveForChannel("pc-1", []model.PodcastImage{
				{URL: "https://example.com/img.jpg", Width: 600},
			})).To(Succeed())

			result, _ := repo.GetByChannel("pc-1")
			Expect(result[0].ChannelID).To(Equal("pc-1"))
			Expect(result[0].EpisodeID).To(BeEmpty())

			Expect(repo.SaveForChannel("pc-1", nil)).To(Succeed())
		})

		It("assigns non-empty ID automatically", func() {
			Expect(repo.SaveForChannel("pc-1", []model.PodcastImage{
				{URL: "https://example.com/img.jpg"},
			})).To(Succeed())

			result, _ := repo.GetByChannel("pc-1")
			Expect(result[0].ID).ToNot(BeEmpty())

			Expect(repo.SaveForChannel("pc-1", nil)).To(Succeed())
		})

		It("replaces existing images on re-save", func() {
			Expect(repo.SaveForChannel("pc-1", []model.PodcastImage{
				{URL: "https://example.com/old.jpg", Width: 100},
			})).To(Succeed())
			Expect(repo.SaveForChannel("pc-1", []model.PodcastImage{
				{URL: "https://example.com/new.jpg", Width: 200},
			})).To(Succeed())

			result, _ := repo.GetByChannel("pc-1")
			Expect(result).To(HaveLen(1))
			Expect(result[0].URL).To(Equal("https://example.com/new.jpg"))

			Expect(repo.SaveForChannel("pc-1", nil)).To(Succeed())
		})

		It("clears images when saved with nil slice", func() {
			Expect(repo.SaveForChannel("pc-1", []model.PodcastImage{
				{URL: "https://example.com/img.jpg"},
			})).To(Succeed())
			Expect(repo.SaveForChannel("pc-1", nil)).To(Succeed())

			result, _ := repo.GetByChannel("pc-1")
			Expect(result).To(BeEmpty())
		})

		It("does not return episode images for channel queries", func() {
			Expect(repo.SaveForChannel("pc-1", []model.PodcastImage{
				{URL: "https://example.com/channel.jpg", Width: 3000},
			})).To(Succeed())
			Expect(repo.SaveForEpisode("ep-1", []model.PodcastImage{
				{URL: "https://example.com/episode.jpg", Width: 600},
			})).To(Succeed())

			result, _ := repo.GetByChannel("pc-1")
			Expect(result).To(HaveLen(1))
			Expect(result[0].URL).To(Equal("https://example.com/channel.jpg"))

			Expect(repo.SaveForChannel("pc-1", nil)).To(Succeed())
			Expect(repo.SaveForEpisode("ep-1", nil)).To(Succeed())
		})

		It("returns empty list for unknown channel", func() {
			result, err := repo.GetByChannel("no-such-channel")
			Expect(err).ToNot(HaveOccurred())
			Expect(result).To(BeEmpty())
		})
	})

	Describe("SaveForEpisode and GetByEpisode", func() {
		It("saves and retrieves episode images", func() {
			images := []model.PodcastImage{
				{URL: "https://example.com/ep-600.jpg", Width: 600},
				{URL: "https://example.com/ep-150.jpg", Width: 150},
			}
			Expect(repo.SaveForEpisode("ep-1", images)).To(Succeed())

			result, err := repo.GetByEpisode("ep-1")
			Expect(err).ToNot(HaveOccurred())
			Expect(result).To(HaveLen(2))

			Expect(repo.SaveForEpisode("ep-1", nil)).To(Succeed())
		})

		It("sets EpisodeID and empty ChannelID on returned items", func() {
			Expect(repo.SaveForEpisode("ep-1", []model.PodcastImage{
				{URL: "https://example.com/ep.jpg", Width: 600},
			})).To(Succeed())

			result, _ := repo.GetByEpisode("ep-1")
			Expect(result[0].EpisodeID).To(Equal("ep-1"))
			Expect(result[0].ChannelID).To(BeEmpty())

			Expect(repo.SaveForEpisode("ep-1", nil)).To(Succeed())
		})

		It("does not return channel images for episode queries", func() {
			Expect(repo.SaveForChannel("pc-1", []model.PodcastImage{
				{URL: "https://example.com/channel.jpg", Width: 3000},
			})).To(Succeed())
			Expect(repo.SaveForEpisode("ep-1", []model.PodcastImage{
				{URL: "https://example.com/episode.jpg", Width: 600},
			})).To(Succeed())

			result, _ := repo.GetByEpisode("ep-1")
			Expect(result).To(HaveLen(1))
			Expect(result[0].URL).To(Equal("https://example.com/episode.jpg"))

			Expect(repo.SaveForChannel("pc-1", nil)).To(Succeed())
			Expect(repo.SaveForEpisode("ep-1", nil)).To(Succeed())
		})

		It("returns empty list for unknown episode", func() {
			result, err := repo.GetByEpisode("no-such-episode")
			Expect(err).ToNot(HaveOccurred())
			Expect(result).To(BeEmpty())
		})
	})

	Describe("GetByChannels — bulk query", func() {
		It("returns images for multiple channels", func() {
			Expect(repo.SaveForChannel("pc-1", []model.PodcastImage{
				{URL: "https://a.example.com/img.jpg", Width: 3000},
			})).To(Succeed())
			Expect(repo.SaveForChannel("pc-2", []model.PodcastImage{
				{URL: "https://b.example.com/img.jpg", Width: 1500},
			})).To(Succeed())

			result, err := repo.GetByChannels([]string{"pc-1", "pc-2"})
			Expect(err).ToNot(HaveOccurred())
			Expect(result).To(HaveLen(2))

			Expect(repo.SaveForChannel("pc-1", nil)).To(Succeed())
			Expect(repo.SaveForChannel("pc-2", nil)).To(Succeed())
		})

		It("returned items carry ChannelID for grouping", func() {
			Expect(repo.SaveForChannel("pc-1", []model.PodcastImage{
				{URL: "https://example.com/img.jpg", Width: 600},
			})).To(Succeed())

			result, _ := repo.GetByChannels([]string{"pc-1"})
			Expect(result[0].ChannelID).To(Equal("pc-1"))

			Expect(repo.SaveForChannel("pc-1", nil)).To(Succeed())
		})

		It("returns nil for empty id slice", func() {
			result, err := repo.GetByChannels([]string{})
			Expect(err).ToNot(HaveOccurred())
			Expect(result).To(BeNil())
		})
	})

	Describe("GetByEpisodes — bulk query", func() {
		It("returns images for multiple episodes", func() {
			Expect(repo.SaveForEpisode("ep-1", []model.PodcastImage{
				{URL: "https://ep1.example.com/img.jpg", Width: 600},
			})).To(Succeed())
			Expect(repo.SaveForEpisode("ep-2", []model.PodcastImage{
				{URL: "https://ep2.example.com/img.jpg", Width: 300},
			})).To(Succeed())

			result, err := repo.GetByEpisodes([]string{"ep-1", "ep-2"})
			Expect(err).ToNot(HaveOccurred())
			Expect(result).To(HaveLen(2))

			Expect(repo.SaveForEpisode("ep-1", nil)).To(Succeed())
			Expect(repo.SaveForEpisode("ep-2", nil)).To(Succeed())
		})

		It("returned items carry EpisodeID for grouping", func() {
			Expect(repo.SaveForEpisode("ep-1", []model.PodcastImage{
				{URL: "https://example.com/ep.jpg", Width: 600},
			})).To(Succeed())

			result, _ := repo.GetByEpisodes([]string{"ep-1"})
			Expect(result[0].EpisodeID).To(Equal("ep-1"))

			Expect(repo.SaveForEpisode("ep-1", nil)).To(Succeed())
		})

		It("returns nil for empty id slice", func() {
			result, err := repo.GetByEpisodes([]string{})
			Expect(err).ToNot(HaveOccurred())
			Expect(result).To(BeNil())
		})
	})
})
