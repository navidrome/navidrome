package persistence

import (
	"context"

	"github.com/deluan/rest"
	"github.com/navidrome/navidrome/log"
	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/model/request"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("PodcastChannelRepository", func() {
	var repo model.PodcastChannelRepository
	var adminCtx, userCtx context.Context

	BeforeEach(func() {
		ctx := log.NewContext(context.TODO())
		adminCtx = request.WithUser(ctx, adminUser)
		userCtx = request.WithUser(ctx, regularUser)
		repo = NewPodcastChannelRepository(GetDBXBuilder())
	})

	Describe("Get", func() {
		It("returns an existing channel", func() {
			ch, err := repo.Get(adminCtx, "pc-1")
			Expect(err).ToNot(HaveOccurred())
			Expect(ch.ID).To(Equal("pc-1"))
			Expect(ch.Title).To(Equal("Test Podcast"))
		})

		It("returns ErrNotFound for unknown id", func() {
			_, err := repo.Get(adminCtx, "no-such-id")
			Expect(err).To(MatchError(model.ErrNotFound))
		})
	})

	Describe("GetAll", func() {
		It("returns all channels without episodes", func() {
			channels, err := repo.GetAll(adminCtx, false)
			Expect(err).ToNot(HaveOccurred())
			Expect(len(channels)).To(BeNumerically(">=", 2))
			for _, ch := range channels {
				Expect(ch.Episodes).To(BeEmpty())
			}
		})

		It("returns channels with episodes when withEpisodes=true", func() {
			channels, err := repo.GetAll(adminCtx, true)
			Expect(err).ToNot(HaveOccurred())
			var ch1 *model.PodcastChannel
			for i := range channels {
				if channels[i].ID == "pc-1" {
					ch1 = &channels[i]
					break
				}
			}
			Expect(ch1).ToNot(BeNil())
			Expect(ch1.Episodes).To(HaveLen(2))
		})
	})

	Describe("Create", func() {
		It("creates a new channel and assigns an ID", func() {
			ch := &model.PodcastChannel{
				URL:    "https://new.example.com/feed.xml",
				Title:  "New Podcast",
				Status: model.PodcastStatusNew,
			}
			err := repo.Create(adminCtx, ch)
			Expect(err).ToNot(HaveOccurred())
			Expect(ch.ID).ToNot(BeEmpty())

			saved, err := repo.Get(adminCtx, ch.ID)
			Expect(err).ToNot(HaveOccurred())
			Expect(saved.Title).To(Equal("New Podcast"))

			// cleanup
			_ = repo.Delete(adminCtx, ch.ID)
		})

		It("denies non-admin users", func() {
			err := repo.Create(userCtx, &model.PodcastChannel{URL: "https://x.com/feed.xml"})
			Expect(err).To(MatchError(rest.ErrPermissionDenied))
		})
	})

	Describe("Update", func() {
		It("updates an existing channel", func() {
			ch := &model.PodcastChannel{
				URL:    "https://update.example.com/feed.xml",
				Title:  "Before Update",
				Status: model.PodcastStatusNew,
			}
			_ = repo.Create(adminCtx, ch)

			ch.Title = "After Update"
			err := repo.UpdateChannel(adminCtx, ch)
			Expect(err).ToNot(HaveOccurred())

			saved, _ := repo.Get(adminCtx, ch.ID)
			Expect(saved.Title).To(Equal("After Update"))

			// cleanup
			_ = repo.Delete(adminCtx, ch.ID)
		})
	})

	Describe("Delete", func() {
		It("deletes an existing channel", func() {
			ch := &model.PodcastChannel{URL: "https://del.example.com/feed.xml", Status: model.PodcastStatusNew}
			_ = repo.Create(adminCtx, ch)

			err := repo.Delete(adminCtx, ch.ID)
			Expect(err).ToNot(HaveOccurred())

			_, err = repo.Get(adminCtx, ch.ID)
			Expect(err).To(MatchError(model.ErrNotFound))
		})

		It("denies non-admin users", func() {
			err := repo.Delete(userCtx, "pc-1")
			Expect(err).To(MatchError(rest.ErrPermissionDenied))
		})
	})

	Describe("Regular user read access", func() {
		It("allows regular users to read channels", func() {
			channels, err := repo.GetAll(userCtx, false)
			Expect(err).ToNot(HaveOccurred())
			Expect(channels).ToNot(BeEmpty())
		})
	})
})
