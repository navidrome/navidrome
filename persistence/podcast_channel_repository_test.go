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
	var adminRepo model.PodcastChannelRepository
	var userRepo model.PodcastChannelRepository

	BeforeEach(func() {
		ctx := log.NewContext(context.TODO())
		adminCtx := request.WithUser(ctx, adminUser)
		userCtx := request.WithUser(ctx, regularUser)
		adminRepo = NewPodcastChannelRepository(adminCtx, GetDBXBuilder())
		userRepo = NewPodcastChannelRepository(userCtx, GetDBXBuilder())
	})

	Describe("Get", func() {
		It("returns an existing channel", func() {
			ch, err := adminRepo.Get("pc-1")
			Expect(err).ToNot(HaveOccurred())
			Expect(ch.ID).To(Equal("pc-1"))
			Expect(ch.Title).To(Equal("Test Podcast"))
		})

		It("returns ErrNotFound for unknown id", func() {
			_, err := adminRepo.Get("no-such-id")
			Expect(err).To(MatchError(model.ErrNotFound))
		})
	})

	Describe("GetAll", func() {
		It("returns all channels without episodes", func() {
			channels, err := adminRepo.GetAll(false)
			Expect(err).ToNot(HaveOccurred())
			Expect(len(channels)).To(BeNumerically(">=", 2))
			for _, ch := range channels {
				Expect(ch.Episodes).To(BeEmpty())
			}
		})

		It("returns channels with episodes when withEpisodes=true", func() {
			channels, err := adminRepo.GetAll(true)
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
			err := adminRepo.Create(ch)
			Expect(err).ToNot(HaveOccurred())
			Expect(ch.ID).ToNot(BeEmpty())

			saved, err := adminRepo.Get(ch.ID)
			Expect(err).ToNot(HaveOccurred())
			Expect(saved.Title).To(Equal("New Podcast"))

			// cleanup
			_ = adminRepo.Delete(ch.ID)
		})

		It("denies non-admin users", func() {
			err := userRepo.Create(&model.PodcastChannel{URL: "https://x.com/feed.xml"})
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
			_ = adminRepo.Create(ch)

			ch.Title = "After Update"
			err := adminRepo.UpdateChannel(ch)
			Expect(err).ToNot(HaveOccurred())

			saved, _ := adminRepo.Get(ch.ID)
			Expect(saved.Title).To(Equal("After Update"))

			// cleanup
			_ = adminRepo.Delete(ch.ID)
		})
	})

	Describe("Delete", func() {
		It("deletes an existing channel", func() {
			ch := &model.PodcastChannel{URL: "https://del.example.com/feed.xml", Status: model.PodcastStatusNew}
			_ = adminRepo.Create(ch)

			err := adminRepo.Delete(ch.ID)
			Expect(err).ToNot(HaveOccurred())

			_, err = adminRepo.Get(ch.ID)
			Expect(err).To(MatchError(model.ErrNotFound))
		})

		It("denies non-admin users", func() {
			err := userRepo.Delete("pc-1")
			Expect(err).To(MatchError(rest.ErrPermissionDenied))
		})
	})

	Describe("Regular user read access", func() {
		It("allows regular users to read channels", func() {
			channels, err := userRepo.GetAll(false)
			Expect(err).ToNot(HaveOccurred())
			Expect(channels).ToNot(BeEmpty())
		})
	})
})
