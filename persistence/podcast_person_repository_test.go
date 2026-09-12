package persistence

import (
	"context"

	"github.com/navidrome/navidrome/log"
	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/model/request"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("PodcastPersonRepository", func() {
	var repo model.PodcastPersonRepository

	BeforeEach(func() {
		ctx := log.NewContext(context.TODO())
		ctx = request.WithUser(ctx, adminUser)
		repo = NewPodcastPersonRepository(ctx, GetDBXBuilder())
	})

	Describe("SaveForChannel and GetByChannel", func() {
		It("saves and retrieves channel-level persons", func() {
			persons := []model.PodcastPerson{
				{Name: "Jane Host", Role: "host", Group: "cast", Img: "https://example.com/jane.jpg", Href: "https://example.com/jane"},
				{Name: "Bob Producer", Role: "producer", Group: "crew"},
			}
			Expect(repo.SaveForChannel("pc-1", persons)).To(Succeed())

			result, err := repo.GetByChannel("pc-1")
			Expect(err).ToNot(HaveOccurred())
			Expect(result).To(HaveLen(2))

			var names []string
			for _, p := range result {
				names = append(names, p.Name)
			}
			Expect(names).To(ConsistOf("Jane Host", "Bob Producer"))

			Expect(repo.SaveForChannel("pc-1", nil)).To(Succeed())
		})

		It("persists all person fields", func() {
			persons := []model.PodcastPerson{
				{Name: "Jane Host", Role: "host", Group: "cast", Img: "https://example.com/jane.jpg", Href: "https://example.com/jane"},
			}
			Expect(repo.SaveForChannel("pc-1", persons)).To(Succeed())

			result, err := repo.GetByChannel("pc-1")
			Expect(err).ToNot(HaveOccurred())
			Expect(result[0].Name).To(Equal("Jane Host"))
			Expect(result[0].Role).To(Equal("host"))
			Expect(result[0].Group).To(Equal("cast"))
			Expect(result[0].Img).To(Equal("https://example.com/jane.jpg"))
			Expect(result[0].Href).To(Equal("https://example.com/jane"))

			Expect(repo.SaveForChannel("pc-1", nil)).To(Succeed())
		})

		It("replaces existing persons on re-save", func() {
			Expect(repo.SaveForChannel("pc-1", []model.PodcastPerson{{Name: "Old Host", Role: "host", Group: "cast"}})).To(Succeed())
			Expect(repo.SaveForChannel("pc-1", []model.PodcastPerson{{Name: "New Host", Role: "host", Group: "cast"}})).To(Succeed())

			result, err := repo.GetByChannel("pc-1")
			Expect(err).ToNot(HaveOccurred())
			Expect(result).To(HaveLen(1))
			Expect(result[0].Name).To(Equal("New Host"))

			Expect(repo.SaveForChannel("pc-1", nil)).To(Succeed())
		})

		It("clears persons when saved with nil slice", func() {
			Expect(repo.SaveForChannel("pc-1", []model.PodcastPerson{{Name: "Jane Host", Role: "host", Group: "cast"}})).To(Succeed())
			Expect(repo.SaveForChannel("pc-1", nil)).To(Succeed())

			result, err := repo.GetByChannel("pc-1")
			Expect(err).ToNot(HaveOccurred())
			Expect(result).To(BeEmpty())
		})

		It("does not affect persons of other channels", func() {
			Expect(repo.SaveForChannel("pc-1", []model.PodcastPerson{{Name: "Host A", Role: "host", Group: "cast"}})).To(Succeed())
			Expect(repo.SaveForChannel("pc-2", []model.PodcastPerson{{Name: "Host B", Role: "host", Group: "cast"}})).To(Succeed())

			resultA, _ := repo.GetByChannel("pc-1")
			resultB, _ := repo.GetByChannel("pc-2")
			Expect(resultA).To(HaveLen(1))
			Expect(resultB).To(HaveLen(1))
			Expect(resultA[0].Name).To(Equal("Host A"))
			Expect(resultB[0].Name).To(Equal("Host B"))

			Expect(repo.SaveForChannel("pc-1", nil)).To(Succeed())
			Expect(repo.SaveForChannel("pc-2", nil)).To(Succeed())
		})

		It("returns empty list for unknown channel", func() {
			result, err := repo.GetByChannel("no-such-channel")
			Expect(err).ToNot(HaveOccurred())
			Expect(result).To(BeEmpty())
		})
	})

	Describe("SaveForEpisode and GetByEpisode", func() {
		It("saves and retrieves episode-level persons", func() {
			persons := []model.PodcastPerson{
				{Name: "John Guest", Role: "guest", Group: "cast"},
			}
			Expect(repo.SaveForEpisode("pe-1", persons)).To(Succeed())

			result, err := repo.GetByEpisode("pe-1")
			Expect(err).ToNot(HaveOccurred())
			Expect(result).To(HaveLen(1))
			Expect(result[0].Name).To(Equal("John Guest"))
			Expect(result[0].Role).To(Equal("guest"))

			Expect(repo.SaveForEpisode("pe-1", nil)).To(Succeed())
		})

		It("replaces existing episode persons on re-save", func() {
			Expect(repo.SaveForEpisode("pe-1", []model.PodcastPerson{{Name: "Old Guest", Role: "guest", Group: "cast"}})).To(Succeed())
			Expect(repo.SaveForEpisode("pe-1", []model.PodcastPerson{{Name: "New Guest", Role: "guest", Group: "cast"}})).To(Succeed())

			result, err := repo.GetByEpisode("pe-1")
			Expect(err).ToNot(HaveOccurred())
			Expect(result).To(HaveLen(1))
			Expect(result[0].Name).To(Equal("New Guest"))

			Expect(repo.SaveForEpisode("pe-1", nil)).To(Succeed())
		})

		It("returns empty list for unknown episode", func() {
			result, err := repo.GetByEpisode("no-such-episode")
			Expect(err).ToNot(HaveOccurred())
			Expect(result).To(BeEmpty())
		})
	})

	Describe("GetByEpisodes — bulk query", func() {
		It("returns persons for multiple episodes in one query", func() {
			Expect(repo.SaveForEpisode("pe-1", []model.PodcastPerson{{Name: "Guest A", Role: "guest", Group: "cast"}})).To(Succeed())
			Expect(repo.SaveForEpisode("pe-2", []model.PodcastPerson{{Name: "Guest B", Role: "guest", Group: "cast"}})).To(Succeed())

			result, err := repo.GetByEpisodes([]string{"pe-1", "pe-2"})
			Expect(err).ToNot(HaveOccurred())
			Expect(result).To(HaveLen(2))

			names := []string{result[0].Name, result[1].Name}
			Expect(names).To(ConsistOf("Guest A", "Guest B"))

			Expect(repo.SaveForEpisode("pe-1", nil)).To(Succeed())
			Expect(repo.SaveForEpisode("pe-2", nil)).To(Succeed())
		})

		It("returns empty list for empty id slice", func() {
			result, err := repo.GetByEpisodes([]string{})
			Expect(err).ToNot(HaveOccurred())
			Expect(result).To(BeEmpty())
		})
	})

	Describe("auto ID generation", func() {
		It("assigns an ID automatically on SaveForChannel", func() {
			Expect(repo.SaveForChannel("pc-1", []model.PodcastPerson{{Name: "Jane", Role: "host", Group: "cast"}})).To(Succeed())

			result, err := repo.GetByChannel("pc-1")
			Expect(err).ToNot(HaveOccurred())
			Expect(result[0].ID).ToNot(BeEmpty())

			Expect(repo.SaveForChannel("pc-1", nil)).To(Succeed())
		})
	})
})
