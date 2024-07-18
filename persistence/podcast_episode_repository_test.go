package persistence

import (
	"context"
	"time"

	"github.com/navidrome/navidrome/db"
	"github.com/navidrome/navidrome/log"
	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/model/request"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("PodcastEpisodeRepository", func() {
	var repo model.PodcastEpisodeRepository
	var database *dbxBuilder
	var ctx context.Context

	BeforeEach(func() {
		database = NewDBXBuilder(db.Db())
		ctx = request.WithUser(log.NewContext(context.TODO()), model.User{ID: "userid", UserName: "johndoe", IsAdmin: false})
		repo = NewPodcastEpisodeRepository(ctx, database)

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
			Expect(retrieved).To(Equal(episode))
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
			Expect(episodes).To(Equal(testPodcastEpisodes))
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
			Expect(eps).To(Equal(model.PodcastEpisodes{completeEpisode, basicEpisode}))

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

			Expect(*item).To(Equal(episode))
		})
	})

	Context("Bookmarks", func() {
		var mfRepo model.MediaFileRepository

		validateMfUnchanged := func() {
			mfBook, err := mfRepo.GetBookmarks()
			Expect(err).To(BeNil())
			Expect(mfBook).To(HaveLen(1))
		}

		BeforeEach(func() {
			mfRepo = NewMediaFileRepository(ctx, database)
			_ = mfRepo.AddBookmark(songDayInALife.ID, "comment", 60)
		})

		AfterEach(func() {
			_ = repo.Cleanup()
			_ = mfRepo.DeleteBookmark(songDayInALife.ID)
		})

		It("should add bookmark", func() {
			now := time.Now()
			err := repo.AddBookmark(completeEpisode.ID, "this is a comment", int64(4))
			Expect(err).To(BeNil())

			validateMfUnchanged()

			bookmark, err := repo.GetBookmarks()
			Expect(err).To(BeNil())
			Expect(bookmark).To(HaveLen(1))
			Expect(bookmark[0].CreatedAt).To(BeTemporally(">", now))
			Expect(bookmark[0].UpdatedAt).To(BeTemporally(">", now))
			bookmark[0].Item.UpdatedAt = time.Time{}
			bookmark[0].CreatedAt = time.Time{}
			bookmark[0].UpdatedAt = time.Time{}

			mf := *completeEpisode.ToMediaFile()
			mf.BookmarkPosition = 4

			Expect(bookmark).To(Equal(model.Bookmarks{
				{
					Item: model.MediaFile{
						Annotations: model.Annotations{
							Starred:   true,
							StarredAt: completeEpisode.StarredAt,
						},
						Bookmarkable: model.Bookmarkable{
							BookmarkPosition: 4,
						},
						ID:          "pe-2",
						LibraryID:   0,
						Path:        completeEpisode.AbsolutePath(),
						Title:       "Sample episode",
						AlbumID:     "pd-2345",
						HasCoverArt: false,
						Year:        2024,
						Size:        41256,
						Suffix:      "ogg",
						Duration:    30.5,
						BitRate:     320,
					},
					Comment:  "this is a comment",
					Position: 4,
				},
			}))
		})

		It("should delete bookmark", func() {
			err := repo.AddBookmark(completeEpisode.ID, "this is a comment", int64(4))
			Expect(err).To(BeNil())

			validateMfUnchanged()

			bookmark, err := repo.GetBookmarks()
			Expect(err).To(BeNil())
			Expect(bookmark).To(HaveLen(1))
			Expect(bookmark[0].Item.ID).To(Equal(completeEpisode.ExternalId()))

			// Run this twice. Second delete should have no impact
			for range 2 {
				err = repo.DeleteBookmark(completeEpisode.ID)
				Expect(err).To(BeNil())
				validateMfUnchanged()
			}
		})

		It("should cleanup bookmarks", func() {
			for _, item := range testPodcastEpisodes {
				err := repo.AddBookmark(item.ID, "a comment", 4)
				Expect(err).To(BeNil())
			}

			validateMfUnchanged()

			bookmark, err := repo.GetBookmarks()
			Expect(err).To(BeNil())
			Expect(bookmark).To(HaveLen(3))
			Expect(bookmark[0].Item.ID).To(Equal(basicEpisode.ExternalId()))
			Expect(bookmark[1].Item.ID).To(Equal(completeEpisode.ExternalId()))
			Expect(bookmark[2].Item.ID).To(Equal(brokenEpisode.ExternalId()))

			for _, ep := range testPodcastEpisodes[1:] {
				err = repo.Delete(ep.ID)
				Expect(err).To(BeNil())
			}

			err = repo.Cleanup()
			Expect(err).To(BeNil())

			bookmark, err = repo.GetBookmarks()
			Expect(err).To(BeNil())
			Expect(bookmark).To(HaveLen(1))
			Expect(bookmark[0].Item.ID).To(Equal(basicEpisode.ExternalId()))

			validateMfUnchanged()
		})
	})
})
