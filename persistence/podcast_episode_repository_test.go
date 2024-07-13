package persistence

import (
	"context"

	"github.com/navidrome/navidrome/db"
	"github.com/navidrome/navidrome/log"
	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/model/request"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("PodcastEpisodeRepository", func() {
	var repo model.PodcastEpisodeRepository

	BeforeEach(func() {
		db := NewDBXBuilder(db.Db())

		ctx := request.WithUser(log.NewContext(context.TODO()), model.User{ID: "userid", UserName: "johndoe", IsAdmin: false})
		repo = NewPodcastEpisodeRepository(ctx, db)

		for i := range testPodcastEpisodes {
			err := repo.Put(&testPodcastEpisodes[i])
			if err != nil {
				panic(err)
			}
		}

		if err := repo.SetStar(true, completeEpisode.ID); err != nil {
			panic(err)
		}

		podcast, _ := repo.Get(completeEpisode.ID)
		completeEpisode.Starred = true
		completeEpisode.StarredAt = podcast.StarredAt
		testPodcastEpisodes[1] = completeEpisode
	})

	AfterEach(func() {
		all, err := repo.GetAll()
		if err != nil {
			panic(err)
		}

		for _, episode := range all {
			err = repo.Delete(episode.ID)
			if err != nil {
				panic(err)
			}
		}

		for i := range testPodcastEpisodes {
			r := testPodcastEpisodes[i]
			if err = repo.Put(&r); err != nil {
				panic(err)
			}
		}

		if err = repo.SetStar(false, completeEpisode.ID); err != nil {
			panic(err)
		}

		completeEpisode.Starred = false
		completeEpisode.StarredAt = nil
		testPodcastEpisodes[1] = completeEpisode
	})

	Describe("Count", func() {
		It("gets count", func() {
			Expect(repo.Count()).To(Equal(int64(3)))
		})
	})

	Describe("CountAll", func() {
		It("Returns the number of podcasts in the DB", func() {
			Expect(repo.CountAll()).To(Equal(int64(3)))
		})
	})

	Describe("Delete", func() {
		It("deletes existing item", func() {
			err := repo.Delete(basicEpisode.ID)
			Expect(err).To(BeNil())

			_, err = repo.Get(basicEpisode.ID)
			Expect(err).To(MatchError(model.ErrNotFound))
		})
	})

	Describe("EntityName", func() {
		It("should return the right value", func() {
			Expect(repo.EntityName()).To(Equal("podcast_episode"))
		})
	})

	Describe("Get", func() {
		DescribeTable("GetExisting", func(episode *model.PodcastEpisode) {
			retrieved, err := repo.Get(episode.ID)
			Expect(err).To(BeNil())

			retrieved.UpdatedAt = episode.UpdatedAt
			Expect(retrieved).To(BeComparableTo(episode))
		},
			Entry("simple episode", &basicEpisode),
			Entry("complete podcast", &completeEpisode),
			Entry("broken episode", &brokenEpisode))

		It("returns ErrNotFound when the podcast does not exist", func() {
			_, err := repo.Get("nonexistent")
			Expect(err).To(MatchError(model.ErrNotFound))
		})
	})

	Describe("GetAll", func() {
		It("Gets all items", func() {
			episodes, err := repo.GetAll()
			Expect(err).To(BeNil())
			for i := range episodes {
				episodes[i].UpdatedAt = testPodcastEpisodes[i].UpdatedAt
			}
			Expect(episodes).To(BeComparableTo(testPodcastEpisodes))
		})
	})

	Describe("GetEpisodeGuids", func() {
		It("returns all guids", func() {
			guids, err := repo.GetEpisodeGuids("1234")
			Expect(err).To(BeNil())
			Expect(guids).To(Equal(map[string]bool{"1": true, "3": true}))
		})
	})

	Describe("GetNewestEpisodes", func() {
		It("Gets newest episodes", func() {
			eps, err := repo.GetNewestEpisodes(2)
			Expect(err).To(BeNil())

			eps[0].UpdatedAt = completeEpisode.UpdatedAt
			eps[1].UpdatedAt = basicEpisode.UpdatedAt
			Expect(eps).To(BeComparableTo(model.PodcastEpisodes{completeEpisode, basicEpisode}))

		})
	})

	Describe("Put", func() {
		It("Successfully updates item", func() {
			episode := completeEpisode
			episode.Annotations = model.Annotations{}
			episode.ID = brokenEpisode.ID

			err := repo.Put(&episode)
			Expect(err).To(BeNil())

			item, err := repo.Get(episode.ID)
			Expect(err).To(BeNil())
			item.UpdatedAt = episode.UpdatedAt
			Expect(*item).To(Equal(episode))
		})

		It("successfully creates an item", func() {
			episode := model.PodcastEpisode{
				PodcastId: simplePodcast.ID,
				Url:       "https://example.org/feed.mp3",
			}

			err := repo.Put(&episode)
			Expect(episode.ID).ToNot(BeEmpty())

			Expect(err).To(BeNil())
			Expect(repo.CountAll()).To(Equal(int64(4)))

			item, err := repo.Get(episode.ID)
			Expect(err).To(BeNil())

			Expect(item.CreatedAt).ToNot(Equal(episode.CreatedAt))
			Expect(item.UpdatedAt).ToNot(Equal(episode.UpdatedAt))
			item.CreatedAt = episode.CreatedAt
			item.UpdatedAt = episode.UpdatedAt

			Expect(*item).To(BeComparableTo(episode))
		})
	})
})
