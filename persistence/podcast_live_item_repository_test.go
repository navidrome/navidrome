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

var _ = Describe("PodcastLiveItemRepository", func() {
	var repo model.PodcastLiveItemRepository

	BeforeEach(func() {
		ctx := log.NewContext(context.TODO())
		ctx = request.WithUser(ctx, adminUser)
		repo = NewPodcastLiveItemRepository(ctx, GetDBXBuilder())
	})

	Describe("Upsert and GetByChannel", func() {
		It("creates a new live item when none exists", func() {
			item := &model.PodcastLiveItem{
				ChannelID:    "pc-1",
				GUID:         "live-guid-001",
				Title:        "Live Show",
				Status:       "live",
				StartTime:    time.Date(2024, 4, 27, 8, 0, 0, 0, time.UTC),
				EndTime:      time.Date(2024, 4, 27, 9, 0, 0, 0, time.UTC),
				EnclosureURL: "https://stream.example.com/live.m3u8",
				EnclosureType: "application/x-mpegURL",
				ContentLinkURL:  "https://youtube.com/live",
				ContentLinkText: "Watch Live",
			}
			Expect(repo.Upsert(item)).To(Succeed())

			result, err := repo.GetByChannel("pc-1")
			Expect(err).ToNot(HaveOccurred())
			Expect(result).ToNot(BeNil())
			Expect(result.ChannelID).To(Equal("pc-1"))
			Expect(result.GUID).To(Equal("live-guid-001"))
			Expect(result.Title).To(Equal("Live Show"))
			Expect(result.Status).To(Equal("live"))
			Expect(result.EnclosureURL).To(Equal("https://stream.example.com/live.m3u8"))
			Expect(result.ContentLinkURL).To(Equal("https://youtube.com/live"))
			Expect(result.ContentLinkText).To(Equal("Watch Live"))

			Expect(repo.DeleteByChannel("pc-1")).To(Succeed())
		})

		It("assigns ID and timestamps automatically on create", func() {
			item := &model.PodcastLiveItem{
				ChannelID: "pc-1",
				Status:    "live",
			}
			Expect(repo.Upsert(item)).To(Succeed())

			result, err := repo.GetByChannel("pc-1")
			Expect(err).ToNot(HaveOccurred())
			Expect(result.ID).ToNot(BeEmpty())
			Expect(result.CreatedAt.IsZero()).To(BeFalse())
			Expect(result.UpdatedAt.IsZero()).To(BeFalse())

			Expect(repo.DeleteByChannel("pc-1")).To(Succeed())
		})

		It("updates existing live item (latest wins)", func() {
			first := &model.PodcastLiveItem{
				ChannelID: "pc-1",
				GUID:      "live-guid-001",
				Title:     "Original Title",
				Status:    "pending",
			}
			Expect(repo.Upsert(first)).To(Succeed())

			second := &model.PodcastLiveItem{
				ChannelID: "pc-1",
				GUID:      "live-guid-001",
				Title:     "Updated Title",
				Status:    "live",
			}
			Expect(repo.Upsert(second)).To(Succeed())

			result, err := repo.GetByChannel("pc-1")
			Expect(err).ToNot(HaveOccurred())
			Expect(result.Title).To(Equal("Updated Title"))
			Expect(result.Status).To(Equal("live"))

			Expect(repo.DeleteByChannel("pc-1")).To(Succeed())
		})

		It("preserves created_at on update", func() {
			item := &model.PodcastLiveItem{
				ChannelID: "pc-1",
				Status:    "pending",
			}
			Expect(repo.Upsert(item)).To(Succeed())

			original, _ := repo.GetByChannel("pc-1")
			originalCreatedAt := original.CreatedAt

			item2 := &model.PodcastLiveItem{
				ChannelID: "pc-1",
				Status:    "live",
			}
			Expect(repo.Upsert(item2)).To(Succeed())

			updated, _ := repo.GetByChannel("pc-1")
			Expect(updated.CreatedAt.UTC().Truncate(time.Second)).
				To(Equal(originalCreatedAt.UTC().Truncate(time.Second)))

			Expect(repo.DeleteByChannel("pc-1")).To(Succeed())
		})

		It("returns ErrNotFound for unknown channel", func() {
			_, err := repo.GetByChannel("no-such-channel")
			Expect(err).To(Equal(model.ErrNotFound))
		})

		It("handles zero-value start/end times", func() {
			item := &model.PodcastLiveItem{
				ChannelID: "pc-1",
				Status:    "live",
			}
			Expect(repo.Upsert(item)).To(Succeed())

			result, err := repo.GetByChannel("pc-1")
			Expect(err).ToNot(HaveOccurred())
			Expect(result).ToNot(BeNil())

			Expect(repo.DeleteByChannel("pc-1")).To(Succeed())
		})
	})

	Describe("DeleteByChannel", func() {
		It("removes live item for the given channel", func() {
			Expect(repo.Upsert(&model.PodcastLiveItem{ChannelID: "pc-1", Status: "live"})).To(Succeed())
			Expect(repo.DeleteByChannel("pc-1")).To(Succeed())

			_, err := repo.GetByChannel("pc-1")
			Expect(err).To(Equal(model.ErrNotFound))
		})

		It("does not error when no item exists", func() {
			Expect(repo.DeleteByChannel("no-such-channel")).To(Succeed())
		})
	})
})
