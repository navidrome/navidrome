package lastfm

import (
	"bytes"
	"context"
	"errors"
	"io"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/navidrome/navidrome/conf"
	"github.com/navidrome/navidrome/core/agents"
	"github.com/navidrome/navidrome/core/scrobbler"
	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/tests"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

const (
	lastfmError3 = `{"error":3,"message":"Invalid Method - No method with that name in this package","links":[]}`
	lastfmError6 = `{"error":6,"message":"The artist you supplied could not be found","links":[]}`
)

var _ = Describe("lastfmAgent", func() {
	var ds model.DataStore
	var ctx context.Context
	BeforeEach(func() {
		ds = &tests.MockDataStore{}
		ctx = context.Background()
	})
	Describe("lastFMConstructor", func() {
		It("uses configured api key and language", func() {
			conf.Server.LastFM.ApiKey = "123"
			conf.Server.LastFM.Secret = "secret"
			conf.Server.LastFM.Language = "pt"
			agent := lastFMConstructor(ds)
			Expect(agent.apiKey).To(Equal("123"))
			Expect(agent.secret).To(Equal("secret"))
			Expect(agent.lang).To(Equal("pt"))
		})
	})

	Describe("GetArtistBiography", func() {
		var agent *lastfmAgent
		var httpClient *tests.FakeHttpClient
		BeforeEach(func() {
			httpClient = &tests.FakeHttpClient{}
			client := newClient("API_KEY", "SECRET", "pt", httpClient)
			agent = lastFMConstructor(ds)
			agent.client = client
		})

		It("returns the biography", func() {
			f, _ := os.Open("tests/fixtures/lastfm.artist.getinfo.json")
			httpClient.Res = http.Response{Body: f, StatusCode: 200}
			Expect(agent.GetArtistBiography(ctx, "123", "U2", "")).To(Equal("U2 é uma das mais importantes bandas de rock de todos os tempos. Formada em 1976 em Dublin, composta por Bono (vocalista  e guitarrista), The Edge (guitarrista, pianista e backing vocal), Adam Clayton (baixista), Larry Mullen, Jr. (baterista e percussionista).\n\nDesde a década de 80, U2 é uma das bandas mais populares no mundo. Seus shows são únicos e um verdadeiro festival de efeitos especiais, além de serem um dos que mais arrecadam anualmente. <a href=\"https://www.last.fm/music/U2\">Read more on Last.fm</a>"))
			Expect(httpClient.RequestCount).To(Equal(1))
			Expect(httpClient.SavedRequest.URL.Query().Get("artist")).To(Equal("U2"))
		})

		It("returns an error if Last.fm call fails", func() {
			httpClient.Err = errors.New("error")
			_, err := agent.GetArtistBiography(ctx, "123", "U2", "")
			Expect(err).To(HaveOccurred())
			Expect(httpClient.RequestCount).To(Equal(1))
			Expect(httpClient.SavedRequest.URL.Query().Get("artist")).To(Equal("U2"))
		})

		It("returns an error if Last.fm call returns an error", func() {
			httpClient.Res = http.Response{Body: io.NopCloser(bytes.NewBufferString(lastfmError3)), StatusCode: 200}
			_, err := agent.GetArtistBiography(ctx, "123", "U2", "")
			Expect(err).To(HaveOccurred())
			Expect(httpClient.RequestCount).To(Equal(1))
			Expect(httpClient.SavedRequest.URL.Query().Get("artist")).To(Equal("U2"))
		})
	})

	Describe("GetSimilarArtists", func() {
		var agent *lastfmAgent
		var httpClient *tests.FakeHttpClient
		BeforeEach(func() {
			httpClient = &tests.FakeHttpClient{}
			client := newClient("API_KEY", "SECRET", "pt", httpClient)
			agent = lastFMConstructor(ds)
			agent.client = client
		})

		It("returns similar artists", func() {
			f, _ := os.Open("tests/fixtures/lastfm.artist.getsimilar.json")
			httpClient.Res = http.Response{Body: f, StatusCode: 200}
			Expect(agent.GetSimilarArtists(ctx, "123", "U2", "", 2)).To(Equal([]agents.Artist{
				{Name: "Passengers", MBID: "e110c11f-1c94-4471-a350-c38f46b29389"},
				{Name: "INXS", MBID: "481bf5f9-2e7c-4c44-b08a-05b32bc7c00d"},
			}))
			Expect(httpClient.RequestCount).To(Equal(1))
			Expect(httpClient.SavedRequest.URL.Query().Get("artist")).To(Equal("U2"))
		})

		It("returns an error if Last.fm call fails", func() {
			httpClient.Err = errors.New("error")
			_, err := agent.GetSimilarArtists(ctx, "123", "U2", "", 2)
			Expect(err).To(HaveOccurred())
			Expect(httpClient.RequestCount).To(Equal(1))
			Expect(httpClient.SavedRequest.URL.Query().Get("artist")).To(Equal("U2"))
		})

		It("returns an error if Last.fm call returns an error", func() {
			httpClient.Res = http.Response{Body: io.NopCloser(bytes.NewBufferString(lastfmError3)), StatusCode: 200}
			_, err := agent.GetSimilarArtists(ctx, "123", "U2", "", 2)
			Expect(err).To(HaveOccurred())
			Expect(httpClient.RequestCount).To(Equal(1))
			Expect(httpClient.SavedRequest.URL.Query().Get("artist")).To(Equal("U2"))
		})
	})

	Describe("GetArtistTopSongs", func() {
		var agent *lastfmAgent
		var httpClient *tests.FakeHttpClient
		BeforeEach(func() {
			httpClient = &tests.FakeHttpClient{}
			client := newClient("API_KEY", "SECRET", "pt", httpClient)
			agent = lastFMConstructor(ds)
			agent.client = client
		})

		It("returns top songs", func() {
			f, _ := os.Open("tests/fixtures/lastfm.artist.gettoptracks.json")
			httpClient.Res = http.Response{Body: f, StatusCode: 200}
			Expect(agent.GetArtistTopSongs(ctx, "123", "U2", "", 2)).To(Equal([]agents.Song{
				{Name: "Beautiful Day", MBID: "f7f264d0-a89b-4682-9cd7-a4e7c37637af"},
				{Name: "With or Without You", MBID: "6b9a509f-6907-4a6e-9345-2f12da09ba4b"},
			}))
			Expect(httpClient.RequestCount).To(Equal(1))
			Expect(httpClient.SavedRequest.URL.Query().Get("artist")).To(Equal("U2"))
		})

		It("returns an error if Last.fm call fails", func() {
			httpClient.Err = errors.New("error")
			_, err := agent.GetArtistTopSongs(ctx, "123", "U2", "", 2)
			Expect(err).To(HaveOccurred())
			Expect(httpClient.RequestCount).To(Equal(1))
			Expect(httpClient.SavedRequest.URL.Query().Get("artist")).To(Equal("U2"))
		})

		It("returns an error if Last.fm call returns an error", func() {
			httpClient.Res = http.Response{Body: io.NopCloser(bytes.NewBufferString(lastfmError3)), StatusCode: 200}
			_, err := agent.GetArtistTopSongs(ctx, "123", "U2", "", 2)
			Expect(err).To(HaveOccurred())
			Expect(httpClient.RequestCount).To(Equal(1))
			Expect(httpClient.SavedRequest.URL.Query().Get("artist")).To(Equal("U2"))
		})
	})

	Describe("Scrobbling", func() {
		var agent *lastfmAgent
		var httpClient *tests.FakeHttpClient
		var track *model.MediaFile
		BeforeEach(func() {
			_ = ds.UserProps(ctx).Put("user-1", sessionKeyProperty, "SK-1")
			httpClient = &tests.FakeHttpClient{}
			client := newClient("API_KEY", "SECRET", "en", httpClient)
			agent = lastFMConstructor(ds)
			agent.client = client
			track = &model.MediaFile{
				ID:             "123",
				Title:          "Track Title",
				Album:          "Track Album",
				Artist:         "Track Artist",
				AlbumArtist:    "Track AlbumArtist",
				TrackNumber:    1,
				Duration:       180,
				MbzRecordingID: "mbz-123",
			}
		})

		Describe("NowPlaying", func() {
			It("calls Last.fm with correct params", func() {
				httpClient.Res = http.Response{Body: io.NopCloser(bytes.NewBufferString("{}")), StatusCode: 200}

				err := agent.NowPlaying(ctx, "user-1", track)

				Expect(err).ToNot(HaveOccurred())
				Expect(httpClient.SavedRequest.Method).To(Equal(http.MethodPost))
				sentParams := httpClient.SavedRequest.URL.Query()
				Expect(sentParams.Get("method")).To(Equal("track.updateNowPlaying"))
				Expect(sentParams.Get("sk")).To(Equal("SK-1"))
				Expect(sentParams.Get("track")).To(Equal(track.Title))
				Expect(sentParams.Get("album")).To(Equal(track.Album))
				Expect(sentParams.Get("artist")).To(Equal(track.Artist))
				Expect(sentParams.Get("albumArtist")).To(Equal(track.AlbumArtist))
				Expect(sentParams.Get("trackNumber")).To(Equal(strconv.Itoa(track.TrackNumber)))
				Expect(sentParams.Get("duration")).To(Equal(strconv.FormatFloat(float64(track.Duration), 'G', -1, 32)))
				Expect(sentParams.Get("mbid")).To(Equal(track.MbzRecordingID))
			})

			It("returns ErrNotAuthorized if user is not linked", func() {
				err := agent.NowPlaying(ctx, "user-2", track)
				Expect(err).To(MatchError(scrobbler.ErrNotAuthorized))
			})
		})

		Describe("scrobble", func() {
			It("calls Last.fm with correct params", func() {
				ts := time.Now()
				httpClient.Res = http.Response{Body: io.NopCloser(bytes.NewBufferString("{}")), StatusCode: 200}

				err := agent.Scrobble(ctx, "user-1", scrobbler.Scrobble{MediaFile: *track, TimeStamp: ts})

				Expect(err).ToNot(HaveOccurred())
				Expect(httpClient.SavedRequest.Method).To(Equal(http.MethodPost))
				sentParams := httpClient.SavedRequest.URL.Query()
				Expect(sentParams.Get("method")).To(Equal("track.scrobble"))
				Expect(sentParams.Get("sk")).To(Equal("SK-1"))
				Expect(sentParams.Get("track")).To(Equal(track.Title))
				Expect(sentParams.Get("album")).To(Equal(track.Album))
				Expect(sentParams.Get("artist")).To(Equal(track.Artist))
				Expect(sentParams.Get("albumArtist")).To(Equal(track.AlbumArtist))
				Expect(sentParams.Get("trackNumber")).To(Equal(strconv.Itoa(track.TrackNumber)))
				Expect(sentParams.Get("duration")).To(Equal(strconv.FormatFloat(float64(track.Duration), 'G', -1, 32)))
				Expect(sentParams.Get("mbid")).To(Equal(track.MbzRecordingID))
				Expect(sentParams.Get("timestamp")).To(Equal(strconv.FormatInt(ts.Unix(), 10)))
			})

			It("skips songs with less than 31 seconds", func() {
				track.Duration = 29
				httpClient.Res = http.Response{Body: io.NopCloser(bytes.NewBufferString("{}")), StatusCode: 200}

				err := agent.Scrobble(ctx, "user-1", scrobbler.Scrobble{MediaFile: *track, TimeStamp: time.Now()})

				Expect(err).ToNot(HaveOccurred())
				Expect(httpClient.SavedRequest).To(BeNil())
			})

			It("returns ErrNotAuthorized if user is not linked", func() {
				err := agent.Scrobble(ctx, "user-2", scrobbler.Scrobble{MediaFile: *track, TimeStamp: time.Now()})
				Expect(err).To(MatchError(scrobbler.ErrNotAuthorized))
			})

			It("returns ErrRetryLater on error 11", func() {
				httpClient.Res = http.Response{
					Body:       io.NopCloser(bytes.NewBufferString(`{"error":11,"message":"Service Offline - This service is temporarily offline. Try again later."}`)),
					StatusCode: 400,
				}

				err := agent.Scrobble(ctx, "user-1", scrobbler.Scrobble{MediaFile: *track, TimeStamp: time.Now()})
				Expect(err).To(MatchError(scrobbler.ErrRetryLater))
			})

			It("returns ErrRetryLater on error 16", func() {
				httpClient.Res = http.Response{
					Body:       io.NopCloser(bytes.NewBufferString(`{"error":16,"message":"There was a temporary error processing your request. Please try again"}`)),
					StatusCode: 400,
				}

				err := agent.Scrobble(ctx, "user-1", scrobbler.Scrobble{MediaFile: *track, TimeStamp: time.Now()})
				Expect(err).To(MatchError(scrobbler.ErrRetryLater))
			})

			It("returns ErrRetryLater on http errors", func() {
				httpClient.Res = http.Response{
					Body:       io.NopCloser(bytes.NewBufferString(`internal server error`)),
					StatusCode: 500,
				}

				err := agent.Scrobble(ctx, "user-1", scrobbler.Scrobble{MediaFile: *track, TimeStamp: time.Now()})
				Expect(err).To(MatchError(scrobbler.ErrRetryLater))
			})

			It("returns ErrUnrecoverable on other errors", func() {
				httpClient.Res = http.Response{
					Body:       io.NopCloser(bytes.NewBufferString(`{"error":8,"message":"Operation failed - Something else went wrong"}`)),
					StatusCode: 400,
				}

				err := agent.Scrobble(ctx, "user-1", scrobbler.Scrobble{MediaFile: *track, TimeStamp: time.Now()})
				Expect(err).To(MatchError(scrobbler.ErrUnrecoverable))
			})
		})
	})

	Describe("GetAlbumInfo", func() {
		var agent *lastfmAgent
		var httpClient *tests.FakeHttpClient
		BeforeEach(func() {
			httpClient = &tests.FakeHttpClient{}
			client := newClient("API_KEY", "SECRET", "pt", httpClient)
			agent = lastFMConstructor(ds)
			agent.client = client
		})

		It("returns the biography", func() {
			f, _ := os.Open("tests/fixtures/lastfm.album.getinfo.json")
			httpClient.Res = http.Response{Body: f, StatusCode: 200}
			Expect(agent.GetAlbumInfo(ctx, "Believe", "Cher", "03c91c40-49a6-44a7-90e7-a700edf97a62")).To(Equal(&agents.AlbumInfo{
				Name:        "Believe",
				MBID:        "03c91c40-49a6-44a7-90e7-a700edf97a62",
				Description: "Believe is the twenty-third studio album by American singer-actress Cher, released on November 10, 1998 by Warner Bros. Records. The RIAA certified it Quadruple Platinum on December 23, 1999, recognizing four million shipments in the United States; Worldwide, the album has sold more than 20 million copies, making it the biggest-selling album of her career. In 1999 the album received three Grammy Awards nominations including \"Record of the Year\", \"Best Pop Album\" and winning \"Best Dance Recording\" for the single \"Believe\". It was released by Warner Bros. Records at the end of 1998. The album was executive produced by Rob <a href=\"https://www.last.fm/music/Cher/Believe\">Read more on Last.fm</a>.",
				URL:         "https://www.last.fm/music/Cher/Believe",
				Images: []agents.ExternalImage{
					{
						URL:  "https://lastfm.freetls.fastly.net/i/u/34s/3b54885952161aaea4ce2965b2db1638.png",
						Size: 34,
					},
					{
						URL:  "https://lastfm.freetls.fastly.net/i/u/64s/3b54885952161aaea4ce2965b2db1638.png",
						Size: 64,
					},
					{
						URL:  "https://lastfm.freetls.fastly.net/i/u/174s/3b54885952161aaea4ce2965b2db1638.png",
						Size: 174,
					},
					{
						URL:  "https://lastfm.freetls.fastly.net/i/u/300x300/3b54885952161aaea4ce2965b2db1638.png",
						Size: 300,
					},
				},
			}))
			Expect(httpClient.RequestCount).To(Equal(1))
			Expect(httpClient.SavedRequest.URL.Query().Get("mbid")).To(Equal("03c91c40-49a6-44a7-90e7-a700edf97a62"))
		})

		It("returns empty images if no images are available", func() {
			f, _ := os.Open("tests/fixtures/lastfm.album.getinfo.empty_urls.json")
			httpClient.Res = http.Response{Body: f, StatusCode: 200}
			Expect(agent.GetAlbumInfo(ctx, "The Definitive Less Damage And More Joy", "The Jesus and Mary Chain", "")).To(Equal(&agents.AlbumInfo{
				Name:   "The Definitive Less Damage And More Joy",
				URL:    "https://www.last.fm/music/The+Jesus+and+Mary+Chain/The+Definitive+Less+Damage+And+More+Joy",
				Images: []agents.ExternalImage{},
			}))
			Expect(httpClient.RequestCount).To(Equal(1))
			Expect(httpClient.SavedRequest.URL.Query().Get("album")).To(Equal("The Definitive Less Damage And More Joy"))
		})

		It("returns an error if Last.fm call fails", func() {
			httpClient.Err = errors.New("error")
			_, err := agent.GetAlbumInfo(ctx, "123", "U2", "mbid-1234")
			Expect(err).To(HaveOccurred())
			Expect(httpClient.RequestCount).To(Equal(1))
			Expect(httpClient.SavedRequest.URL.Query().Get("mbid")).To(Equal("mbid-1234"))
		})

		It("returns an error if Last.fm call returns an error", func() {
			httpClient.Res = http.Response{Body: io.NopCloser(bytes.NewBufferString(lastfmError3)), StatusCode: 200}
			_, err := agent.GetAlbumInfo(ctx, "123", "U2", "mbid-1234")
			Expect(err).To(HaveOccurred())
			Expect(httpClient.RequestCount).To(Equal(1))
			Expect(httpClient.SavedRequest.URL.Query().Get("mbid")).To(Equal("mbid-1234"))
		})

		It("returns an error if Last.fm call returns an error 6 and mbid is empty", func() {
			httpClient.Res = http.Response{Body: io.NopCloser(bytes.NewBufferString(lastfmError6)), StatusCode: 200}
			_, err := agent.GetAlbumInfo(ctx, "123", "U2", "")
			Expect(err).To(HaveOccurred())
			Expect(httpClient.RequestCount).To(Equal(1))
		})

		Context("MBID non existent in Last.fm", func() {
			It("calls again when last.fm returns an error 6", func() {
				httpClient.Res = http.Response{Body: io.NopCloser(bytes.NewBufferString(lastfmError6)), StatusCode: 200}
				_, _ = agent.GetAlbumInfo(ctx, "123", "U2", "mbid-1234")
				Expect(httpClient.RequestCount).To(Equal(2))
				Expect(httpClient.SavedRequest.URL.Query().Get("mbid")).To(BeEmpty())
			})
		})
	})
})
