package subsonic

import (
	"context"
	"net/http"
	"time"

	"github.com/deluan/rest"
	"github.com/navidrome/navidrome/conf"
	"github.com/navidrome/navidrome/conf/configtest"
	"github.com/navidrome/navidrome/consts"
	"github.com/navidrome/navidrome/core"
	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/model/request"
	"github.com/navidrome/navidrome/server/subsonic/responses"
	"github.com/navidrome/navidrome/tests"
	"github.com/navidrome/navidrome/utils/gg"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Subsonic", func() {
	var router *Router
	var podcast *tests.MockedPodcastRepo
	var episode *tests.MockedPodcastEpisodeRepo
	var ds model.DataStore
	var manager *fakePodcastManager
	var adminCtx context.Context
	var r *http.Request

	BeforeEach(func() {
		podcast = tests.CreateMockPodcastRepo()
		episode = tests.CreateMockedPodcastEpisodeRepo()
		ds = &tests.MockDataStore{MockedPodcast: podcast, MockedPodcastEpisode: episode}
		manager = &fakePodcastManager{}
		router = New(ds, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, manager)
		DeferCleanup(configtest.SetupConfig())
		conf.Server.Podcast.AdminOnly = true

		adminCtx = request.WithUser(context.TODO(), model.User{ID: "1234", IsAdmin: true})
	})

	Describe("createPodcastChannel", func() {
		BeforeEach(func() {
			r = newGetRequest("url=https://example.org")
		})

		Describe("admin", func() {
			BeforeEach(func() {
				r = r.WithContext(adminCtx)
			})

			It("should error if creating feed fails", func() {
				manager.err = newError(responses.ErrorGeneric, "Error creating feed")

				resp, err := router.CreatePodcastChannel(r)
				Expect(err).To(Equal(manager.err))
				Expect(resp).To(BeNil())
				Expect(manager.savedId).To(Equal("https://example.org"))
			})

			It("responds with empty body on success", func() {
				resp, err := router.CreatePodcastChannel(r)
				Expect(err).To(BeNil())
				Expect(resp).To(Equal(newResponse()))
				Expect(manager.savedId).To(Equal("https://example.org"))
			})
		})

		Describe("regular user", func() {
			It("denies access", func() {
				resp, err := router.CreatePodcastChannel(r)
				Expect(err).To(Equal(newError(responses.ErrorAuthorizationFail, "Creating podcasts is admin only")))
				Expect(resp).To(BeNil())
				Expect(manager.savedId).To(Equal(""))
			})

			It("allows access if not globally limited", func() {
				conf.Server.Podcast.AdminOnly = false

				resp, err := router.CreatePodcastChannel(r)
				Expect(err).To(BeNil())
				Expect(resp).To(Equal(newResponse()))
				Expect(manager.savedId).To(Equal("https://example.org"))
			})
		})
	})

	Describe("deletePodcastChannel", func() {
		BeforeEach(func() {
			r = newGetRequest("id=pd-1234")
		})

		Context("admin", func() {
			BeforeEach(func() {
				r = r.WithContext(adminCtx)
			})

			It("should error if deleting fails", func() {
				manager.err = newError(responses.ErrorGeneric, "Error deleting feed")

				resp, err := router.DeletePodcastChannel(r)
				Expect(err).To(Equal(manager.err))
				Expect(resp).To(BeNil())
				Expect(manager.savedId).To(Equal("1234"))
			})

			It("responds with empty body on success", func() {
				resp, err := router.DeletePodcastChannel(r)
				Expect(err).To(BeNil())
				Expect(resp).To(Equal(newResponse()))
				Expect(manager.savedId).To(Equal("1234"))
			})

			It("fails to parse non-podcast id", func() {
				r = newGetRequest("id=1234")
				r = r.WithContext(adminCtx)

				resp, err := router.DeletePodcastChannel(r)
				Expect(err).To(Equal(rest.ErrNotFound))
				Expect(resp).To(BeNil())
				Expect(manager.savedId).To(Equal(""))
			})
		})

		Context("regular user", func() {
			It("denies access", func() {
				resp, err := router.DeletePodcastChannel(r)
				Expect(err).To(Equal(newError(responses.ErrorAuthorizationFail, "Deleting podcasts is admin only")))
				Expect(resp).To(BeNil())
				Expect(manager.savedId).To(Equal(""))
			})

			It("allows access if not globally limited", func() {
				conf.Server.Podcast.AdminOnly = false

				resp, err := router.DeletePodcastChannel(r)
				Expect(err).To(BeNil())
				Expect(resp).To(Equal(newResponse()))
				Expect(manager.savedId).To(Equal("1234"))
			})
		})
	})

	Describe("deletePodcastEpisode", func() {
		BeforeEach(func() {
			r = newGetRequest("id=pe-1234")
		})

		Context("admin", func() {
			BeforeEach(func() {
				r = r.WithContext(adminCtx)
			})

			It("should error if deleting fails", func() {
				manager.err = newError(responses.ErrorGeneric, "Error deleting feed")

				resp, err := router.DeletePodcastEpisode(r)
				Expect(err).To(Equal(manager.err))
				Expect(resp).To(BeNil())
				Expect(manager.savedId).To(Equal("1234"))
			})

			It("responds with empty body on success", func() {
				resp, err := router.DeletePodcastEpisode(r)
				Expect(err).To(BeNil())
				Expect(resp).To(Equal(newResponse()))
				Expect(manager.savedId).To(Equal("1234"))
			})

			It("fails to parse non-podcast id", func() {
				r = newGetRequest("id=1234")
				r = r.WithContext(adminCtx)

				resp, err := router.DeletePodcastEpisode(r)
				Expect(err).To(Equal(rest.ErrNotFound))
				Expect(resp).To(BeNil())
				Expect(manager.savedId).To(Equal(""))
			})
		})

		Context("regular user", func() {
			It("denies access", func() {
				resp, err := router.DeletePodcastEpisode(r)
				Expect(err).To(Equal(newError(responses.ErrorAuthorizationFail, "Deleting podcast episodes is admin only")))
				Expect(resp).To(BeNil())
				Expect(manager.savedId).To(Equal(""))
			})

			It("allows access if not globally limited", func() {
				conf.Server.Podcast.AdminOnly = false

				resp, err := router.DeletePodcastEpisode(r)
				Expect(err).To(BeNil())
				Expect(resp).To(Equal(newResponse()))
				Expect(manager.savedId).To(Equal("1234"))
			})
		})
	})

	Describe("downloadPodcastEpisode", func() {
		BeforeEach(func() {
			r = newGetRequest("id=pe-1234")
		})

		Context("admin", func() {
			BeforeEach(func() {
				r = r.WithContext(adminCtx)
			})

			It("should error if deleting fails", func() {
				manager.err = newError(responses.ErrorGeneric, "Error deleting feed")

				resp, err := router.DownloadPodcastEpisode(r)
				Expect(err).To(Equal(manager.err))
				Expect(resp).To(BeNil())
				Expect(manager.savedId).To(Equal("1234"))
			})

			It("responds with empty body on success", func() {
				resp, err := router.DownloadPodcastEpisode(r)
				Expect(err).To(BeNil())
				Expect(resp).To(Equal(newResponse()))
				Expect(manager.savedId).To(Equal("1234"))
			})

			It("fails to parse non-podcast id", func() {
				r = newGetRequest("id=1234")
				r = r.WithContext(adminCtx)

				resp, err := router.DownloadPodcastEpisode(r)
				Expect(err).To(Equal(rest.ErrNotFound))
				Expect(resp).To(BeNil())
				Expect(manager.savedId).To(Equal(""))
			})
		})

		Context("regular user", func() {
			It("denies access", func() {
				resp, err := router.DownloadPodcastEpisode(r)
				Expect(err).To(Equal(newError(responses.ErrorAuthorizationFail, "Downloading podcast episodes is admin only")))
				Expect(resp).To(BeNil())
				Expect(manager.savedId).To(Equal(""))
			})

			It("allows access if not globally limited", func() {
				conf.Server.Podcast.AdminOnly = false

				resp, err := router.DownloadPodcastEpisode(r)
				Expect(err).To(BeNil())
				Expect(resp).To(Equal(newResponse()))
				Expect(manager.savedId).To(Equal("1234"))
			})
		})
	})

	Describe("getPodcasts", func() {
		var expectedResponse responses.Podcasts

		BeforeEach(func() {
			podcast.SetData(model.Podcasts{
				{
					ID:          "full",
					Url:         "https://example.org",
					Title:       "Title",
					Description: "Description",
					ImageUrl:    "https://example.org/favicon.ico",
					PodcastEpisodes: model.PodcastEpisodes{
						{
							Annotations: model.Annotations{
								PlayCount: 5,
								Rating:    3,
								Starred:   true,
								StarredAt: &time.Time{},
							},
							ID:          "full",
							Guid:        "guid",
							PodcastId:   "full",
							Url:         "https://example.org/full.mp3",
							Title:       "Title",
							Description: "Description",
							ImageUrl:    "https://example.org/full.jpg",
							PublishDate: gg.P(time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)),
							Duration:    31.5,
							Suffix:      "mp3",
							Size:        124214,
							BitRate:     128,
							State:       consts.PodcastStatusCompleted,
						},
						{
							ID:        "partial",
							Guid:      "partial",
							PodcastId: "full",
							Url:       "https://example.org/partial.mp3",
							State:     consts.PodcastStatusError,
							Error:     "500 Internal Server Error",
						},
					},
				},
				{
					ID:    "empty",
					Url:   "https://example.org/broken",
					State: consts.PodcastStatusError,
					Error: "404 not found",
				},
			})

			expectedResponse = responses.Podcasts{
				Podcasts: []responses.PodcastChannel{
					{
						ID:               "pd-full",
						Url:              "https://example.org",
						Title:            "Title",
						Description:      "Description",
						CoverArt:         "pd-full",
						OriginalImageUrl: "https://example.org/favicon.ico",
						Episodes: []responses.PodcastEpisode{
							{
								Child: responses.Child{
									Id:          "pe-full",
									Title:       "Title",
									CoverArt:    "pe-full",
									Size:        124214,
									ContentType: "audio/mpeg",
									Suffix:      "mp3",
									Year:        2024,
									Duration:    31,
									BitRate:     128,
									Path:        "full/full.mp3",
									UserRating:  3,
									Starred:     &time.Time{},
									PlayCount:   5,
								},
								StreamId:    "pe-full",
								ChannelId:   "pd-full",
								Status:      consts.PodcastStatusCompleted,
								Description: "Description",
								PublishDate: gg.P(time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)),
							},
							{
								Child: responses.Child{
									Id:       "pe-partial",
									CoverArt: "pe-partial",
								},
								StreamId:     "pe-partial",
								ChannelId:    "pd-full",
								Status:       consts.PodcastStatusError,
								ErrorMessage: "500 Internal Server Error",
							},
						},
					},
					{
						ID:           "pd-empty",
						Url:          "https://example.org/broken",
						CoverArt:     "pd-empty",
						Status:       consts.PodcastStatusError,
						ErrorMessage: "404 not found",
					},
				},
			}
		})

		It("gets podcasts with episodes", func() {
			r = newGetRequest()
			resp, err := router.GetPodcasts(r)
			Expect(err).To(BeNil())
			Expect(*resp.Podcasts).To(Equal(expectedResponse))
		})

		It("gets podcasts without episodes", func() {
			r = newGetRequest("includeEpisodes=false")
			resp, err := router.GetPodcasts(r)
			Expect(err).To(BeNil())

			expectedResponse.Podcasts[0].Episodes = nil
			expectedResponse.Podcasts[1].Episodes = nil
			Expect(*resp.Podcasts).To(Equal(expectedResponse))
		})

		It("gets a single episode, without episodes", func() {
			r = newGetRequest("includeEpisodes=false&id=pd-empty")
			resp, err := router.GetPodcasts(r)
			Expect(err).To(BeNil())

			podcasts := []responses.PodcastChannel{}
			podcasts = append(podcasts, expectedResponse.Podcasts[1])
			podcasts[0].Episodes = nil
			expectedResponse.Podcasts = podcasts
			Expect(*resp.Podcasts).To(Equal(expectedResponse))
		})

		It("fails to parse bad id", func() {
			r = newGetRequest("id=empty")
			resp, err := router.GetPodcasts(r)
			Expect(err).To(Equal(rest.ErrNotFound))
			Expect(resp).To(BeNil())
		})

		It("fails when searching for nonexistent id", func() {
			r = newGetRequest("id=pe-idontexist")
			resp, err := router.GetPodcasts(r)
			Expect(err).To(Equal(rest.ErrNotFound))
			Expect(resp).To(BeNil())
		})
	})

	Describe("getNewestPodcasts", func() {
		// This test is solely to check the length of the return value.
		// Checking of the output is done in getPodcasts
		BeforeEach(func() {
			episodes := make(model.PodcastEpisodes, 25)

			for i := range len(episodes) {
				episodes[i] = model.PodcastEpisode{}
			}

			episode.SetData(episodes)
		})

		It("gets newest tracks", func() {
			r = newGetRequest()
			resp, err := router.GetNewestPodcasts(r)
			Expect(err).To(BeNil())
			Expect(resp.NewestPodcasts.Episodes).To(HaveLen(20))
		})

		It("gets all tracks with large enough value", func() {
			r = newGetRequest("count=25")
			resp, err := router.GetNewestPodcasts(r)
			Expect(err).To(BeNil())
			Expect(resp.NewestPodcasts.Episodes).To(HaveLen(25))
		})
	})

	Describe("refreshPodcasts", func() {
		BeforeEach(func() {
			r = newGetRequest()
		})

		Context("admin", func() {
			BeforeEach(func() {
				r = r.WithContext(adminCtx)
			})

			It("should error if deleting fails", func() {
				manager.err = newError(responses.ErrorGeneric, "Error deleting feed")

				resp, err := router.RefreshPodcasts(r)
				Expect(err).To(Equal(manager.err))
				Expect(resp).To(BeNil())
				Expect(manager.savedId).To(Equal(""))
			})

			It("responds with empty body on success", func() {
				resp, err := router.RefreshPodcasts(r)
				Expect(err).To(BeNil())
				Expect(resp).To(Equal(newResponse()))
				Expect(manager.savedId).To(Equal(""))
			})

		})

		Context("regular user", func() {
			It("denies access", func() {
				resp, err := router.RefreshPodcasts(r)
				Expect(err).To(Equal(newError(responses.ErrorAuthorizationFail, "Refreshing podcasts is admin only")))
				Expect(resp).To(BeNil())
				Expect(manager.savedId).To(Equal(""))
			})

			It("allows access if not globally limited", func() {
				conf.Server.Podcast.AdminOnly = false

				resp, err := router.RefreshPodcasts(r)
				Expect(err).To(BeNil())
				Expect(resp).To(Equal(newResponse()))
				Expect(manager.savedId).To(Equal(""))
			})
		})
	})
})

type fakePodcastManager struct {
	core.PodcastManager
	channel *model.Podcast
	err     error
	savedId string
}

func (f *fakePodcastManager) CreateFeed(ctx context.Context, url string) (*model.Podcast, error) {
	f.savedId = url
	if f.err != nil {
		return nil, f.err
	}

	return f.channel, nil
}

func (f *fakePodcastManager) DeletePodcast(ctx context.Context, id string) error {
	f.savedId = id
	return f.err
}

func (f *fakePodcastManager) DeletePodcastEpisode(ctx context.Context, id string) error {
	f.savedId = id
	return f.err
}

func (f *fakePodcastManager) Download(ctx context.Context, id string) error {
	f.savedId = id
	return f.err
}

func (f *fakePodcastManager) Refresh(ctx context.Context) error {
	return f.err
}
