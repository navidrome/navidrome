package podcasts_test

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/navidrome/navidrome/conf"
	"github.com/navidrome/navidrome/conf/configtest"
	"github.com/navidrome/navidrome/core/podcasts"
	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/model/request"
	"github.com/navidrome/navidrome/tests"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("PodcastService", func() {
	var svc podcasts.Podcasts
	var ds *tests.MockDataStore
	var channelRepo *tests.MockPodcastChannelRepo
	var episodeRepo *tests.MockPodcastEpisodeRepo
	var mockServer *httptest.Server
	var ctx context.Context

	BeforeEach(func() {
		DeferCleanup(configtest.SetupConfig())
		channelRepo = tests.CreateMockPodcastChannelRepo()
		episodeRepo = tests.CreateMockPodcastEpisodeRepo()
		ds = &tests.MockDataStore{
			MockedPodcastChannel: channelRepo,
			MockedPodcastEpisode: episodeRepo,
		}
		ctx = request.WithUser(context.Background(), model.User{ID: "admin", IsAdmin: true})

		mockServer = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if strings.HasSuffix(r.URL.Path, ".mp3") {
				w.Header().Set("Content-Type", "audio/mpeg")
				w.Write([]byte("fake audio data"))
				return
			}
			w.Header().Set("Content-Type", "application/rss+xml")
			fmt.Fprint(w, testRSSFeed)
		}))
		DeferCleanup(mockServer.Close)

		conf.Server.DataFolder = GinkgoT().TempDir()
		svc = podcasts.NewPodcastService(ctx, ds, nil, nil)
	})

	Describe("AddChannel", func() {
		It("creates the channel in DB", func() {
			err := svc.AddChannel(ctx, mockServer.URL+"/feed.xml")
			Expect(err).ToNot(HaveOccurred())
			Expect(channelRepo.Data).To(HaveLen(1))
		})

		It("creates episodes from the feed", func() {
			err := svc.AddChannel(ctx, mockServer.URL+"/feed.xml")
			Expect(err).ToNot(HaveOccurred())
			Expect(episodeRepo.Data).To(HaveLen(2))
		})

		It("sets channel title from RSS", func() {
			_ = svc.AddChannel(ctx, mockServer.URL+"/feed.xml")
			for _, ch := range channelRepo.Data {
				Expect(ch.Title).To(Equal("Test Podcast"))
			}
		})

		It("sets channel status to completed", func() {
			_ = svc.AddChannel(ctx, mockServer.URL+"/feed.xml")
			for _, ch := range channelRepo.Data {
				Expect(ch.Status).To(Equal(model.PodcastStatusCompleted))
			}
		})

		It("sets episode status to new", func() {
			_ = svc.AddChannel(ctx, mockServer.URL+"/feed.xml")
			for _, ep := range episodeRepo.Data {
				Expect(ep.Status).To(Equal(model.PodcastStatusNew))
			}
		})

		It("returns error for unreachable URL", func() {
			err := svc.AddChannel(ctx, "http://localhost:0/invalid")
			Expect(err).To(HaveOccurred())
		})
	})

	Describe("RefreshChannels", func() {
		BeforeEach(func() {
			channelRepo.Data["ch-1"] = &model.PodcastChannel{
				ID:  "ch-1",
				URL: mockServer.URL + "/feed.xml",
			}
		})

		It("adds new episodes from feed", func() {
			err := svc.RefreshChannels(ctx)
			Expect(err).ToNot(HaveOccurred())
			Expect(episodeRepo.Data).To(HaveLen(2))
		})

		It("does not duplicate existing episodes", func() {
			episodeRepo.Data["ep-existing"] = &model.PodcastEpisode{
				ID:        "ep-existing",
				ChannelID: "ch-1",
				GUID:      "guid-ep-001",
			}
			err := svc.RefreshChannels(ctx)
			Expect(err).ToNot(HaveOccurred())
			Expect(episodeRepo.Data).To(HaveLen(2)) // 1 existing + 1 new
		})
	})

	Describe("DownloadEpisode", func() {
		var episode *model.PodcastEpisode

		BeforeEach(func() {
			channelRepo.Data["ch-1"] = &model.PodcastChannel{
				ID:    "ch-1",
				Title: "Test Channel",
				URL:   "http://example.com/feed.xml",
			}
			episode = &model.PodcastEpisode{
				ID:           "ep-1",
				ChannelID:    "ch-1",
				EnclosureURL: mockServer.URL + "/audio.mp3",
				Suffix:       "mp3",
				Status:       model.PodcastStatusNew,
			}
			episodeRepo.Data[episode.ID] = episode
		})

		It("immediately sets status to downloading", func() {
			_ = svc.DownloadEpisode(ctx, "ep-1")
			Expect(episodeRepo.Data["ep-1"].Status).To(Equal(model.PodcastStatusDownloading))
		})

		It("creates the audio file at the expected path", func() {
			_ = svc.DownloadEpisode(ctx, "ep-1")
			expectedPath := filepath.Join(conf.Server.DataFolder, "podcasts", "ch-1", "ep-1.mp3")
			Eventually(func() bool {
				_, err := os.Stat(expectedPath)
				return err == nil
			}, "3s").Should(BeTrue())
		})

		It("sets status to completed after download", func() {
			_ = svc.DownloadEpisode(ctx, "ep-1")
			Eventually(func() model.PodcastStatus {
				return episodeRepo.Data["ep-1"].Status
			}, "3s").Should(Equal(model.PodcastStatusCompleted))
		})

		It("records the file path after download", func() {
			_ = svc.DownloadEpisode(ctx, "ep-1")
			expectedPath := filepath.Join(conf.Server.DataFolder, "podcasts", "ch-1", "ep-1.mp3")
			Eventually(func() string {
				return episodeRepo.Data["ep-1"].Path
			}, "3s").Should(Equal(expectedPath))
		})
	})

	Describe("DeleteEpisode", func() {
		It("resets episode to new status", func() {
			episodeRepo.Data["ep-1"] = &model.PodcastEpisode{ID: "ep-1", ChannelID: "ch-1", Status: model.PodcastStatusCompleted}
			err := svc.DeleteEpisode(ctx, "ep-1")
			Expect(err).ToNot(HaveOccurred())
			Expect(episodeRepo.Data["ep-1"].Status).To(Equal(model.PodcastStatusNew))
		})

		It("deletes the downloaded file when path is set", func() {
			tmpFile := filepath.Join(GinkgoT().TempDir(), "ep.mp3")
			Expect(os.WriteFile(tmpFile, []byte("audio"), 0600)).To(Succeed())
			episodeRepo.Data["ep-1"] = &model.PodcastEpisode{ID: "ep-1", Path: tmpFile}

			_ = svc.DeleteEpisode(ctx, "ep-1")
			_, err := os.Stat(tmpFile)
			Expect(os.IsNotExist(err)).To(BeTrue())
		})
	})

	Describe("DeleteChannel", func() {
		It("removes channel from DB", func() {
			channelRepo.Data["ch-1"] = &model.PodcastChannel{ID: "ch-1"}
			err := svc.DeleteChannel(ctx, "ch-1")
			Expect(err).ToNot(HaveOccurred())
			Expect(channelRepo.Data).To(BeEmpty())
		})

		It("deletes all episode files for the channel", func() {
			tmpDir := GinkgoT().TempDir()
			epFile := filepath.Join(tmpDir, "ep.mp3")
			Expect(os.WriteFile(epFile, []byte("audio"), 0600)).To(Succeed())

			channelRepo.Data["ch-1"] = &model.PodcastChannel{ID: "ch-1"}
			episodeRepo.Data["ep-1"] = &model.PodcastEpisode{
				ID: "ep-1", ChannelID: "ch-1",
				Path: epFile,
			}

			_ = svc.DeleteChannel(ctx, "ch-1")
			_, err := os.Stat(epFile)
			Expect(os.IsNotExist(err)).To(BeTrue())
		})
	})

	Describe("DownloadEpisode error handling", func() {
		BeforeEach(func() {
			channelRepo.Data["ch-1"] = &model.PodcastChannel{ID: "ch-1", Title: "Test Channel"}
		})
		It("sets status to error when download fails", func() {
			episodeRepo.Data["ep-bad"] = &model.PodcastEpisode{
				ID:           "ep-bad",
				ChannelID:    "ch-1",
				EnclosureURL: "http://localhost:0/no-such.mp3",
				Suffix:       "mp3",
				Status:       model.PodcastStatusNew,
			}
			_ = svc.DownloadEpisode(ctx, "ep-bad")
			Eventually(func() model.PodcastStatus {
				return episodeRepo.Data["ep-bad"].Status
			}, "3s").Should(Equal(model.PodcastStatusError))
		})

		It("records error message when download fails", func() {
			episodeRepo.Data["ep-bad"] = &model.PodcastEpisode{
				ID:           "ep-bad",
				ChannelID:    "ch-1",
				EnclosureURL: "http://localhost:0/no-such.mp3",
				Suffix:       "mp3",
				Status:       model.PodcastStatusNew,
			}
			_ = svc.DownloadEpisode(ctx, "ep-bad")
			Eventually(func() string {
				return episodeRepo.Data["ep-bad"].ErrorMessage
			}, "3s").ShouldNot(BeEmpty())
		})
	})

	Describe("AddChannel — Podcasting 2.0 field persistence", func() {
		var transcriptRepo *tests.MockPodcastTranscriptRepo
		var personRepo *tests.MockPodcastPersonRepo
		var p20Server *httptest.Server

		BeforeEach(func() {
			transcriptRepo = tests.CreateMockPodcastTranscriptRepo()
			personRepo = tests.CreateMockPodcastPersonRepo()
			ds.MockedPodcastTranscript = transcriptRepo
			ds.MockedPodcastPerson = personRepo

			p20Server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/rss+xml")
				fmt.Fprint(w, testRSSFeedPodcast20)
			}))
			DeferCleanup(p20Server.Close)
		})

		It("stores PodcastGUID from feed", func() {
			Expect(svc.AddChannel(ctx, p20Server.URL+"/feed.xml")).To(Succeed())
			for _, ch := range channelRepo.Data {
				Expect(ch.PodcastGUID).To(Equal("917393e3-1b1e-5cef-ace4-edaa54e1f810"))
			}
		})

		It("stores Locked flag and LockedOwner from feed", func() {
			Expect(svc.AddChannel(ctx, p20Server.URL+"/feed.xml")).To(Succeed())
			for _, ch := range channelRepo.Data {
				Expect(ch.Locked).To(BeTrue())
				Expect(ch.LockedOwner).To(Equal("owner@example.com"))
			}
		})

		It("stores Medium from feed", func() {
			Expect(svc.AddChannel(ctx, p20Server.URL+"/feed.xml")).To(Succeed())
			for _, ch := range channelRepo.Data {
				Expect(ch.Medium).To(Equal("podcast"))
			}
		})

		It("saves funding items to funding repo", func() {
			fundingRepo := tests.CreateMockPodcastFundingRepo()
			ds.MockedPodcastFunding = fundingRepo
			Expect(svc.AddChannel(ctx, p20Server.URL+"/feed.xml")).To(Succeed())
			Expect(fundingRepo.Data).ToNot(BeEmpty())
			var urls []string
			for _, f := range fundingRepo.Data {
				urls = append(urls, f.URL)
			}
			Expect(urls).To(ContainElement("https://example.com/donate"))
		})

		It("stores UpdateFrequency and UpdateRRule from feed", func() {
			Expect(svc.AddChannel(ctx, p20Server.URL+"/feed.xml")).To(Succeed())
			for _, ch := range channelRepo.Data {
				Expect(ch.UpdateFrequency).To(Equal("Weekly"))
				Expect(ch.UpdateRRule).To(Equal("FREQ=WEEKLY"))
			}
		})

		It("saves channel-level podcast:person entries", func() {
			Expect(svc.AddChannel(ctx, p20Server.URL+"/feed.xml")).To(Succeed())
			Expect(personRepo.Data).ToNot(BeEmpty())
			var channelPersons []string
			for _, p := range personRepo.Data {
				if p.ChannelID != "" {
					channelPersons = append(channelPersons, p.Name)
				}
			}
			Expect(channelPersons).To(ConsistOf("Jane Host", "Bob Producer"))
		})

		It("saves episode podcast:transcript entries", func() {
			Expect(svc.AddChannel(ctx, p20Server.URL+"/feed.xml")).To(Succeed())
			Expect(transcriptRepo.Data).ToNot(BeEmpty())
			var mimeTypes []string
			for _, t := range transcriptRepo.Data {
				mimeTypes = append(mimeTypes, t.MimeType)
			}
			Expect(mimeTypes).To(ConsistOf("text/vtt", "application/x-subrip"))
		})

		It("stores transcript language and rel attributes", func() {
			Expect(svc.AddChannel(ctx, p20Server.URL+"/feed.xml")).To(Succeed())
			var vttLanguage, vttRel string
			for _, t := range transcriptRepo.Data {
				if t.MimeType == "text/vtt" {
					vttLanguage = t.Language
					vttRel = t.Rel
				}
			}
			Expect(vttLanguage).To(Equal("en"))
			Expect(vttRel).To(Equal("captions"))
		})

		It("saves episode-level podcast:person entries", func() {
			Expect(svc.AddChannel(ctx, p20Server.URL+"/feed.xml")).To(Succeed())
			var episodePersonNames []string
			for _, p := range personRepo.Data {
				if p.EpisodeID != "" {
					episodePersonNames = append(episodePersonNames, p.Name)
				}
			}
			Expect(episodePersonNames).To(ContainElement("John Guest"))
		})

		It("stores episode ChaptersURL", func() {
			Expect(svc.AddChannel(ctx, p20Server.URL+"/feed.xml")).To(Succeed())
			var chaptersURLs []string
			for _, ep := range episodeRepo.Data {
				if ep.ChaptersURL != "" {
					chaptersURLs = append(chaptersURLs, ep.ChaptersURL)
				}
			}
			Expect(chaptersURLs).To(ContainElement("https://example.com/ep1/chapters.json"))
		})

		It("stores episode Season number and name", func() {
			Expect(svc.AddChannel(ctx, p20Server.URL+"/feed.xml")).To(Succeed())
			var ep1 *model.PodcastEpisode
			for _, ep := range episodeRepo.Data {
				if ep.GUID == "guid-ep-001" {
					ep1 = ep
				}
			}
			Expect(ep1).ToNot(BeNil())
			Expect(ep1.Season).To(Equal(1))
			Expect(ep1.SeasonName).To(Equal("Season One"))
		})

		It("stores episode EpisodeNumber and EpisodeDisplay", func() {
			Expect(svc.AddChannel(ctx, p20Server.URL+"/feed.xml")).To(Succeed())
			var ep1 *model.PodcastEpisode
			for _, ep := range episodeRepo.Data {
				if ep.GUID == "guid-ep-001" {
					ep1 = ep
				}
			}
			Expect(ep1).ToNot(BeNil())
			Expect(ep1.EpisodeNumber).To(Equal("1"))
			Expect(ep1.EpisodeDisplay).To(Equal("Ep.1"))
		})

		It("stores episode Soundbite fields", func() {
			Expect(svc.AddChannel(ctx, p20Server.URL+"/feed.xml")).To(Succeed())
			var ep1 *model.PodcastEpisode
			for _, ep := range episodeRepo.Data {
				if ep.GUID == "guid-ep-001" {
					ep1 = ep
				}
			}
			Expect(ep1).ToNot(BeNil())
			Expect(ep1.SoundbiteStart).To(BeNumerically("~", 73.5, 0.001))
			Expect(ep1.SoundbiteDur).To(BeNumerically("~", 60.0, 0.001))
			Expect(ep1.SoundbiteTitle).To(Equal("Best moment"))
		})
	})

	Describe("DownloadEpisode with timestamp", func() {
		BeforeEach(func() {
			channelRepo.Data["ch-1"] = &model.PodcastChannel{ID: "ch-1", Title: "Test Channel"}
		})
		It("sets updated_at after status change", func() {
			episodeRepo.Data["ep-ts"] = &model.PodcastEpisode{
				ID:           "ep-ts",
				ChannelID:    "ch-1",
				EnclosureURL: mockServer.URL + "/audio.mp3",
				Suffix:       "mp3",
				Status:       model.PodcastStatusNew,
				UpdatedAt:    time.Time{},
			}
			_ = svc.DownloadEpisode(ctx, "ep-ts")
			Eventually(func() bool {
				return !episodeRepo.Data["ep-ts"].UpdatedAt.IsZero()
			}, "3s").Should(BeTrue())
		})
	})

	Describe("AddChannel — Tier 3 field persistence", func() {
		var podrollRepo *tests.MockPodcastPodrollRepo
		var liveItemRepo *tests.MockPodcastLiveItemRepo
		var tier3Server *httptest.Server

		BeforeEach(func() {
			podrollRepo = tests.CreateMockPodcastPodrollRepo()
			liveItemRepo = tests.CreateMockPodcastLiveItemRepo()
			ds.MockedPodcastPodroll = podrollRepo
			ds.MockedPodcastLiveItem = liveItemRepo
		})

		Context("when feed has podcast:podping usesPodping=true", func() {
			BeforeEach(func() {
				tier3Server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					w.Header().Set("Content-Type", "application/rss+xml")
					fmt.Fprint(w, testRSSFeedPodping)
				}))
				DeferCleanup(tier3Server.Close)
			})

			It("stores UsesPodping=true on the channel", func() {
				Expect(svc.AddChannel(ctx, tier3Server.URL+"/feed.xml")).To(Succeed())
				for _, ch := range channelRepo.Data {
					Expect(ch.UsesPodping).To(BeTrue())
				}
			})
		})

		Context("when feed has podcast:podroll", func() {
			BeforeEach(func() {
				tier3Server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					w.Header().Set("Content-Type", "application/rss+xml")
					fmt.Fprint(w, testRSSFeedPodroll)
				}))
				DeferCleanup(tier3Server.Close)
			})

			It("saves podroll items for the channel", func() {
				Expect(svc.AddChannel(ctx, tier3Server.URL+"/feed.xml")).To(Succeed())
				Expect(podrollRepo.Data).ToNot(BeEmpty())
				var urls []string
				for _, item := range podrollRepo.Data {
					urls = append(urls, item.FeedURL)
				}
				Expect(urls).To(ConsistOf(
					"https://example.com/feed.xml",
					"https://other.com/feed.xml",
				))
			})
		})

		Context("when feed has podcast:liveItem", func() {
			BeforeEach(func() {
				tier3Server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					w.Header().Set("Content-Type", "application/rss+xml")
					fmt.Fprint(w, testRSSFeedLiveItem)
				}))
				DeferCleanup(tier3Server.Close)
			})

			It("saves the live item for the channel", func() {
				Expect(svc.AddChannel(ctx, tier3Server.URL+"/feed.xml")).To(Succeed())
				Expect(liveItemRepo.Data).ToNot(BeEmpty())
				for _, li := range liveItemRepo.Data {
					Expect(li.Status).To(Equal("live"))
					Expect(li.GUID).To(Equal("live-guid-001"))
				}
			})
		})
	})

	Describe("RefreshChannels — Tier 3 podping skip", func() {
		var podrollRepo *tests.MockPodcastPodrollRepo
		var liveItemRepo *tests.MockPodcastLiveItemRepo

		BeforeEach(func() {
			podrollRepo = tests.CreateMockPodcastPodrollRepo()
			liveItemRepo = tests.CreateMockPodcastLiveItemRepo()
			ds.MockedPodcastPodroll = podrollRepo
			ds.MockedPodcastLiveItem = liveItemRepo
		})

		It("skips channels with UsesPodping=true during refresh", func() {
			// UsesPodping channel points to a server that would add episodes.
			channelRepo.Data["ch-podping"] = &model.PodcastChannel{
				ID:          "ch-podping",
				URL:         mockServer.URL + "/feed.xml",
				UsesPodping: true,
			}
			initialEpisodeCount := len(episodeRepo.Data)

			Expect(svc.RefreshChannels(ctx)).To(Succeed())
			// No new episodes should be added because the only channel uses podping.
			Expect(episodeRepo.Data).To(HaveLen(initialEpisodeCount))
		})

		It("still refreshes channels with UsesPodping=false", func() {
			channelRepo.Data["ch-normal"] = &model.PodcastChannel{
				ID:          "ch-normal",
				URL:         mockServer.URL + "/feed.xml",
				UsesPodping: false,
			}
			Expect(svc.RefreshChannels(ctx)).To(Succeed())
			// Episodes from the mock feed should have been added.
			Expect(episodeRepo.Data).ToNot(BeEmpty())
		})
	})
})
