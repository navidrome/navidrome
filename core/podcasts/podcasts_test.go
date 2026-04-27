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
		svc = podcasts.NewPodcastService(ds, nil, nil)
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
			Expect(episodeRepo.Data).To(HaveLen(2)) // 기존 1 + 신규 1
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
})
