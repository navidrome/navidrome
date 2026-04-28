package persistence

import (
	"context"

	"github.com/navidrome/navidrome/log"
	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/model/request"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("PodcastFundingRepository", func() {
	var repo model.PodcastFundingRepository

	BeforeEach(func() {
		ctx := log.NewContext(context.TODO())
		ctx = request.WithUser(ctx, adminUser)
		repo = NewPodcastFundingRepository(ctx, GetDBXBuilder())
	})

	Describe("SaveForChannel and GetByChannel", func() {
		It("saves and retrieves funding items", func() {
			items := []model.PodcastFundingItem{
				{URL: "https://patreon.com/show", Text: "Support on Patreon"},
				{URL: "https://ko-fi.com/show", Text: "Buy me a coffee"},
			}
			Expect(repo.SaveForChannel("pc-1", items)).To(Succeed())

			result, err := repo.GetByChannel("pc-1")
			Expect(err).ToNot(HaveOccurred())
			Expect(result).To(HaveLen(2))

			Expect(repo.SaveForChannel("pc-1", nil)).To(Succeed())
		})

		It("preserves sort_order in insertion order", func() {
			items := []model.PodcastFundingItem{
				{URL: "https://first.example.com", Text: "First"},
				{URL: "https://second.example.com", Text: "Second"},
				{URL: "https://third.example.com", Text: "Third"},
			}
			Expect(repo.SaveForChannel("pc-1", items)).To(Succeed())

			result, err := repo.GetByChannel("pc-1")
			Expect(err).ToNot(HaveOccurred())
			Expect(result).To(HaveLen(3))
			Expect(result[0].Text).To(Equal("First"))
			Expect(result[1].Text).To(Equal("Second"))
			Expect(result[2].Text).To(Equal("Third"))

			Expect(repo.SaveForChannel("pc-1", nil)).To(Succeed())
		})

		It("replaces existing items on re-save", func() {
			Expect(repo.SaveForChannel("pc-1", []model.PodcastFundingItem{
				{URL: "https://old.example.com", Text: "Old"},
			})).To(Succeed())
			Expect(repo.SaveForChannel("pc-1", []model.PodcastFundingItem{
				{URL: "https://new.example.com", Text: "New"},
			})).To(Succeed())

			result, err := repo.GetByChannel("pc-1")
			Expect(err).ToNot(HaveOccurred())
			Expect(result).To(HaveLen(1))
			Expect(result[0].Text).To(Equal("New"))

			Expect(repo.SaveForChannel("pc-1", nil)).To(Succeed())
		})

		It("clears items when saved with nil slice", func() {
			Expect(repo.SaveForChannel("pc-1", []model.PodcastFundingItem{
				{URL: "https://example.com", Text: "Something"},
			})).To(Succeed())
			Expect(repo.SaveForChannel("pc-1", nil)).To(Succeed())

			result, err := repo.GetByChannel("pc-1")
			Expect(err).ToNot(HaveOccurred())
			Expect(result).To(BeEmpty())
		})

		It("does not affect items of other channels", func() {
			Expect(repo.SaveForChannel("pc-1", []model.PodcastFundingItem{
				{URL: "https://a.example.com", Text: "Channel A"},
			})).To(Succeed())
			Expect(repo.SaveForChannel("pc-2", []model.PodcastFundingItem{
				{URL: "https://b.example.com", Text: "Channel B"},
			})).To(Succeed())

			resultA, _ := repo.GetByChannel("pc-1")
			resultB, _ := repo.GetByChannel("pc-2")
			Expect(resultA).To(HaveLen(1))
			Expect(resultB).To(HaveLen(1))
			Expect(resultA[0].Text).To(Equal("Channel A"))
			Expect(resultB[0].Text).To(Equal("Channel B"))

			Expect(repo.SaveForChannel("pc-1", nil)).To(Succeed())
			Expect(repo.SaveForChannel("pc-2", nil)).To(Succeed())
		})

		It("returns empty list for unknown channel", func() {
			result, err := repo.GetByChannel("no-such-channel")
			Expect(err).ToNot(HaveOccurred())
			Expect(result).To(BeEmpty())
		})

		It("assigns non-empty ID automatically", func() {
			Expect(repo.SaveForChannel("pc-1", []model.PodcastFundingItem{
				{URL: "https://example.com", Text: "Test"},
			})).To(Succeed())

			result, err := repo.GetByChannel("pc-1")
			Expect(err).ToNot(HaveOccurred())
			Expect(result[0].ID).ToNot(BeEmpty())

			Expect(repo.SaveForChannel("pc-1", nil)).To(Succeed())
		})

		It("sets ChannelID on returned items", func() {
			Expect(repo.SaveForChannel("pc-1", []model.PodcastFundingItem{
				{URL: "https://example.com", Text: "Test"},
			})).To(Succeed())

			result, _ := repo.GetByChannel("pc-1")
			Expect(result[0].ChannelID).To(Equal("pc-1"))

			Expect(repo.SaveForChannel("pc-1", nil)).To(Succeed())
		})
	})

	Describe("GetByChannels — bulk query", func() {
		It("returns items for multiple channels", func() {
			Expect(repo.SaveForChannel("pc-1", []model.PodcastFundingItem{
				{URL: "https://a.example.com", Text: "Feed A"},
			})).To(Succeed())
			Expect(repo.SaveForChannel("pc-2", []model.PodcastFundingItem{
				{URL: "https://b.example.com", Text: "Feed B"},
			})).To(Succeed())

			result, err := repo.GetByChannels([]string{"pc-1", "pc-2"})
			Expect(err).ToNot(HaveOccurred())
			Expect(result).To(HaveLen(2))
			texts := []string{result[0].Text, result[1].Text}
			Expect(texts).To(ConsistOf("Feed A", "Feed B"))

			Expect(repo.SaveForChannel("pc-1", nil)).To(Succeed())
			Expect(repo.SaveForChannel("pc-2", nil)).To(Succeed())
		})

		It("returned items carry ChannelID for grouping", func() {
			Expect(repo.SaveForChannel("pc-1", []model.PodcastFundingItem{
				{URL: "https://a.example.com", Text: "A"},
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
})
