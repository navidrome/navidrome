package core

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path"
	"strings"
	"time"

	"github.com/mmcdole/gofeed"
	"github.com/navidrome/navidrome/conf"
	"github.com/navidrome/navidrome/conf/configtest"
	"github.com/navidrome/navidrome/consts"
	"github.com/navidrome/navidrome/log"
	"github.com/navidrome/navidrome/model"
	_ "github.com/navidrome/navidrome/scanner/metadata/taglib"
	"github.com/navidrome/navidrome/tests"
	"github.com/navidrome/navidrome/utils/gg"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("PodcastManager", func() {
	var tmpDir string
	var client *http.Client
	var parser *gofeed.Parser
	var mockPodcast *tests.MockedPodcastRepo
	var mockEpisode *tests.MockedPodcastEpisodeRepo

	var ds model.DataStore
	var ch chan string
	var del chan deleteReq
	var manager podcastManager
	var server *httptest.Server
	ctx := log.NewContext(context.TODO())

	var url string
	var body io.ReadCloser
	var status int

	BeforeEach(func() {
		tmpDir, _ = os.MkdirTemp("", "podcasts")
		DeferCleanup(func() {
			configtest.SetupConfig()
			_ = os.RemoveAll(tmpDir)
		})
		conf.Server.PodcastFolder = tmpDir
		conf.Server.Scanner.Extractor = "taglib"

		server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			url = r.URL.String()
			w.WriteHeader(status)
			_, _ = io.Copy(w, body)
		}))

		client = server.Client()

		parser = gofeed.NewParser()
		parser.Client = client

		mockPodcast = tests.CreateMockPodcastRepo()
		mockEpisode = tests.CreateMockedPodcastEpisodeRepo()

		ds = &tests.MockDataStore{
			MockedPodcast: mockPodcast, MockedPodcastEpisode: mockEpisode,
		}
		ch = make(chan string, 1)
		del = make(chan deleteReq, 1)
		manager = podcastManager{
			ch:     ch,
			client: client,
			del:    del,
			ds:     ds,
			parser: parser,
		}
		status = 200
	})

	AfterEach(func() {
		defer server.Close()
	})

	Describe("CreateFeed", func() {
		It("should fail if fetch errors out", func() {
			status = 404
			body = io.NopCloser(strings.NewReader("Not Found"))

			podcast, err := manager.CreateFeed(ctx, server.URL+"/404.rss")
			Expect(url).To(Equal("/404.rss"))
			Expect(err).To(Equal(gofeed.HTTPError{StatusCode: 404, Status: "404 Not Found"}))
			Expect(podcast).To(BeNil())
		})

		It("should fail if not valid feed", func() {
			body = io.NopCloser(strings.NewReader("garbage"))

			podcast, err := manager.CreateFeed(ctx, server.URL+"/bad.rss")
			Expect(url).To(Equal("/bad.rss"))
			Expect(err).To(Equal(gofeed.ErrFeedTypeNotDetected))
			Expect(podcast).To(BeNil())
		})

		It("should handle XML feed", func() {
			f, _ := os.Open("tests/fixtures/example.podcastFeed.xml")
			body = f

			podcast, err := manager.CreateFeed(ctx, server.URL+"/good.rss")
			id := podcast.ID
			Expect(url).To(Equal("/good.rss"))
			Expect(err).To(BeNil())
			Expect(podcast.ID).ToNot(BeEmpty())

			file, err := os.Stat(path.Join(tmpDir, podcast.ID))
			Expect(err).To(BeNil())
			Expect(file.IsDir()).To(BeTrue())

			podcast.ID = ""
			Expect(*podcast).To(Equal(model.Podcast{
				Url:         server.URL + "/good.rss",
				Title:       "Dafna's Zebra Podcast",
				Description: "A pet-owner's guide to the popular striped equine.",
				ImageUrl:    "https://www.example.com/podcasts/dafnas-zebras/img/dafna-zebra-pod-logo.jpg",
				State:       consts.PodcastStatusNew,
			}))

			episodes, _ := ds.PodcastEpisode(ctx).GetAll()
			for i := range episodes {
				Expect(episodes[i].ID).ToNot(BeEmpty())
				Expect(episodes[i].PodcastId).To(Equal(id))
				episodes[i].ID = ""
				episodes[i].PodcastId = ""
			}

			Expect(episodes).To(BeComparableTo(model.PodcastEpisodes{
				{
					Guid:        "dzpodtop10",
					Url:         "https://www.example.com/podcasts/dafnas-zebras/audio/toptenmyths.mp3",
					Title:       "Top 10 myths about caring for a zebra",
					Description: "Here are the top 10 misunderstandings about the care, feeding, and breeding of these lovable striped animals.",
					PublishDate: gg.P(time.Date(2017, 3, 14, 12, 0, 0, 0, time.UTC)),
					Duration:    1830,
					Suffix:      "mp3",
					Size:        34216300,
					State:       "skipped",
				},
				{
					Guid:        "dzpodclean",
					Url:         "https://www.example.com/podcasts/dafnas-zebras/audio/cleanstripes.mp3",
					Title:       "Keeping those stripes neat and clean",
					Description: "Keeping your zebra clean is time consuming, but worth the effort.",
					ImageUrl:    "https://example.org/episode.jpg",
					PublishDate: gg.P(time.Date(2017, 2, 24, 12, 0, 0, 0, time.UTC)),
					Duration:    1390,
					Suffix:      "mp3",
					Size:        26004388,
					BitRate:     0,
					State:       "skipped",
				},
			}))
		})
	})

	Describe("channel operations", func() {
		It("should enqueue podcast delete request", func() {
			err := manager.DeletePodcast(ctx, "1234")
			Expect(err).To(BeNil())

			item := <-del
			Expect(item).To(Equal(deleteReq{id: "1234", isPodcast: true}))
		})

		It("should enqueue podcast episode delete request", func() {
			err := manager.DeletePodcastEpisode(ctx, "1234")
			Expect(err).To(BeNil())

			item := <-del
			Expect(item).To(Equal(deleteReq{id: "1234", isPodcast: false}))
		})

		It("should enqueue empty string on refresh", func() {
			err := manager.Refresh(ctx)
			Expect(err).To(BeNil())
			item := <-ch
			Expect(item).To(BeEmpty())
		})

		It("should enqueue id string on download", func() {
			err := manager.Download(ctx, "1234")
			Expect(err).To(BeNil())
			item := <-ch
			Expect(item).To(Equal("1234"))
		})
	})

	Context("With existing hierarchy", func() {
		var channel model.Podcast
		var episode model.PodcastEpisode

		BeforeEach(func() {
			channel = model.Podcast{
				ID:          "1234",
				Url:         server.URL + "/good.rss",
				Title:       "Dafna's Zebra Podcast",
				Description: "A pet-owner's guide to the popular striped equine.",
				ImageUrl:    "https://www.example.com/podcasts/dafnas-zebras/img/dafna-zebra-pod-logo.jpg",
			}
			_ = ds.Podcast(ctx).Put(&channel)

			episode = model.PodcastEpisode{
				ID:          "5678",
				PodcastId:   "1234",
				Guid:        "dzpodclean",
				Url:         "https://www.example.com/podcasts/dafnas-zebras/audio/cleanstripes.mp3",
				Title:       "Keeping those stripes neat and clean",
				Description: "Keeping your zebra clean is time consuming, but worth the effort.",
				ImageUrl:    "https://example.org/episode.jpg",
				PublishDate: gg.P(time.Date(2017, 2, 24, 12, 0, 0, 0, time.UTC)),
				Duration:    1390,
				Suffix:      "mp3",
				Size:        26004388,
				BitRate:     0,
				State:       "skipped",
			}
			_ = ds.PodcastEpisode(ctx).Put(&episode)

			mockEpisode.Cleaned = false
			mockPodcast.Cleaned = false

			_ = os.MkdirAll(channel.AbsolutePath(), os.ModePerm)
		})

		Describe("download podcast", func() {
			BeforeEach(func() {
				episode.Url = server.URL + "/example.mp3"
			})

			It("should successfully download file", func() {
				f, _ := os.Open("tests/fixtures/test.mp3")
				body = f

				err := manager.downloadPodcast(ctx, episode.ID)
				Expect(err).To(BeNil())

				Expect(episode).To(Equal(model.PodcastEpisode{
					ID:          "5678",
					Guid:        "dzpodclean",
					PodcastId:   "1234",
					Url:         server.URL + "/example.mp3",
					Title:       "Keeping those stripes neat and clean",
					Description: "Keeping your zebra clean is time consuming, but worth the effort.",
					ImageUrl:    "https://example.org/episode.jpg",
					PublishDate: episode.PublishDate,
					Duration:    1.0199999809265137,
					Suffix:      "mp3",
					Size:        51876,
					BitRate:     192,
					State:       "completed",
				}))

				file, err := os.Stat(episode.AbsolutePath())
				Expect(err).To(BeNil())
				Expect(file.Size()).To(Equal(int64(51876)))
			})

			It("should handle failure on downloading", func() {
				status = 404
				err := manager.downloadPodcast(ctx, episode.ID)

				Expect(err).To(HaveOccurred())
				Expect(episode.State).To(Equal(consts.PodcastStatusError))
				Expect(episode.Error).To(Equal("http error: 404 Not Found"))
			})
		})

		Describe("delete podcast", func() {
			It("will cleanup all resources", func() {
				err := manager.deletePodcast(ctx, channel.ID)
				Expect(err).To(BeNil())

				file, err := os.Stat(channel.AbsolutePath())
				Expect(err).To(HaveOccurred())
				Expect(file).To(BeNil())

				Expect(mockPodcast.Cleaned).To(BeTrue())
				Expect(mockEpisode.Cleaned).To(BeTrue())

				channels, _ := mockPodcast.GetAll(false)
				Expect(channels).To(BeEmpty())

				// We don't expect podcast episode to be empty, because
				// this is handled in reality with foreign keys
			})

			It("will cleanup resources even if folder does not exist", func() {
				_ = os.RemoveAll(channel.AbsolutePath())

				err := manager.deletePodcast(ctx, channel.ID)
				Expect(err).To(BeNil())

				Expect(mockPodcast.Cleaned).To(BeTrue())
				Expect(mockEpisode.Cleaned).To(BeTrue())

				channels, _ := mockPodcast.GetAll(false)
				Expect(channels).To(BeEmpty())
			})
		})

		Describe("delete podcast episode", func() {
			AfterEach(func() {
				_ = os.Remove(episode.AbsolutePath())
			})

			It("will delete file", func() {
				episode.State = consts.PodcastStatusCompleted
				_, err := os.Create(episode.AbsolutePath())
				Expect(err).To(BeNil())

				err = manager.deleteEpisode(ctx, episode.ID)
				Expect(err).To(BeNil())

				file, err := os.Stat(episode.AbsolutePath())
				Expect(err).To(HaveOccurred())
				Expect(file).To(BeNil())

				Expect(episode.State).To(Equal(consts.PodcastStatusDeleted))
			})

			It("will do nothing if file is not completed", func() {
				_, _ = os.Create(episode.AbsolutePath())

				err := manager.deleteEpisode(ctx, episode.ID)
				Expect(err).To(BeNil())

				file, err := os.Stat(episode.AbsolutePath())
				Expect(err).To(BeNil())
				Expect(file).ToNot(BeNil())
			})

			It("will update state even if file does not exist", func() {
				episode.State = consts.PodcastStatusCompleted

				err := manager.deleteEpisode(ctx, episode.ID)
				Expect(err).To(BeNil())

				Expect(episode.State).To(Equal(consts.PodcastStatusDeleted))
			})
		})

		Describe("refresh", func() {
			BeforeEach(func() {
				f, _ := os.Open("tests/fixtures/example.podcastFeed.xml")
				body = f
			})

			var addition model.PodcastEpisode

			BeforeEach(func() {
				addition = model.PodcastEpisode{
					PodcastId:   "1234",
					Guid:        "dzpodtop10",
					Url:         "https://www.example.com/podcasts/dafnas-zebras/audio/toptenmyths.mp3",
					Title:       "Top 10 myths about caring for a zebra",
					Description: "Here are the top 10 misunderstandings about the care, feeding, and breeding of these lovable striped animals.",
					PublishDate: gg.P(time.Date(2017, 3, 14, 12, 0, 0, 0, time.UTC)),
					Duration:    1830,
					Suffix:      "mp3",
					Size:        34216300,
					State:       "skipped",
				}
			})

			It("should add existing episode and clear error", func() {
				channel.Error = "error"
				channel.State = consts.PodcastStatusError

				manager.refreshPodcasts(ctx)

				episodes, _ := ds.PodcastEpisode(ctx).GetAll()
				Expect(episodes).To(HaveLen(2))
				addition.ID = episodes[1].ID
				Expect(episodes).To(BeComparableTo(model.PodcastEpisodes{
					episode, addition,
				}))

				podcast, _ := ds.Podcast(ctx).Get(channel.ID, false)
				Expect(podcast.Error).To(BeEmpty())
				Expect(podcast.State).To(BeEmpty())
			})

			It("should refresh single podcast", func() {
				err := manager.refreshPodcast(ctx, &channel)
				Expect(err).To(BeNil())

				episodes, _ := ds.PodcastEpisode(ctx).GetAll()
				Expect(episodes).To(HaveLen(2))
				addition.ID = episodes[1].ID
				Expect(episodes).To(BeComparableTo(model.PodcastEpisodes{
					episode, addition,
				}))

				f, _ := os.Open("tests/fixtures/example.podcastFeed.xml")
				body = f

				// Second refresh'; should have no effect
				err = manager.refreshPodcast(ctx, &channel)
				Expect(err).To(BeNil())
				Expect(episodes).To(HaveLen(2))
				addition.ID = episodes[1].ID
				Expect(episodes).To(BeComparableTo(model.PodcastEpisodes{
					episode, addition,
				}))
			})

			It("should update error if failing to refresh", func() {
				status = 404
				manager.refreshPodcasts(ctx)

				podcast, _ := ds.Podcast(ctx).Get(channel.ID, false)

				Expect(podcast.State).To(Equal(consts.PodcastStatusError))
				Expect(podcast.Error).To(Equal("http error: 404 Not Found"))

				episodes, _ := ds.PodcastEpisode(ctx).GetAll()
				Expect(episodes).To(Equal(model.PodcastEpisodes{episode}))
			})
		})
	})
})
