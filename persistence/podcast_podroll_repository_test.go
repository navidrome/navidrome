package persistence

import (
	"context"

	"github.com/navidrome/navidrome/log"
	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/model/request"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("PodcastPodrollRepository", func() {
	var repo model.PodcastPodrollRepository

	BeforeEach(func() {
		ctx := log.NewContext(context.TODO())
		ctx = request.WithUser(ctx, adminUser)
		repo = NewPodcastPodrollRepository(ctx, GetDBXBuilder())
	})

	Describe("SaveForChannel and GetByChannel", func() {
		It("saves and retrieves podroll items", func() {
			items := []model.PodcastPodrollItem{
				{FeedGUID: "guid-a", FeedURL: "https://a.example.com/feed.xml", Title: "Show A"},
				{FeedGUID: "guid-b", FeedURL: "https://b.example.com/feed.xml", Title: "Show B"},
			}
			Expect(repo.SaveForChannel("pc-1", items)).To(Succeed())

			result, err := repo.GetByChannel("pc-1")
			Expect(err).ToNot(HaveOccurred())
			Expect(result).To(HaveLen(2))

			Expect(repo.SaveForChannel("pc-1", nil)).To(Succeed())
		})

		It("preserves sort_order in insertion order", func() {
			items := []model.PodcastPodrollItem{
				{FeedURL: "https://first.example.com/feed.xml", Title: "First"},
				{FeedURL: "https://second.example.com/feed.xml", Title: "Second"},
				{FeedURL: "https://third.example.com/feed.xml", Title: "Third"},
			}
			Expect(repo.SaveForChannel("pc-1", items)).To(Succeed())

			result, err := repo.GetByChannel("pc-1")
			Expect(err).ToNot(HaveOccurred())
			Expect(result).To(HaveLen(3))
			Expect(result[0].Title).To(Equal("First"))
			Expect(result[1].Title).To(Equal("Second"))
			Expect(result[2].Title).To(Equal("Third"))

			Expect(repo.SaveForChannel("pc-1", nil)).To(Succeed())
		})

		It("replaces existing items on re-save", func() {
			Expect(repo.SaveForChannel("pc-1", []model.PodcastPodrollItem{
				{FeedURL: "https://old.example.com/feed.xml", Title: "Old Show"},
			})).To(Succeed())
			Expect(repo.SaveForChannel("pc-1", []model.PodcastPodrollItem{
				{FeedURL: "https://new.example.com/feed.xml", Title: "New Show"},
			})).To(Succeed())

			result, err := repo.GetByChannel("pc-1")
			Expect(err).ToNot(HaveOccurred())
			Expect(result).To(HaveLen(1))
			Expect(result[0].Title).To(Equal("New Show"))

			Expect(repo.SaveForChannel("pc-1", nil)).To(Succeed())
		})

		It("clears items when saved with nil slice", func() {
			Expect(repo.SaveForChannel("pc-1", []model.PodcastPodrollItem{
				{FeedURL: "https://example.com/feed.xml", Title: "Some Show"},
			})).To(Succeed())
			Expect(repo.SaveForChannel("pc-1", nil)).To(Succeed())

			result, err := repo.GetByChannel("pc-1")
			Expect(err).ToNot(HaveOccurred())
			Expect(result).To(BeEmpty())
		})

		It("does not affect items of other channels", func() {
			Expect(repo.SaveForChannel("pc-1", []model.PodcastPodrollItem{
				{FeedURL: "https://a.example.com/feed.xml", Title: "Channel A Feed"},
			})).To(Succeed())
			Expect(repo.SaveForChannel("pc-2", []model.PodcastPodrollItem{
				{FeedURL: "https://b.example.com/feed.xml", Title: "Channel B Feed"},
			})).To(Succeed())

			resultA, _ := repo.GetByChannel("pc-1")
			resultB, _ := repo.GetByChannel("pc-2")
			Expect(resultA).To(HaveLen(1))
			Expect(resultB).To(HaveLen(1))
			Expect(resultA[0].Title).To(Equal("Channel A Feed"))
			Expect(resultB[0].Title).To(Equal("Channel B Feed"))

			Expect(repo.SaveForChannel("pc-1", nil)).To(Succeed())
			Expect(repo.SaveForChannel("pc-2", nil)).To(Succeed())
		})

		It("returns empty list for unknown channel", func() {
			result, err := repo.GetByChannel("no-such-channel")
			Expect(err).ToNot(HaveOccurred())
			Expect(result).To(BeEmpty())
		})

		It("assigns ID automatically", func() {
			Expect(repo.SaveForChannel("pc-1", []model.PodcastPodrollItem{
				{FeedURL: "https://example.com/feed.xml"},
			})).To(Succeed())

			result, err := repo.GetByChannel("pc-1")
			Expect(err).ToNot(HaveOccurred())
			Expect(result[0].ID).ToNot(BeEmpty())

			Expect(repo.SaveForChannel("pc-1", nil)).To(Succeed())
		})
	})

	Describe("GetByChannels — bulk query", func() {
		It("returns items for multiple channels", func() {
			Expect(repo.SaveForChannel("pc-1", []model.PodcastPodrollItem{
				{FeedURL: "https://a.example.com/feed.xml", Title: "Feed A"},
			})).To(Succeed())
			Expect(repo.SaveForChannel("pc-2", []model.PodcastPodrollItem{
				{FeedURL: "https://b.example.com/feed.xml", Title: "Feed B"},
			})).To(Succeed())

			result, err := repo.GetByChannels([]string{"pc-1", "pc-2"})
			Expect(err).ToNot(HaveOccurred())
			Expect(result).To(HaveLen(2))
			titles := []string{result[0].Title, result[1].Title}
			Expect(titles).To(ConsistOf("Feed A", "Feed B"))

			Expect(repo.SaveForChannel("pc-1", nil)).To(Succeed())
			Expect(repo.SaveForChannel("pc-2", nil)).To(Succeed())
		})

		It("returns nil for empty id slice", func() {
			result, err := repo.GetByChannels([]string{})
			Expect(err).ToNot(HaveOccurred())
			Expect(result).To(BeNil())
		})
	})
})
