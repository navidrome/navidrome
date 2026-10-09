package persistence

import (
	"context"
	"time"

	"github.com/navidrome/navidrome/log"
	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/model/request"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("PodcastEpisodeRepository", func() {
	var ctx context.Context
	var repo model.PodcastEpisodeRepository

	BeforeEach(func() {
		ctx = log.NewContext(context.TODO())
		ctx = request.WithUser(ctx, adminUser)
		repo = NewPodcastEpisodeRepository(GetDBXBuilder())
	})

	Describe("Get", func() {
		It("returns an existing episode", func() {
			ep, err := repo.Get(ctx, "pe-1")
			Expect(err).ToNot(HaveOccurred())
			Expect(ep.Title).To(Equal("Episode 1"))
			Expect(ep.ChannelID).To(Equal("pc-1"))
		})

		It("returns ErrNotFound for unknown id", func() {
			_, err := repo.Get(ctx, "no-such-id")
			Expect(err).To(MatchError(model.ErrNotFound))
		})
	})

	Describe("GetNewest", func() {
		It("returns episodes ordered by publish_date DESC", func() {
			eps, err := repo.GetNewest(ctx, 10)
			Expect(err).ToNot(HaveOccurred())
			Expect(len(eps)).To(BeNumerically(">=", 2))
			Expect(eps[0].PublishDate.After(eps[1].PublishDate)).To(BeTrue())
		})

		It("respects the count limit", func() {
			eps, err := repo.GetNewest(ctx, 1)
			Expect(err).ToNot(HaveOccurred())
			Expect(eps).To(HaveLen(1))
		})
	})

	Describe("GetByChannel", func() {
		It("returns only episodes belonging to the channel", func() {
			eps, err := repo.GetByChannel(ctx, "pc-1")
			Expect(err).ToNot(HaveOccurred())
			Expect(eps).To(HaveLen(2))
			for _, ep := range eps {
				Expect(ep.ChannelID).To(Equal("pc-1"))
			}
		})

		It("returns empty slice for channel with no episodes", func() {
			eps, err := repo.GetByChannel(ctx, "pc-2")
			Expect(err).ToNot(HaveOccurred())
			Expect(eps).To(BeEmpty())
		})
	})

	Describe("GetByGUID", func() {
		It("returns the episode matching channel+guid", func() {
			ep, err := repo.GetByGUID(ctx, "pc-1", "guid-001")
			Expect(err).ToNot(HaveOccurred())
			Expect(ep.Title).To(Equal("Episode 1"))
		})

		It("returns ErrNotFound for unknown guid", func() {
			_, err := repo.GetByGUID(ctx, "pc-1", "no-such-guid")
			Expect(err).To(MatchError(model.ErrNotFound))
		})

		It("returns ErrNotFound when channel does not match", func() {
			_, err := repo.GetByGUID(ctx, "pc-2", "guid-001")
			Expect(err).To(MatchError(model.ErrNotFound))
		})
	})

	Describe("Create and Delete", func() {
		It("creates an episode and hard deletes it", func() {
			ep := &model.PodcastEpisode{
				ChannelID:   "pc-1",
				GUID:        "guid-temp",
				Title:       "Temp Episode",
				Status:      model.PodcastStatusNew,
				PublishDate: time.Now(),
			}
			err := repo.Create(ctx, ep)
			Expect(err).ToNot(HaveOccurred())
			Expect(ep.ID).ToNot(BeEmpty())

			err = repo.Delete(ctx, ep.ID)
			Expect(err).ToNot(HaveOccurred())

			_, err = repo.Get(ctx, ep.ID)
			Expect(err).To(MatchError(model.ErrNotFound))
		})
	})

	Describe("Update", func() {
		It("updates episode fields", func() {
			ep := &model.PodcastEpisode{
				ChannelID:   "pc-1",
				GUID:        "guid-upd",
				Title:       "Before",
				Status:      model.PodcastStatusNew,
				PublishDate: time.Now(),
			}
			_ = repo.Create(ctx, ep)

			ep.Status = model.PodcastStatusCompleted
			ep.Path = "/podcasts/pc-1/ep.mp3"
			err := repo.Update(ctx, ep)
			Expect(err).ToNot(HaveOccurred())

			saved, _ := repo.Get(ctx, ep.ID)
			Expect(saved.Status).To(Equal(model.PodcastStatusCompleted))
			Expect(saved.Path).To(Equal("/podcasts/pc-1/ep.mp3"))

			// cleanup
			_ = repo.Delete(ctx, ep.ID)
		})
	})
})
