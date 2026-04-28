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

	Describe("GetPodcasts — Podcasting 2.0 channel fields", func() {
		BeforeEach(func() {
			channelRepo.Data = map[string]*model.PodcastChannel{
				"ch-p20": {
					ID:              "ch-p20",
					Title:           "P2.0 Podcast",
					Status:          model.PodcastStatusCompleted,
					PodcastGUID:     "917393e3-1b1e-5cef-ace4-edaa54e1f810",
					Locked:          true,
					Medium:          "podcast",
					FundingURL:      "https://example.com/donate",
					FundingText:     "Support us!",
					UpdateFrequency: "Weekly",
					UpdateRRule:     "FREQ=WEEKLY",
					Complete:        false,
				},
			}
		})

		It("includes podcastGuid in response", func() {
			r := httptest.NewRequest("GET", "/rest/getPodcasts", nil)
			r = r.WithContext(userCtx)
			resp, err := api.GetPodcasts(r)
			Expect(err).ToNot(HaveOccurred())
			Expect(resp.Podcasts.Channel[0].PodcastGuid).To(Equal("917393e3-1b1e-5cef-ace4-edaa54e1f810"))
		})

		It("includes locked flag in response", func() {
			r := httptest.NewRequest("GET", "/rest/getPodcasts", nil)
			r = r.WithContext(userCtx)
			resp, err := api.GetPodcasts(r)
			Expect(err).ToNot(HaveOccurred())
			Expect(resp.Podcasts.Channel[0].Locked).To(BeTrue())
		})

		It("includes medium in response", func() {
			r := httptest.NewRequest("GET", "/rest/getPodcasts", nil)
			r = r.WithContext(userCtx)
			resp, err := api.GetPodcasts(r)
			Expect(err).ToNot(HaveOccurred())
			Expect(resp.Podcasts.Channel[0].Medium).To(Equal("podcast"))
		})

		It("includes funding items in response", func() {
			fundingRepo := tests.CreateMockPodcastFundingRepo()
			_ = fundingRepo.SaveForChannel("ch-p20", []model.PodcastFundingItem{
				{URL: "https://example.com/donate", Text: "Support us!"},
			})
			ds.MockedPodcastFunding = fundingRepo
			r := httptest.NewRequest("GET", "/rest/getPodcasts", nil)
			r = r.WithContext(userCtx)
			resp, err := api.GetPodcasts(r)
			Expect(err).ToNot(HaveOccurred())
			ch := resp.Podcasts.Channel[0]
			Expect(ch.Funding).To(HaveLen(1))
			Expect(ch.Funding[0].URL).To(Equal("https://example.com/donate"))
			Expect(ch.Funding[0].Text).To(Equal("Support us!"))
		})

		It("includes updateFrequency in response", func() {
			r := httptest.NewRequest("GET", "/rest/getPodcasts", nil)
			r = r.WithContext(userCtx)
			resp, err := api.GetPodcasts(r)
			Expect(err).ToNot(HaveOccurred())
			Expect(resp.Podcasts.Channel[0].UpdateFrequency).To(Equal("Weekly"))
		})

		It("includes channel person list from PersonRepo", func() {
			personRepo := tests.CreateMockPodcastPersonRepo()
			_ = personRepo.SaveForChannel("ch-p20", []model.PodcastPerson{
				{Name: "Jane Host", Role: "host", Group: "cast"},
			})
			ds.MockedPodcastPerson = personRepo

			r := httptest.NewRequest("GET", "/rest/getPodcasts", nil)
			r = r.WithContext(userCtx)
			resp, err := api.GetPodcasts(r)
			Expect(err).ToNot(HaveOccurred())
			Expect(resp.Podcasts.Channel[0].Person).To(HaveLen(1))
			Expect(resp.Podcasts.Channel[0].Person[0].Name).To(Equal("Jane Host"))
			Expect(resp.Podcasts.Channel[0].Person[0].Role).To(Equal("host"))
		})
	})

	Describe("GetPodcastEpisode — Podcasting 2.0 episode fields", func() {
		BeforeEach(func() {
			episodeRepo.Data["ep-p20"] = &model.PodcastEpisode{
				ID:             "ep-p20",
				Title:          "P2.0 Episode",
				Status:         model.PodcastStatusCompleted,
				Season:         2,
				SeasonName:     "Season Two",
				EpisodeNumber:  "5",
				EpisodeDisplay: "Ep.5",
				ChaptersURL:    "https://example.com/chapters.json",
				ChaptersType:   "application/json+chapters",
				SoundbiteStart: 73.5,
				SoundbiteDur:   60.0,
				SoundbiteTitle: "Best moment",
			}
		})

		It("includes season number and name in response", func() {
			r := newGetRequest("id=ep-p20")
			r = r.WithContext(userCtx)
			resp, err := api.GetPodcastEpisode(r)
			Expect(err).ToNot(HaveOccurred())
			Expect(resp.PodcastEpisode.Season).To(Equal(2))
			Expect(resp.PodcastEpisode.SeasonName).To(Equal("Season Two"))
		})

		It("includes episode number and display label in response", func() {
			r := newGetRequest("id=ep-p20")
			r = r.WithContext(userCtx)
			resp, err := api.GetPodcastEpisode(r)
			Expect(err).ToNot(HaveOccurred())
			Expect(resp.PodcastEpisode.EpisodeNumber).To(Equal("5"))
			Expect(resp.PodcastEpisode.EpisodeDisplay).To(Equal("Ep.5"))
		})

		It("includes chaptersUrl in response", func() {
			r := newGetRequest("id=ep-p20")
			r = r.WithContext(userCtx)
			resp, err := api.GetPodcastEpisode(r)
			Expect(err).ToNot(HaveOccurred())
			Expect(resp.PodcastEpisode.ChaptersUrl).To(Equal("https://example.com/chapters.json"))
		})

		It("includes soundbite fields in response", func() {
			r := newGetRequest("id=ep-p20")
			r = r.WithContext(userCtx)
			resp, err := api.GetPodcastEpisode(r)
			Expect(err).ToNot(HaveOccurred())
			Expect(resp.PodcastEpisode.SoundbiteStart).To(BeNumerically("~", 73.5, 0.001))
			Expect(resp.PodcastEpisode.SoundbiteDur).To(BeNumerically("~", 60.0, 0.001))
		})

		It("includes transcript array from TranscriptRepo in response", func() {
			transcriptRepo := tests.CreateMockPodcastTranscriptRepo()
			_ = transcriptRepo.Save([]model.PodcastTranscript{
				{EpisodeID: "ep-p20", URL: "https://example.com/t.vtt", MimeType: "text/vtt", Language: "en", Rel: "captions"},
				{EpisodeID: "ep-p20", URL: "https://example.com/t.srt", MimeType: "application/x-subrip", Language: "en"},
			})
			ds.MockedPodcastTranscript = transcriptRepo

			r := newGetRequest("id=ep-p20")
			r = r.WithContext(userCtx)
			resp, err := api.GetPodcastEpisode(r)
			Expect(err).ToNot(HaveOccurred())
			Expect(resp.PodcastEpisode.Transcript).To(HaveLen(2))
			// map iteration order is non-deterministic; use ConsistOf for order-independent check
			types := []string{
				resp.PodcastEpisode.Transcript[0].Type,
				resp.PodcastEpisode.Transcript[1].Type,
			}
			Expect(types).To(ConsistOf("text/vtt", "application/x-subrip"))
			var vttRel string
			for _, t := range resp.PodcastEpisode.Transcript {
				if t.Type == "text/vtt" {
					vttRel = t.Rel
				}
			}
			Expect(vttRel).To(Equal("captions"))
		})

		It("includes person array from PersonRepo in response", func() {
			personRepo := tests.CreateMockPodcastPersonRepo()
			_ = personRepo.SaveForEpisode("ep-p20", []model.PodcastPerson{
				{Name: "Jane Host", Role: "host", Group: "cast"},
			})
			ds.MockedPodcastPerson = personRepo

			r := newGetRequest("id=ep-p20")
			r = r.WithContext(userCtx)
			resp, err := api.GetPodcastEpisode(r)
			Expect(err).ToNot(HaveOccurred())
			Expect(resp.PodcastEpisode.Person).To(HaveLen(1))
			Expect(resp.PodcastEpisode.Person[0].Name).To(Equal("Jane Host"))
		})
	})

	Describe("GetPodcasts — Tier 3 fields", func() {
		BeforeEach(func() {
			channelRepo.Data = map[string]*model.PodcastChannel{
				"ch-t3": {
					ID:          "ch-t3",
					Title:       "Tier3 Podcast",
					Status:      model.PodcastStatusCompleted,
					UsesPodping: true,
				},
			}
		})

		It("includes usesPodping in channel response", func() {
			r := httptest.NewRequest("GET", "/rest/getPodcasts", nil)
			r = r.WithContext(userCtx)
			resp, err := api.GetPodcasts(r)
			Expect(err).ToNot(HaveOccurred())
			Expect(resp.Podcasts.Channel[0].UsesPodping).To(BeTrue())
		})

		It("includes podroll items in channel response", func() {
			podrollRepo := tests.CreateMockPodcastPodrollRepo()
			_ = podrollRepo.SaveForChannel("ch-t3", []model.PodcastPodrollItem{
				{FeedGUID: "guid-a", FeedURL: "https://a.example.com/feed.xml", Title: "Show A"},
				{FeedGUID: "guid-b", FeedURL: "https://b.example.com/feed.xml"},
			})
			ds.MockedPodcastPodroll = podrollRepo

			r := httptest.NewRequest("GET", "/rest/getPodcasts", nil)
			r = r.WithContext(userCtx)
			resp, err := api.GetPodcasts(r)
			Expect(err).ToNot(HaveOccurred())
			ch := resp.Podcasts.Channel[0]
			Expect(ch.Podroll).To(HaveLen(2))
			feedURLs := []string{ch.Podroll[0].FeedURL, ch.Podroll[1].FeedURL}
			Expect(feedURLs).To(ConsistOf("https://a.example.com/feed.xml", "https://b.example.com/feed.xml"))
		})

		It("includes podroll title and feedGuid", func() {
			podrollRepo := tests.CreateMockPodcastPodrollRepo()
			_ = podrollRepo.SaveForChannel("ch-t3", []model.PodcastPodrollItem{
				{FeedGUID: "guid-a", FeedURL: "https://a.example.com/feed.xml", Title: "Show A"},
			})
			ds.MockedPodcastPodroll = podrollRepo

			r := httptest.NewRequest("GET", "/rest/getPodcasts", nil)
			r = r.WithContext(userCtx)
			resp, err := api.GetPodcasts(r)
			Expect(err).ToNot(HaveOccurred())
			item := resp.Podcasts.Channel[0].Podroll[0]
			Expect(item.FeedGUID).To(Equal("guid-a"))
			Expect(item.Title).To(Equal("Show A"))
		})

		It("includes liveItem in channel response", func() {
			liveItemRepo := tests.CreateMockPodcastLiveItemRepo()
			_ = liveItemRepo.Upsert(&model.PodcastLiveItem{
				ChannelID:       "ch-t3",
				GUID:            "live-guid-001",
				Title:           "Live Show",
				Status:          "live",
				EnclosureURL:    "https://stream.example.com/live.m3u8",
				EnclosureType:   "application/x-mpegURL",
				ContentLinkURL:  "https://youtube.com/live",
				ContentLinkText: "Watch Live",
			})
			ds.MockedPodcastLiveItem = liveItemRepo

			r := httptest.NewRequest("GET", "/rest/getPodcasts", nil)
			r = r.WithContext(userCtx)
			resp, err := api.GetPodcasts(r)
			Expect(err).ToNot(HaveOccurred())
			ch := resp.Podcasts.Channel[0]
			Expect(ch.LiveItem).ToNot(BeNil())
			Expect(ch.LiveItem.Status).To(Equal("live"))
			Expect(ch.LiveItem.GUID).To(Equal("live-guid-001"))
			Expect(ch.LiveItem.Title).To(Equal("Live Show"))
			Expect(ch.LiveItem.EnclosureURL).To(Equal("https://stream.example.com/live.m3u8"))
			Expect(ch.LiveItem.ContentLinkURL).To(Equal("https://youtube.com/live"))
			Expect(ch.LiveItem.ContentLinkText).To(Equal("Watch Live"))
		})

		It("omits liveItem when none exists", func() {
			r := httptest.NewRequest("GET", "/rest/getPodcasts", nil)
			r = r.WithContext(userCtx)
			resp, err := api.GetPodcasts(r)
			Expect(err).ToNot(HaveOccurred())
			Expect(resp.Podcasts.Channel[0].LiveItem).To(BeNil())
		})

		It("formats liveItem startTime and endTime as RFC3339", func() {
			liveItemRepo := tests.CreateMockPodcastLiveItemRepo()
			_ = liveItemRepo.Upsert(&model.PodcastLiveItem{
				ChannelID: "ch-t3",
				Status:    "live",
				StartTime: time.Date(2024, 4, 27, 8, 0, 0, 0, time.UTC),
				EndTime:   time.Date(2024, 4, 27, 9, 0, 0, 0, time.UTC),
			})
			ds.MockedPodcastLiveItem = liveItemRepo

			r := httptest.NewRequest("GET", "/rest/getPodcasts", nil)
			r = r.WithContext(userCtx)
			resp, err := api.GetPodcasts(r)
			Expect(err).ToNot(HaveOccurred())
			li := resp.Podcasts.Channel[0].LiveItem
			Expect(li).ToNot(BeNil())
			Expect(li.StartTime).To(Equal("2024-04-27T08:00:00Z"))
			Expect(li.EndTime).To(Equal("2024-04-27T09:00:00Z"))
		})
	})
})
