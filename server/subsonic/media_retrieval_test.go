package subsonic

import (
	"bytes"
	"cmp"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"slices"
	"time"

	"github.com/navidrome/navidrome/conf"
	"github.com/navidrome/navidrome/conf/configtest"
	"github.com/navidrome/navidrome/core/artwork"
	"github.com/navidrome/navidrome/core/lyrics"
	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/server"
	"github.com/navidrome/navidrome/server/subsonic/responses"
	"github.com/navidrome/navidrome/tests"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("MediaRetrievalController", func() {
	var router *Router
	var ds model.DataStore
	mockRepo := &mockedMediaFile{MockMediaFileRepo: tests.MockMediaFileRepo{}}
	var artwork *fakeArtwork
	var w *httptest.ResponseRecorder

	BeforeEach(func() {
		ds = &tests.MockDataStore{
			MockedMediaFile: mockRepo,
		}
		artwork = &fakeArtwork{data: "image data"}
		router = New(ds, artwork, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, lyrics.NewLyrics(nil), nil, nil, nil)
		w = httptest.NewRecorder()
		DeferCleanup(configtest.SetupConfig())
		conf.Server.LyricsPriority = "embedded,.lrc"
	})

	Describe("GetCoverArt", func() {
		It("should return data for that id", func() {
			r := newGetRequest("id=34", "size=128", "square=true")
			_, err := router.GetCoverArt(w, r)

			Expect(err).ToNot(HaveOccurred())
			Expect(artwork.recvSize).To(Equal(128))
			Expect(artwork.recvSquare).To(BeTrue())
			Expect(w.Body.String()).To(Equal(artwork.data))
		})

		It("should return placeholder if id parameter is missing (mimicking Subsonic)", func() {
			r := newGetRequest() // No id parameter
			_, err := router.GetCoverArt(w, r)

			Expect(err).To(BeNil())
			Expect(artwork.recvId).To(BeEmpty())
			Expect(w.Body.String()).To(Equal(artwork.data))
		})

		It("should fail when the file is not found", func() {
			artwork.err = model.ErrNotFound
			r := newGetRequest("id=34", "size=128", "square=true")
			_, err := router.GetCoverArt(w, r)

			Expect(err).To(MatchError("Artwork not found"))
		})

		It("should fail when there is an unknown error", func() {
			artwork.err = errors.New("weird error")
			r := newGetRequest("id=34", "size=128")
			_, err := router.GetCoverArt(w, r)

			Expect(err).To(MatchError("weird error"))
		})

		When("artwork throttle is enabled", func() {
			BeforeEach(func() {
				conf.Server.DevArtworkMaxRequests = 1
				conf.Server.DevArtworkThrottleBacklogLimit = 0
				conf.Server.DevArtworkThrottleBacklogTimeout = time.Second
				router.artworkThrottle = server.NewRequestThrottle()
			})

			It("should return data when throttle token is available", func() {
				r := newGetRequest("id=34", "size=128")
				_, err := router.GetCoverArt(w, r)

				Expect(err).ToNot(HaveOccurred())
				Expect(w.Body.String()).To(Equal(artwork.data))
			})

			It("should return 429 when throttle capacity is exceeded", func() {
				// Hold the only token by starting a DoBuffered call that blocks
				held := make(chan struct{})
				done := make(chan struct{})
				go func() {
					defer close(done)
					_, _, _ = router.artworkThrottle.DoBuffered(context.Background(), func() (io.ReadCloser, time.Time, error) {
						close(held) // signal that we're holding the token
						time.Sleep(2 * time.Second)
						return io.NopCloser(bytes.NewReader(nil)), time.Time{}, nil
					})
				}()
				<-held // wait until token is held

				r := newGetRequest("id=34", "size=128")
				_, err := router.GetCoverArt(w, r)

				Expect(err).ToNot(HaveOccurred())
				Expect(w.Code).To(Equal(http.StatusTooManyRequests))
				Eventually(done, 3*time.Second).Should(BeClosed())
			})

			It("should not starve concurrent requests when a client is slow to read (regression)", func() {
				// Reproduces the starvation scenario from the bug report: with only 1
				// throttle token, a slow client that blocks during response writing must
				// NOT prevent other requests from being served. The fix buffers the image
				// and releases the token before writing to the client.
				conf.Server.DevArtworkMaxRequests = 1
				conf.Server.DevArtworkThrottleBacklogLimit = 1
				conf.Server.DevArtworkThrottleBacklogTimeout = 5 * time.Second
				router.artworkThrottle = server.NewRequestThrottle()

				unblocked := make(chan struct{})
				slow := &slowResponseWriter{
					ResponseRecorder: httptest.NewRecorder(),
					unblocked:        unblocked,
				}

				// Start a request that will stall on the slow writer
				reqDone := make(chan struct{})
				go func() {
					defer close(reqDone)
					r := newGetRequest("id=slow", "size=128")
					_, _ = router.GetCoverArt(slow, r)
				}()

				// A concurrent request should succeed while the first is still writing
				Eventually(func() int {
					w2 := httptest.NewRecorder()
					r2 := newGetRequest("id=concurrent", "size=64")
					_, _ = router.GetCoverArt(w2, r2)
					return w2.Code
				}, 2*time.Second, 10*time.Millisecond).Should(Equal(http.StatusOK))

				close(unblocked)
				Eventually(reqDone, 2*time.Second).Should(BeClosed())
			})
		})

		When("artwork throttle is disabled (nil)", func() {
			It("should still return data normally", func() {
				router.artworkThrottle = nil
				r := newGetRequest("id=34", "size=128")
				_, err := router.GetCoverArt(w, r)

				Expect(err).ToNot(HaveOccurred())
				Expect(w.Body.String()).To(Equal(artwork.data))
			})
		})

		When("client disconnects (context is cancelled)", func() {
			It("should not call the service if cancelled before the call", func() {
				// Create a request
				ctx, cancel := context.WithCancel(context.Background())
				r := newGetRequest("id=34", "size=128", "square=true")
				r = r.WithContext(ctx)
				cancel() // Cancel the context before the call

				// Call the GetCoverArt method
				_, err := router.GetCoverArt(w, r)

				// Expect no error and no call to the artwork service
				Expect(err).ToNot(HaveOccurred())
				Expect(artwork.recvId).To(Equal(""))
				Expect(artwork.recvSize).To(Equal(0))
				Expect(artwork.recvSquare).To(BeFalse())
				Expect(w.Body.String()).To(BeEmpty())
			})

			It("should not return data if cancelled during the call", func() {
				// Create a request with a context that will be cancelled
				ctx, cancel := context.WithCancel(context.Background())
				defer cancel() // Ensure the context is cancelled after the test (best practices)
				r := newGetRequest("id=34", "size=128", "square=true")
				r = r.WithContext(ctx)
				artwork.ctxCancelFunc = cancel // Set the cancel function to simulate cancellation in the service

				// Call the GetCoverArt method
				_, err := router.GetCoverArt(w, r)

				// Expect no error and the service to have been called
				Expect(err).ToNot(HaveOccurred())
				Expect(artwork.recvId).To(Equal("34"))
				Expect(artwork.recvSize).To(Equal(128))
				Expect(artwork.recvSquare).To(BeTrue())
				Expect(w.Body.String()).To(BeEmpty())
			})
		})
	})

	Describe("GetLyrics", func() {
		It("should return data for given artist & title", func() {
			r := newGetRequest("artist=Rick+Astley", "title=Never+Gonna+Give+You+Up")
			lyrics, _ := model.ToLyrics("eng", "[00:18.80]We're no strangers to love\n[00:22.80]You know the rules and so do I")
			lyricsJson, err := json.Marshal(model.LyricList{
				*lyrics,
			})
			Expect(err).ToNot(HaveOccurred())

			baseTime := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
			mockRepo.SetData(model.MediaFiles{
				{
					ID:        "2",
					Artist:    "Rick Astley",
					Title:     "Never Gonna Give You Up",
					Lyrics:    "[]",
					UpdatedAt: baseTime.Add(2 * time.Hour), // No lyrics, newer
				},
				{
					ID:        "1",
					Artist:    "Rick Astley",
					Title:     "Never Gonna Give You Up",
					Lyrics:    string(lyricsJson),
					UpdatedAt: baseTime.Add(1 * time.Hour), // Has lyrics, older
				},
				{
					ID:        "3",
					Artist:    "Rick Astley",
					Title:     "Never Gonna Give You Up",
					Lyrics:    "[]",
					UpdatedAt: baseTime.Add(3 * time.Hour), // No lyrics, newest
				},
			})
			response, err := router.GetLyrics(r)
			Expect(err).ToNot(HaveOccurred())
			Expect(response.Lyrics.Artist).To(Equal("Rick Astley"))
			Expect(response.Lyrics.Title).To(Equal("Never Gonna Give You Up"))
			Expect(response.Lyrics.Value).To(Equal("We're no strangers to love\nYou know the rules and so do I\n"))
		})
		It("should return empty subsonic response if the record corresponding to the given artist & title is not found", func() {
			r := newGetRequest("artist=Dheeraj", "title=Rinkiya+Ke+Papa")
			mockRepo.SetData(model.MediaFiles{})
			response, err := router.GetLyrics(r)
			Expect(err).ToNot(HaveOccurred())
			Expect(response.Lyrics.Artist).To(Equal(""))
			Expect(response.Lyrics.Title).To(Equal(""))
			Expect(response.Lyrics.Value).To(Equal(""))
		})
		It("should return lyric file when finding mediafile with no embedded lyrics but present on filesystem", func() {
			r := newGetRequest("artist=Rick+Astley", "title=Never+Gonna+Give+You+Up")
			mockRepo.SetData(model.MediaFiles{
				{
					Path:   "tests/fixtures/test.mp3",
					ID:     "1",
					Artist: "Rick Astley",
					Title:  "Never Gonna Give You Up",
				},
				{
					Path:   "tests/fixtures/test.mp3",
					ID:     "2",
					Artist: "Rick Astley",
					Title:  "Never Gonna Give You Up",
				},
			})
			response, err := router.GetLyrics(r)
			Expect(err).ToNot(HaveOccurred())
			Expect(response.Lyrics.Artist).To(Equal("Rick Astley"))
			Expect(response.Lyrics.Title).To(Equal("Never Gonna Give You Up"))
			Expect(response.Lyrics.Value).To(Equal("We're no strangers to love\nYou know the rules and so do I\n"))
		})
	})

	Describe("GetLyricsBySongId", func() {
		const syncedLyrics = "[00:18.80]We're no strangers to love\n[00:22.801]You know the rules and so do I"
		const unsyncedLyrics = "We're no strangers to love\nYou know the rules and so do I"
		const metadata = "[ar:Rick Astley]\n[ti:That one song]\n[offset:-100]"
		var times = []int64{18800, 22801}

		compareResponses := func(actual *responses.LyricsList, expected responses.LyricsList) {
			Expect(actual).ToNot(BeNil())
			Expect(actual.StructuredLyrics).To(HaveLen(len(expected.StructuredLyrics)))
			for i, realLyric := range actual.StructuredLyrics {
				expectedLyric := expected.StructuredLyrics[i]

				Expect(realLyric.DisplayArtist).To(Equal(expectedLyric.DisplayArtist))
				Expect(realLyric.DisplayTitle).To(Equal(expectedLyric.DisplayTitle))
				Expect(realLyric.Lang).To(Equal(expectedLyric.Lang))
				Expect(realLyric.Synced).To(Equal(expectedLyric.Synced))

				if expectedLyric.Offset == nil {
					Expect(realLyric.Offset).To(BeNil())
				} else {
					Expect(*realLyric.Offset).To(Equal(*expectedLyric.Offset))
				}

				Expect(realLyric.Line).To(HaveLen(len(expectedLyric.Line)))
				for j, realLine := range realLyric.Line {
					expectedLine := expectedLyric.Line[j]
					Expect(realLine.Value).To(Equal(expectedLine.Value))

					if expectedLine.Start == nil {
						Expect(realLine.Start).To(BeNil())
					} else {
						Expect(*realLine.Start).To(Equal(*expectedLine.Start))
					}
				}
			}
		}

		It("should return mixed lyrics", func() {
			r := newGetRequest("id=1")
			synced, _ := model.ToLyrics("eng", syncedLyrics)
			unsynced, _ := model.ToLyrics("xxx", unsyncedLyrics)
			lyricsJson, err := json.Marshal(model.LyricList{
				*synced, *unsynced,
			})
			Expect(err).ToNot(HaveOccurred())

			mockRepo.SetData(model.MediaFiles{
				{
					ID:     "1",
					Artist: "Rick Astley",
					Title:  "Never Gonna Give You Up",
					Lyrics: string(lyricsJson),
				},
			})

			response, err := router.GetLyricsBySongId(r)
			Expect(err).ToNot(HaveOccurred())
			compareResponses(response.LyricsList, responses.LyricsList{
				StructuredLyrics: responses.StructuredLyrics{
					{
						Lang:          "eng",
						DisplayArtist: "Rick Astley",
						DisplayTitle:  "Never Gonna Give You Up",
						Synced:        true,
						Line: []responses.Line{
							{
								Start: &times[0],
								Value: "We're no strangers to love",
							},
							{
								Start: &times[1],
								Value: "You know the rules and so do I",
							},
						},
					},
					{
						Lang:          "xxx",
						DisplayArtist: "Rick Astley",
						DisplayTitle:  "Never Gonna Give You Up",
						Synced:        false,
						Line: []responses.Line{
							{
								Value: "We're no strangers to love",
							},
							{
								Value: "You know the rules and so do I",
							},
						},
					},
				},
			})
		})

		It("should parse lrc metadata", func() {
			r := newGetRequest("id=1")
			synced, _ := model.ToLyrics("eng", metadata+"\n"+syncedLyrics)
			lyricsJson, err := json.Marshal(model.LyricList{
				*synced,
			})
			Expect(err).ToNot(HaveOccurred())
			mockRepo.SetData(model.MediaFiles{
				{
					ID:     "1",
					Artist: "Rick Astley",
					Title:  "Never Gonna Give You Up",
					Lyrics: string(lyricsJson),
				},
			})

			response, err := router.GetLyricsBySongId(r)
			Expect(err).ToNot(HaveOccurred())

			offset := int64(-100)
			compareResponses(response.LyricsList, responses.LyricsList{
				StructuredLyrics: responses.StructuredLyrics{
					{
						DisplayArtist: "Rick Astley",
						DisplayTitle:  "That one song",
						Lang:          "eng",
						Synced:        true,
						Line: []responses.Line{
							{
								Start: &times[0],
								Value: "We're no strangers to love",
							},
							{
								Start: &times[1],
								Value: "You know the rules and so do I",
							},
						},
						Offset: &offset,
					},
				},
			})
		})
	})
})

type fakeArtwork struct {
	artwork.Artwork
	data          string
	err           error
	ctxCancelFunc func()
	recvId        string
	recvSize      int
	recvSquare    bool
}

func (c *fakeArtwork) GetOrPlaceholder(_ context.Context, id string, size int, square bool) (io.ReadCloser, time.Time, error) {
	if c.err != nil {
		return nil, time.Time{}, c.err
	}
	c.recvId = id
	c.recvSize = size
	c.recvSquare = square
	if c.ctxCancelFunc != nil {
		c.ctxCancelFunc()
		return nil, time.Time{}, context.Canceled
	}
	return io.NopCloser(bytes.NewReader([]byte(c.data))), time.Time{}, nil
}

// slowResponseWriter simulates a client that stalls during response reading.
// Write blocks until unblocked is closed, reproducing the starvation scenario.
type slowResponseWriter struct {
	*httptest.ResponseRecorder
	unblocked chan struct{}
}

func (w *slowResponseWriter) Write(p []byte) (int, error) {
	<-w.unblocked
	return w.ResponseRecorder.Write(p)
}

type mockedMediaFile struct {
	tests.MockMediaFileRepo
}

func (m *mockedMediaFile) GetAll(opts ...model.QueryOptions) (model.MediaFiles, error) {
	data, err := m.MockMediaFileRepo.GetAll(opts...)
	if err != nil {
		return nil, err
	}
	if len(opts) == 0 || opts[0].Sort != "lyrics, updated_at" {
		return data, nil
	}

	// Hardcoded support for lyrics sorting
	result := slices.Clone(data)
	// Sort by presence of lyrics, then by updated_at. Respect the order specified in opts.
	slices.SortFunc(result, func(a, b model.MediaFile) int {
		diff := cmp.Or(
			cmp.Compare(a.Lyrics, b.Lyrics),
			cmp.Compare(a.UpdatedAt.Unix(), b.UpdatedAt.Unix()),
		)
		if opts[0].Order == "desc" {
			return -diff
		}
		return diff
	})
	return result, nil
}
