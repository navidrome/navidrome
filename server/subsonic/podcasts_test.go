package subsonic

import (
	"context"
	"net/http/httptest"
	"time"

	"github.com/navidrome/navidrome/core/auth"
	"github.com/navidrome/navidrome/core/podcasts"
	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/model/request"
	"github.com/navidrome/navidrome/tests"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

// mockPodcastService is a local mock for the Podcasts service interface.
type mockPodcastService struct {
	podcasts.Podcasts
	addChannelURL     string
	refreshCalled     bool
	deleteChannelID   string
	deleteEpisodeID   string
	downloadEpisodeID string
	err               error
}

func (m *mockPodcastService) AddChannel(_ context.Context, rssURL string) error {
	m.addChannelURL = rssURL
	return m.err
}
func (m *mockPodcastService) RefreshChannels(_ context.Context) error {
	m.refreshCalled = true
	return m.err
}
func (m *mockPodcastService) DeleteChannel(_ context.Context, id string) error {
	m.deleteChannelID = id
	return m.err
}
func (m *mockPodcastService) DeleteEpisode(_ context.Context, id string) error {
	m.deleteEpisodeID = id
	return m.err
}
func (m *mockPodcastService) DownloadEpisode(_ context.Context, id string) error {
	m.downloadEpisodeID = id
	return m.err
}

var _ = Describe("Podcasts", func() {
	var api *Router
	var ds *tests.MockDataStore
	var channelRepo *tests.MockPodcastChannelRepo
	var episodeRepo *tests.MockPodcastEpisodeRepo
	var svc *mockPodcastService
	var adminCtx, userCtx context.Context

	BeforeEach(func() {
		ds = &tests.MockDataStore{}
		auth.Init(ds)
		channelRepo = tests.CreateMockPodcastChannelRepo()
		episodeRepo = tests.CreateMockPodcastEpisodeRepo()
		svc = &mockPodcastService{}
		ds.MockedPodcastChannel = channelRepo
		ds.MockedPodcastEpisode = episodeRepo

		api = &Router{ds: ds, podcasts: svc}
		adminCtx = request.WithUser(context.Background(), model.User{ID: "admin", IsAdmin: true})
		userCtx = request.WithUser(context.Background(), model.User{ID: "user", IsAdmin: false})
	})

	Describe("GetPodcasts", func() {
		BeforeEach(func() {
			channelRepo.Data = map[string]*model.PodcastChannel{
				"ch-1": {ID: "ch-1", Title: "Podcast 1", Status: model.PodcastStatusCompleted},
				"ch-2": {ID: "ch-2", Title: "Podcast 2", Status: model.PodcastStatusNew},
			}
		})

		It("returns all channels", func() {
			r := httptest.NewRequest("GET", "/rest/getPodcasts", nil)
			r = r.WithContext(adminCtx)

			resp, err := api.GetPodcasts(r)
			Expect(err).ToNot(HaveOccurred())
			Expect(resp.Podcasts.Channel).To(HaveLen(2))
		})

		It("returns channel fields correctly", func() {
			r := httptest.NewRequest("GET", "/rest/getPodcasts", nil)
			r = r.WithContext(adminCtx)

			resp, err := api.GetPodcasts(r)
			Expect(err).ToNot(HaveOccurred())
			ids := []string{resp.Podcasts.Channel[0].ID, resp.Podcasts.Channel[1].ID}
			Expect(ids).To(ContainElements("ch-1", "ch-2"))
		})

		It("filters by id when provided", func() {
			r := newGetRequest("id=ch-1")
			r = r.WithContext(adminCtx)

			resp, err := api.GetPodcasts(r)
			Expect(err).ToNot(HaveOccurred())
			Expect(resp.Podcasts.Channel).To(HaveLen(1))
			Expect(resp.Podcasts.Channel[0].ID).To(Equal("ch-1"))
		})

		It("is accessible by regular users", func() {
			r := httptest.NewRequest("GET", "/rest/getPodcasts", nil)
			r = r.WithContext(userCtx)

			_, err := api.GetPodcasts(r)
			Expect(err).ToNot(HaveOccurred())
		})
	})

	Describe("GetNewestPodcasts", func() {
		BeforeEach(func() {
			now := time.Now()
			episodeRepo.Data = map[string]*model.PodcastEpisode{
				"ep-1": {ID: "ep-1", Title: "Ep1", ChannelID: "ch-1", PublishDate: now.Add(-time.Hour), Status: model.PodcastStatusCompleted},
				"ep-2": {ID: "ep-2", Title: "Ep2", ChannelID: "ch-1", PublishDate: now, Status: model.PodcastStatusNew},
			}
		})

		It("returns episodes in Child format", func() {
			r := httptest.NewRequest("GET", "/rest/getNewestPodcasts", nil)
			r = r.WithContext(userCtx)

			resp, err := api.GetNewestPodcasts(r)
			Expect(err).ToNot(HaveOccurred())
			Expect(resp.NewestPodcasts.Episode).To(HaveLen(2))
		})

		It("sets type to podcast", func() {
			r := httptest.NewRequest("GET", "/rest/getNewestPodcasts", nil)
			r = r.WithContext(userCtx)

			resp, err := api.GetNewestPodcasts(r)
			Expect(err).ToNot(HaveOccurred())
			for _, ep := range resp.NewestPodcasts.Episode {
				Expect(ep.Type).To(Equal("podcast"))
			}
		})

		It("respects count parameter", func() {
			r := newGetRequest("count=1")
			r = r.WithContext(userCtx)

			resp, err := api.GetNewestPodcasts(r)
			Expect(err).ToNot(HaveOccurred())
			Expect(resp.NewestPodcasts.Episode).To(HaveLen(1))
		})

		It("defaults count to 20", func() {
			r := httptest.NewRequest("GET", "/rest/getNewestPodcasts", nil)
			r = r.WithContext(userCtx)

			_, err := api.GetNewestPodcasts(r)
			Expect(err).ToNot(HaveOccurred())
		})
	})

	Describe("CreatePodcastChannel", func() {
		It("calls service with URL", func() {
			r := newGetRequest("url=https://example.com/feed.xml")
			r = r.WithContext(adminCtx)

			_, err := api.CreatePodcastChannel(r)
			Expect(err).ToNot(HaveOccurred())
			Expect(svc.addChannelURL).To(Equal("https://example.com/feed.xml"))
		})

		It("denies non-admin users", func() {
			r := newGetRequest("url=https://example.com/feed.xml")
			r = r.WithContext(userCtx)

			_, err := api.CreatePodcastChannel(r)
			Expect(err).To(HaveOccurred())
		})

		It("returns error when url param is missing", func() {
			r := newGetRequest()
			r = r.WithContext(adminCtx)

			_, err := api.CreatePodcastChannel(r)
			Expect(err).To(HaveOccurred())
		})
	})

	Describe("RefreshPodcasts", func() {
		It("calls service", func() {
			r := httptest.NewRequest("GET", "/rest/refreshPodcasts", nil)
			r = r.WithContext(adminCtx)

			_, err := api.RefreshPodcasts(r)
			Expect(err).ToNot(HaveOccurred())
			Expect(svc.refreshCalled).To(BeTrue())
		})

		It("denies non-admin users", func() {
			r := httptest.NewRequest("GET", "/rest/refreshPodcasts", nil)
			r = r.WithContext(userCtx)

			_, err := api.RefreshPodcasts(r)
			Expect(err).To(HaveOccurred())
		})
	})

	Describe("DeletePodcastChannel", func() {
		It("calls service with id", func() {
			r := newGetRequest("id=ch-1")
			r = r.WithContext(adminCtx)

			_, err := api.DeletePodcastChannel(r)
			Expect(err).ToNot(HaveOccurred())
			Expect(svc.deleteChannelID).To(Equal("ch-1"))
		})

		It("denies non-admin users", func() {
			r := newGetRequest("id=ch-1")
			r = r.WithContext(userCtx)

			_, err := api.DeletePodcastChannel(r)
			Expect(err).To(HaveOccurred())
		})
	})

	Describe("DeletePodcastEpisode", func() {
		It("calls service with id", func() {
			r := newGetRequest("id=ep-1")
			r = r.WithContext(adminCtx)

			_, err := api.DeletePodcastEpisode(r)
			Expect(err).ToNot(HaveOccurred())
			Expect(svc.deleteEpisodeID).To(Equal("ep-1"))
		})

		It("denies non-admin users", func() {
			r := newGetRequest("id=ep-1")
			r = r.WithContext(userCtx)

			_, err := api.DeletePodcastEpisode(r)
			Expect(err).To(HaveOccurred())
		})
	})

	Describe("DownloadPodcastEpisode", func() {
		It("calls service with id", func() {
			r := newGetRequest("id=ep-1")
			r = r.WithContext(adminCtx)

			_, err := api.DownloadPodcastEpisode(r)
			Expect(err).ToNot(HaveOccurred())
			Expect(svc.downloadEpisodeID).To(Equal("ep-1"))
		})

		It("denies non-admin users", func() {
			r := newGetRequest("id=ep-1")
			r = r.WithContext(userCtx)

			_, err := api.DownloadPodcastEpisode(r)
			Expect(err).To(HaveOccurred())
		})
	})

	Describe("GetPodcastEpisode", func() {
		BeforeEach(func() {
			episodeRepo.Data["ep-1"] = &model.PodcastEpisode{
				ID:     "ep-1",
				Title:  "Test Episode",
				Status: model.PodcastStatusCompleted,
			}
		})

		It("returns single episode", func() {
			r := newGetRequest("id=ep-1")
			r = r.WithContext(userCtx)

			resp, err := api.GetPodcastEpisode(r)
			Expect(err).ToNot(HaveOccurred())
			Expect(resp.PodcastEpisode).ToNot(BeNil())
			Expect(resp.PodcastEpisode.ID).To(Equal("ep-1"))
		})

		It("returns error for unknown id", func() {
			r := newGetRequest("id=no-such-id")
			r = r.WithContext(userCtx)

			_, err := api.GetPodcastEpisode(r)
			Expect(err).To(HaveOccurred())
		})

		It("is accessible by regular users", func() {
			r := newGetRequest("id=ep-1")
			r = r.WithContext(userCtx)

			_, err := api.GetPodcastEpisode(r)
			Expect(err).ToNot(HaveOccurred())
		})
	})
})
