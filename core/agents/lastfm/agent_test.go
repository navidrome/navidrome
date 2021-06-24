package lastfm

import (
	"bytes"
	"context"
	"errors"
	"io/ioutil"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/navidrome/navidrome/core/scrobbler"

	"github.com/navidrome/navidrome/model/request"

	"github.com/navidrome/navidrome/model"

	"github.com/navidrome/navidrome/conf"
	"github.com/navidrome/navidrome/core/agents"
	"github.com/navidrome/navidrome/tests"
	. "github.com/onsi/ginkgo"
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

	Describe("GetBiography", func() {
		var agent *lastfmAgent
		var httpClient *tests.FakeHttpClient
		BeforeEach(func() {
			httpClient = &tests.FakeHttpClient{}
			client := NewClient("API_KEY", "SECRET", "pt", httpClient)
			agent = lastFMConstructor(ds)
			agent.client = client
		})

		It("returns the biography", func() {
			f, _ := os.Open("tests/fixtures/lastfm.artist.getinfo.json")
			httpClient.Res = http.Response{Body: f, StatusCode: 200}
			Expect(agent.GetBiography(ctx, "123", "U2", "mbid-1234")).To(Equal("U2 é uma das mais importantes bandas de rock de todos os tempos. Formada em 1976 em Dublin, composta por Bono (vocalista  e guitarrista), The Edge (guitarrista, pianista e backing vocal), Adam Clayton (baixista), Larry Mullen, Jr. (baterista e percussionista).\n\nDesde a década de 80, U2 é uma das bandas mais populares no mundo. Seus shows são únicos e um verdadeiro festival de efeitos especiais, além de serem um dos que mais arrecadam anualmente. <a href=\"https://www.last.fm/music/U2\">Read more on Last.fm</a>"))
			Expect(httpClient.RequestCount).To(Equal(1))
			Expect(httpClient.SavedRequest.URL.Query().Get("mbid")).To(Equal("mbid-1234"))
		})

		It("returns an error if Last.FM call fails", func() {
			httpClient.Err = errors.New("error")
			_, err := agent.GetBiography(ctx, "123", "U2", "mbid-1234")
			Expect(err).To(HaveOccurred())
			Expect(httpClient.RequestCount).To(Equal(1))
			Expect(httpClient.SavedRequest.URL.Query().Get("mbid")).To(Equal("mbid-1234"))
		})

		It("returns an error if Last.FM call returns an error", func() {
			httpClient.Res = http.Response{Body: ioutil.NopCloser(bytes.NewBufferString(lastfmError3)), StatusCode: 200}
			_, err := agent.GetBiography(ctx, "123", "U2", "mbid-1234")
			Expect(err).To(HaveOccurred())
			Expect(httpClient.RequestCount).To(Equal(1))
			Expect(httpClient.SavedRequest.URL.Query().Get("mbid")).To(Equal("mbid-1234"))
		})

		It("returns an error if Last.FM call returns an error 6 and mbid is empty", func() {
			httpClient.Res = http.Response{Body: ioutil.NopCloser(bytes.NewBufferString(lastfmError6)), StatusCode: 200}
			_, err := agent.GetBiography(ctx, "123", "U2", "")
			Expect(err).To(HaveOccurred())
			Expect(httpClient.RequestCount).To(Equal(1))
		})

		Context("MBID non existent in Last.FM", func() {
			It("calls again when the response is artist == [unknown]", func() {
				f, _ := os.Open("tests/fixtures/lastfm.artist.getinfo.unknown.json")
				httpClient.Res = http.Response{Body: f, StatusCode: 200}
				_, _ = agent.GetBiography(ctx, "123", "U2", "mbid-1234")
				Expect(httpClient.RequestCount).To(Equal(2))
				Expect(httpClient.SavedRequest.URL.Query().Get("mbid")).To(BeEmpty())
			})
			It("calls again when last.fm returns an error 6", func() {
				httpClient.Res = http.Response{Body: ioutil.NopCloser(bytes.NewBufferString(lastfmError6)), StatusCode: 200}
				_, _ = agent.GetBiography(ctx, "123", "U2", "mbid-1234")
				Expect(httpClient.RequestCount).To(Equal(2))
				Expect(httpClient.SavedRequest.URL.Query().Get("mbid")).To(BeEmpty())
			})
		})
	})

	Describe("GetSimilar", func() {
		var agent *lastfmAgent
		var httpClient *tests.FakeHttpClient
		BeforeEach(func() {
			httpClient = &tests.FakeHttpClient{}
			client := NewClient("API_KEY", "SECRET", "pt", httpClient)
			agent = lastFMConstructor(ds)
			agent.client = client
		})

		It("returns similar artists", func() {
			f, _ := os.Open("tests/fixtures/lastfm.artist.getsimilar.json")
			httpClient.Res = http.Response{Body: f, StatusCode: 200}
			Expect(agent.GetSimilar(ctx, "123", "U2", "mbid-1234", 2)).To(Equal([]agents.Artist{
				{Name: "Passengers", MBID: "e110c11f-1c94-4471-a350-c38f46b29389"},
				{Name: "INXS", MBID: "481bf5f9-2e7c-4c44-b08a-05b32bc7c00d"},
			}))
			Expect(httpClient.RequestCount).To(Equal(1))
			Expect(httpClient.SavedRequest.URL.Query().Get("mbid")).To(Equal("mbid-1234"))
		})

		It("returns an error if Last.FM call fails", func() {
			httpClient.Err = errors.New("error")
			_, err := agent.GetSimilar(ctx, "123", "U2", "mbid-1234", 2)
			Expect(err).To(HaveOccurred())
			Expect(httpClient.RequestCount).To(Equal(1))
			Expect(httpClient.SavedRequest.URL.Query().Get("mbid")).To(Equal("mbid-1234"))
		})

		It("returns an error if Last.FM call returns an error", func() {
			httpClient.Res = http.Response{Body: ioutil.NopCloser(bytes.NewBufferString(lastfmError3)), StatusCode: 200}
			_, err := agent.GetSimilar(ctx, "123", "U2", "mbid-1234", 2)
			Expect(err).To(HaveOccurred())
			Expect(httpClient.RequestCount).To(Equal(1))
			Expect(httpClient.SavedRequest.URL.Query().Get("mbid")).To(Equal("mbid-1234"))
		})

		It("returns an error if Last.FM call returns an error 6 and mbid is empty", func() {
			httpClient.Res = http.Response{Body: ioutil.NopCloser(bytes.NewBufferString(lastfmError6)), StatusCode: 200}
			_, err := agent.GetSimilar(ctx, "123", "U2", "", 2)
			Expect(err).To(HaveOccurred())
			Expect(httpClient.RequestCount).To(Equal(1))
		})

		Context("MBID non existent in Last.FM", func() {
			It("calls again when the response is artist == [unknown]", func() {
				f, _ := os.Open("tests/fixtures/lastfm.artist.getsimilar.unknown.json")
				httpClient.Res = http.Response{Body: f, StatusCode: 200}
				_, _ = agent.GetSimilar(ctx, "123", "U2", "mbid-1234", 2)
				Expect(httpClient.RequestCount).To(Equal(2))
				Expect(httpClient.SavedRequest.URL.Query().Get("mbid")).To(BeEmpty())
			})
			It("calls again when last.fm returns an error 6", func() {
				httpClient.Res = http.Response{Body: ioutil.NopCloser(bytes.NewBufferString(lastfmError6)), StatusCode: 200}
				_, _ = agent.GetSimilar(ctx, "123", "U2", "mbid-1234", 2)
				Expect(httpClient.RequestCount).To(Equal(2))
				Expect(httpClient.SavedRequest.URL.Query().Get("mbid")).To(BeEmpty())
			})
		})
	})

	Describe("GetTopSongs", func() {
		var agent *lastfmAgent
		var httpClient *tests.FakeHttpClient
		BeforeEach(func() {
			httpClient = &tests.FakeHttpClient{}
			client := NewClient("API_KEY", "SECRET", "pt", httpClient)
			agent = lastFMConstructor(ds)
			agent.client = client
		})

		It("returns top songs", func() {
			f, _ := os.Open("tests/fixtures/lastfm.artist.gettoptracks.json")
			httpClient.Res = http.Response{Body: f, StatusCode: 200}
			Expect(agent.GetTopSongs(ctx, "123", "U2", "mbid-1234", 2)).To(Equal([]agents.Song{
				{Name: "Beautiful Day", MBID: "f7f264d0-a89b-4682-9cd7-a4e7c37637af"},
				{Name: "With or Without You", MBID: "6b9a509f-6907-4a6e-9345-2f12da09ba4b"},
			}))
			Expect(httpClient.RequestCount).To(Equal(1))
			Expect(httpClient.SavedRequest.URL.Query().Get("mbid")).To(Equal("mbid-1234"))
		})

		It("returns an error if Last.FM call fails", func() {
			httpClient.Err = errors.New("error")
			_, err := agent.GetTopSongs(ctx, "123", "U2", "mbid-1234", 2)
			Expect(err).To(HaveOccurred())
			Expect(httpClient.RequestCount).To(Equal(1))
			Expect(httpClient.SavedRequest.URL.Query().Get("mbid")).To(Equal("mbid-1234"))
		})

		It("returns an error if Last.FM call returns an error", func() {
			httpClient.Res = http.Response{Body: ioutil.NopCloser(bytes.NewBufferString(lastfmError3)), StatusCode: 200}
			_, err := agent.GetTopSongs(ctx, "123", "U2", "mbid-1234", 2)
			Expect(err).To(HaveOccurred())
			Expect(httpClient.RequestCount).To(Equal(1))
			Expect(httpClient.SavedRequest.URL.Query().Get("mbid")).To(Equal("mbid-1234"))
		})

		It("returns an error if Last.FM call returns an error 6 and mbid is empty", func() {
			httpClient.Res = http.Response{Body: ioutil.NopCloser(bytes.NewBufferString(lastfmError6)), StatusCode: 200}
			_, err := agent.GetTopSongs(ctx, "123", "U2", "", 2)
			Expect(err).To(HaveOccurred())
			Expect(httpClient.RequestCount).To(Equal(1))
		})

		Context("MBID non existent in Last.FM", func() {
			It("calls again when the response is artist == [unknown]", func() {
				f, _ := os.Open("tests/fixtures/lastfm.artist.gettoptracks.unknown.json")
				httpClient.Res = http.Response{Body: f, StatusCode: 200}
				_, _ = agent.GetTopSongs(ctx, "123", "U2", "mbid-1234", 2)
				Expect(httpClient.RequestCount).To(Equal(2))
				Expect(httpClient.SavedRequest.URL.Query().Get("mbid")).To(BeEmpty())
			})
			It("calls again when last.fm returns an error 6", func() {
				httpClient.Res = http.Response{Body: ioutil.NopCloser(bytes.NewBufferString(lastfmError6)), StatusCode: 200}
				_, _ = agent.GetTopSongs(ctx, "123", "U2", "mbid-1234", 2)
				Expect(httpClient.RequestCount).To(Equal(2))
				Expect(httpClient.SavedRequest.URL.Query().Get("mbid")).To(BeEmpty())
			})
		})
	})

	Describe("Scrobbling", func() {
		var agent *lastfmAgent
		var httpClient *tests.FakeHttpClient
		var track *model.MediaFile
		BeforeEach(func() {
			ctx = request.WithUser(ctx, model.User{ID: "user-1"})
			_ = ds.UserProps(ctx).Put(sessionKeyProperty, "SK-1")
			httpClient = &tests.FakeHttpClient{}
			client := NewClient("API_KEY", "SECRET", "en", httpClient)
			agent = lastFMConstructor(ds)
			agent.client = client
			track = &model.MediaFile{
				ID:          "123",
				Title:       "Track Title",
				Album:       "Track Album",
				Artist:      "Track Artist",
				AlbumArtist: "Track AlbumArtist",
				TrackNumber: 1,
				Duration:    180,
				MbzTrackID:  "mbz-123",
			}
		})

		Describe("NowPlaying", func() {
			It("calls Last.fm with correct params", func() {
				httpClient.Res = http.Response{Body: ioutil.NopCloser(bytes.NewBufferString("{}")), StatusCode: 200}

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
				Expect(sentParams.Get("mbid")).To(Equal(track.MbzTrackID))
			})
		})

		Describe("Scrobble", func() {
			It("calls Last.fm with correct params", func() {
				ts := time.Now()
				scrobbles := []scrobbler.Scrobble{{MediaFile: *track, TimeStamp: ts}}
				httpClient.Res = http.Response{Body: ioutil.NopCloser(bytes.NewBufferString("{}")), StatusCode: 200}

				err := agent.Scrobble(ctx, "user-1", scrobbles)

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
				Expect(sentParams.Get("mbid")).To(Equal(track.MbzTrackID))
				Expect(sentParams.Get("timestamp")).To(Equal(strconv.FormatInt(ts.Unix(), 10)))
			})

			It("skips songs with less than 31 seconds", func() {
				track.Duration = 29
				scrobbles := []scrobbler.Scrobble{{MediaFile: *track, TimeStamp: time.Now()}}
				httpClient.Res = http.Response{Body: ioutil.NopCloser(bytes.NewBufferString("{}")), StatusCode: 200}

				err := agent.Scrobble(ctx, "user-1", scrobbles)

				Expect(err).ToNot(HaveOccurred())
				Expect(httpClient.SavedRequest).To(BeNil())
			})
		})
	})

})
