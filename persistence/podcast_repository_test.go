package persistence

import (
	"context"
	"time"

	"github.com/deluan/rest"
	"github.com/navidrome/navidrome/conf"
	"github.com/navidrome/navidrome/conf/configtest"
	"github.com/navidrome/navidrome/db"
	"github.com/navidrome/navidrome/log"
	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/model/request"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("PodcastRepository", func() {
	var adminRepo model.PodcastRepository
	var repo model.PodcastRepository
	var database *dbxBuilder
	var repos []model.PodcastRepository

	BeforeEach(func() {
		database = NewDBXBuilder(db.Db())

		adminCtx := request.WithUser(log.NewContext(context.TODO()), model.User{ID: "userid", UserName: "johndoe", IsAdmin: true})
		adminRepo = NewPodcastRepository(adminCtx, database)

		ctx := request.WithUser(log.NewContext(context.TODO()), model.User{ID: "userid", UserName: "johndoe", IsAdmin: false})
		repo = NewPodcastRepository(ctx, database)

		repos = []model.PodcastRepository{adminRepo, repo}

		if err := adminRepo.SetStar(true, fullPodcast.ID); err != nil {
			panic(err)
		}

		podcast, _ := adminRepo.Get(fullPodcast.ID, false)
		fullPodcast.Starred = true
		fullPodcast.StarredAt = podcast.StarredAt
		testPodcasts[1] = fullPodcast

		DeferCleanup(configtest.SetupConfig())
	})

	AfterEach(func() {
		all, _ := adminRepo.GetAll(false)

		for _, podcast := range all {
			_ = adminRepo.Delete(podcast.ID)
		}

		for i := range testPodcasts {
			r := testPodcasts[i]
			err := adminRepo.Put(&r)
			if err != nil {
				panic(err)
			}
		}

		fullPodcast.Starred = false
		fullPodcast.StarredAt = nil
		testPodcasts[1] = fullPodcast
	})

	Describe("Count", func() {
		It("gets count", func() {
			Expect(adminRepo.Count()).To(Equal(int64(3)))
			Expect(repo.Count()).To(Equal(int64(3)))
		})
	})

	Describe("CountAll", func() {
		It("Returns the number of podcasts in the DB", func() {
			Expect(adminRepo.CountAll()).To(Equal(int64(3)))
			Expect(repo.CountAll()).To(Equal(int64(3)))
		})
	})

	Describe("Delete", func() {
		It("deletes existing item", func() {
			err := adminRepo.Delete(simplePodcast.ID)
			Expect(err).To(BeNil())

			_, err = adminRepo.Get(simplePodcast.ID, false)
			Expect(err).To(MatchError(model.ErrNotFound))
		})

		It("fails to delete item as regular user", func() {
			err := repo.Delete(simplePodcast.ID)
			Expect(err).To(Equal(rest.ErrPermissionDenied))
		})

		It("deletes as regular user when not admin only", func() {
			conf.Server.Podcast.AdminOnly = false
			err := repo.Delete(simplePodcast.ID)
			Expect(err).To(BeNil())

			_, err = repo.Get(simplePodcast.ID, false)
			Expect(err).To(MatchError(model.ErrNotFound))
		})
	})

	Describe("DeleteInternal", func() {
		It("deletes existing item", func() {
			err := adminRepo.DeleteInternal(simplePodcast.ID)
			Expect(err).To(BeNil())

			_, err = adminRepo.Get(simplePodcast.ID, false)
			Expect(err).To(MatchError(model.ErrNotFound))
		})

		It("deletes existing item internally as regular user", func() {
			err := repo.DeleteInternal(simplePodcast.ID)
			Expect(err).To(BeNil())

			_, err = repo.Get(simplePodcast.ID, false)
			Expect(err).To(MatchError(model.ErrNotFound))
		})
	})

	Describe("EntityName", func() {
		It("should return the right value", func() {
			Expect(adminRepo.EntityName()).To(Equal("podcast"))
			Expect(repo.EntityName()).To(Equal("podcast"))
		})
	})

	Describe("Get", func() {
		DescribeTable("GetExisting",
			func(withPodcasts bool, podcast model.Podcast, episodes ...model.PodcastEpisode) {
				for _, r := range repos {
					retrieved, err := r.Get(podcast.ID, withPodcasts)
					Expect(err).To(BeNil())

					expected := podcast
					if withPodcasts {
						for i := range retrieved.PodcastEpisodes {
							Expect(retrieved.PodcastEpisodes[i].UpdatedAt).ToNot(Equal(time.Time{}))
							retrieved.PodcastEpisodes[i].UpdatedAt = time.Time{}
						}

						expected.PodcastEpisodes = episodes

					}

					if expected.ID == fullPodcast.ID {
						podcast, _ := r.Get(fullPodcast.ID, false)
						expected.Starred = true
						expected.StarredAt = podcast.StarredAt
					}

					retrieved.CreatedAt = podcast.CreatedAt
					retrieved.UpdatedAt = podcast.UpdatedAt
					Expect(*retrieved).To(Equal(expected))
				}
			},

			Entry("Simple podcast, no episodes", false, simplePodcast),
			Entry("Simple podcast with episodes", true, simplePodcast, basicEpisode, brokenEpisode),
			Entry("Complex podcast, no episodes", false, fullPodcast),
			Entry("Complex podcast, with episodes", true, fullPodcast, completeEpisode),
			Entry("Podcast with no episodes", true, erroredPodcast),
		)

		It("returns ErrNotFound when the podcast does not exist", func() {
			for _, r := range repos {
				_, err := r.Get("nonexistent", true)
				Expect(err).To(MatchError(model.ErrNotFound))

				_, err = r.Get("nonexistent", false)
				Expect(err).To(MatchError(model.ErrNotFound))
			}
		})
	})

	Describe("GetAll", func() {
		It("gets all items without episodes", func() {
			for _, r := range repos {
				podcasts, err := r.GetAll(false)
				Expect(err).To(BeNil())
				for i := range podcasts {
					podcasts[i].UpdatedAt = testPodcasts[i].UpdatedAt
				}
				Expect(podcasts).To(Equal(testPodcasts))
			}

		})

		It("gets all items with episodes", func() {
			for _, r := range repos {
				podcasts, err := r.GetAll(true)
				Expect(err).To(BeNil())

				first := simplePodcast
				first.PodcastEpisodes = model.PodcastEpisodes{basicEpisode, brokenEpisode}

				second := fullPodcast
				second.PodcastEpisodes = model.PodcastEpisodes{completeEpisode}

				third := erroredPodcast
				third.PodcastEpisodes = model.PodcastEpisodes{}

				expected := model.Podcasts{first, second, third}

				for i := range podcasts {
					for j := range podcasts[i].PodcastEpisodes {
						podcasts[i].PodcastEpisodes[j].UpdatedAt = time.Time{}
					}
					podcasts[i].UpdatedAt = expected[i].UpdatedAt
				}
				Expect(podcasts).To(Equal(expected))
			}

		})
	})

	Describe("Put", func() {
		It("Successfully updates items", func() {
			podcast := fullPodcast
			podcast.Annotations = model.Annotations{}
			podcast.ID = brokenEpisode.ID

			err := adminRepo.Put(&podcast)
			Expect(err).To(BeNil())

			item, err := adminRepo.Get(podcast.ID, false)
			Expect(err).To(BeNil())
			item.UpdatedAt = podcast.UpdatedAt
			Expect(*item).To(Equal(podcast))
		})

		It("successfully creates an item", func() {
			podcast := model.Podcast{
				Url:             "https://example.com/nested/stream.url",
				PodcastEpisodes: model.PodcastEpisodes{},
			}

			err := adminRepo.Put(&podcast)
			Expect(podcast.ID).ToNot(BeEmpty())

			Expect(err).To(BeNil())
			Expect(adminRepo.CountAll()).To(Equal(int64(4)))

			item, err := adminRepo.Get(podcast.ID, true)
			Expect(err).To(BeNil())

			Expect(item.CreatedAt).ToNot(Equal(podcast.CreatedAt))
			Expect(item.UpdatedAt).ToNot(Equal(podcast.UpdatedAt))
			item.CreatedAt = podcast.CreatedAt
			item.UpdatedAt = podcast.UpdatedAt

			Expect(*item).To(Equal(podcast))
		})

		It("fails to update as regular user", func() {
			err := repo.Put(&fullPodcast)
			Expect(err).To(Equal(rest.ErrPermissionDenied))
		})

		It("Successfully updates as regular user when not admin only", func() {
			conf.Server.Podcast.AdminOnly = false

			podcast := fullPodcast
			podcast.Annotations = model.Annotations{}
			podcast.ID = brokenEpisode.ID

			err := repo.Put(&podcast)
			Expect(err).To(BeNil())

			item, err := repo.Get(podcast.ID, false)
			Expect(err).To(BeNil())
			item.UpdatedAt = podcast.UpdatedAt
			Expect(*item).To(Equal(podcast))
		})
	})

	Describe("PutInternal", func() {
		It("Successfully updates items", func() {
			for _, r := range repos {
				podcast := fullPodcast
				podcast.Annotations = model.Annotations{}
				podcast.ID = brokenEpisode.ID

				err := r.PutInternal(&podcast)
				Expect(err).To(BeNil())

				item, err := r.Get(podcast.ID, false)
				Expect(err).To(BeNil())
				item.UpdatedAt = podcast.UpdatedAt
				Expect(*item).To(Equal(podcast))
			}
		})

		It("successfully creates an item", func() {
			for i, r := range repos {
				podcast := model.Podcast{
					Url:             "https://example.com/nested/stream.url",
					PodcastEpisodes: model.PodcastEpisodes{},
				}

				err := r.PutInternal(&podcast)
				Expect(podcast.ID).ToNot(BeEmpty())

				Expect(err).To(BeNil())
				Expect(adminRepo.CountAll()).To(Equal(int64(4 + i)))

				item, err := adminRepo.Get(podcast.ID, true)
				Expect(err).To(BeNil())

				Expect(item.CreatedAt).ToNot(Equal(podcast.CreatedAt))
				Expect(item.UpdatedAt).ToNot(Equal(podcast.UpdatedAt))
				item.CreatedAt = podcast.CreatedAt
				item.UpdatedAt = podcast.UpdatedAt

				Expect(*item).To(Equal(podcast))
			}
		})
	})

	Describe("cleanup", Ordered, func() {
		BeforeEach(func() {
			_, err := database.NewQuery("PRAGMA foreign_keys = ON").Execute()
			if err != nil {
				panic(err)
			}
		})

		AfterEach(func() {
			_, err := database.NewQuery("PRAGMA foreign_keys = OFF").Execute()
			if err != nil {
				panic(err)
			}

			adminCtx := request.WithUser(log.NewContext(context.TODO()), model.User{ID: "userid", UserName: "johndoe", IsAdmin: true})
			epRepo := NewPodcastEpisodeRepository(adminCtx, database)

			all, err := epRepo.GetAll()
			if err != nil {
				panic(err)
			}

			for _, episode := range all {
				err = epRepo.Delete(episode.ID)
				if err != nil {
					panic(err)
				}
			}

			for i := range testPodcastEpisodes {
				r := testPodcastEpisodes[i]
				err = epRepo.Put(&r)
				if err != nil {
					panic(err)
				}
			}
		})

		It("removes annotations and nested episodes", func() {
			err := adminRepo.Delete(fullPodcast.ID)
			Expect(err).To(BeNil())

			err = adminRepo.Cleanup()
			Expect(err).To(BeNil())

			stillExisting, err := adminRepo.Get(fullPodcast.ID, false)
			Expect(stillExisting).To(BeNil())
			Expect(err).To(Equal(model.ErrNotFound))

			err = adminRepo.Put(&fullPodcast)
			Expect(err).To(BeNil())

			expected := fullPodcast
			expected.Annotations = model.Annotations{}
			expected.PodcastEpisodes = model.PodcastEpisodes{}

			podcast, err := adminRepo.Get(fullPodcast.ID, true)
			Expect(err).To(BeNil())
			Expect(*podcast).To(BeComparableTo(expected))
		})
	})
})
