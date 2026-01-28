package lastfm

import (
	"bytes"
	"context"
	"crypto/md5"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"

	"github.com/navidrome/navidrome/tests"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("client", func() {
	var httpClient *tests.FakeHttpClient
	var client *client

	BeforeEach(func() {
		httpClient = &tests.FakeHttpClient{}
		client = newClient("API_KEY", "SECRET", "pt", httpClient)
	})

	Describe("albumGetInfo", func() {
		It("returns an album on successful response", func() {
			f, _ := os.Open("tests/fixtures/lastfm.album.getinfo.json")
			httpClient.Res = http.Response{Body: f, StatusCode: 200}

			album, err := client.albumGetInfo(context.Background(), "Believe", "U2", "mbid-1234")
			Expect(err).To(BeNil())
			Expect(album.Name).To(Equal("Believe"))
			Expect(httpClient.SavedRequest.URL.String()).To(Equal(apiBaseUrl + "?album=Believe&api_key=API_KEY&artist=U2&format=json&lang=pt&mbid=mbid-1234&method=album.getInfo"))
		})
	})

	Describe("artistGetInfo", func() {
		It("returns an artist for a successful response", func() {
			f, _ := os.Open("tests/fixtures/lastfm.artist.getinfo.json")
			httpClient.Res = http.Response{Body: f, StatusCode: 200}

			artist, err := client.artistGetInfo(context.Background(), "U2")
			Expect(err).To(BeNil())
			Expect(artist.Name).To(Equal("U2"))
			Expect(httpClient.SavedRequest.URL.String()).To(Equal(apiBaseUrl + "?api_key=API_KEY&artist=U2&format=json&lang=pt&method=artist.getInfo"))
		})

		It("fails if Last.fm returns an http status != 200", func() {
			httpClient.Res = http.Response{
				Body:       io.NopCloser(bytes.NewBufferString(`Internal Server Error`)),
				StatusCode: 500,
			}

			_, err := client.artistGetInfo(context.Background(), "U2")
			Expect(err).To(MatchError("last.fm http status: (500)"))
		})

		It("fails if Last.fm returns an http status != 200", func() {
			httpClient.Res = http.Response{
				Body:       io.NopCloser(bytes.NewBufferString(`{"error":3,"message":"Invalid Method - No method with that name in this package"}`)),
				StatusCode: 400,
			}

			_, err := client.artistGetInfo(context.Background(), "U2")
			Expect(err).To(MatchError(&lastFMError{Code: 3, Message: "Invalid Method - No method with that name in this package"}))
		})

		It("fails if Last.fm returns an error", func() {
			httpClient.Res = http.Response{
				Body:       io.NopCloser(bytes.NewBufferString(`{"error":6,"message":"The artist you supplied could not be found"}`)),
				StatusCode: 200,
			}

			_, err := client.artistGetInfo(context.Background(), "U2")
			Expect(err).To(MatchError(&lastFMError{Code: 6, Message: "The artist you supplied could not be found"}))
		})

		It("fails if HttpClient.Do() returns error", func() {
			httpClient.Err = errors.New("generic error")

			_, err := client.artistGetInfo(context.Background(), "U2")
			Expect(err).To(MatchError("generic error"))
		})

		It("fails if returned body is not a valid JSON", func() {
			httpClient.Res = http.Response{
				Body:       io.NopCloser(bytes.NewBufferString(`<xml>NOT_VALID_JSON</xml>`)),
				StatusCode: 200,
			}

			_, err := client.artistGetInfo(context.Background(), "U2")
			Expect(err).To(MatchError("invalid character '<' looking for beginning of value"))
		})

	})

	Describe("artistGetSimilar", func() {
		It("returns an artist for a successful response", func() {
			f, _ := os.Open("tests/fixtures/lastfm.artist.getsimilar.json")
			httpClient.Res = http.Response{Body: f, StatusCode: 200}

			similar, err := client.artistGetSimilar(context.Background(), "U2", 2)
			Expect(err).To(BeNil())
			Expect(len(similar.Artists)).To(Equal(2))
			Expect(httpClient.SavedRequest.URL.String()).To(Equal(apiBaseUrl + "?api_key=API_KEY&artist=U2&format=json&limit=2&method=artist.getSimilar"))
		})
	})

	Describe("artistGetTopTracks", func() {
		It("returns top tracks for a successful response", func() {
			f, _ := os.Open("tests/fixtures/lastfm.artist.gettoptracks.json")
			httpClient.Res = http.Response{Body: f, StatusCode: 200}

			top, err := client.artistGetTopTracks(context.Background(), "U2", 2)
			Expect(err).To(BeNil())
			Expect(len(top.Track)).To(Equal(2))
			Expect(httpClient.SavedRequest.URL.String()).To(Equal(apiBaseUrl + "?api_key=API_KEY&artist=U2&format=json&limit=2&method=artist.getTopTracks"))
		})
	})

	Describe("trackGetSimilar", func() {
		It("returns similar tracks for a successful response", func() {
			f, _ := os.Open("tests/fixtures/lastfm.track.getsimilar.json")
			httpClient.Res = http.Response{Body: f, StatusCode: 200}

			similar, err := client.trackGetSimilar(context.Background(), "Just Can't Get Enough", "Depeche Mode", 5)
			Expect(err).To(BeNil())
			Expect(len(similar.Track)).To(Equal(5))
			Expect(similar.Track[0].Name).To(Equal("Dreaming of Me"))
			Expect(similar.Track[0].Artist.Name).To(Equal("Depeche Mode"))
			Expect(similar.Track[0].Match).To(Equal(1.0))
			Expect(httpClient.SavedRequest.URL.String()).To(Equal(apiBaseUrl + "?api_key=API_KEY&artist=Depeche+Mode&format=json&limit=5&method=track.getSimilar&track=Just+Can%27t+Get+Enough"))
		})

		It("returns empty list when no similar tracks found", func() {
			f, _ := os.Open("tests/fixtures/lastfm.track.getsimilar.unknown.json")
			httpClient.Res = http.Response{Body: f, StatusCode: 200}

			similar, err := client.trackGetSimilar(context.Background(), "UnknownTrack", "UnknownArtist", 3)
			Expect(err).To(BeNil())
			Expect(similar.Track).To(BeEmpty())
		})
	})

	Describe("GetToken", func() {
		It("returns a token when the request is successful", func() {
			httpClient.Res = http.Response{
				Body:       io.NopCloser(bytes.NewBufferString(`{"token":"TOKEN"}`)),
				StatusCode: 200,
			}

			Expect(client.GetToken(context.Background())).To(Equal("TOKEN"))
			queryParams := httpClient.SavedRequest.URL.Query()
			Expect(queryParams.Get("method")).To(Equal("auth.getToken"))
			Expect(queryParams.Get("format")).To(Equal("json"))
			Expect(queryParams.Get("api_key")).To(Equal("API_KEY"))
			Expect(queryParams.Get("api_sig")).ToNot(BeEmpty())
		})
	})

	Describe("getSession", func() {
		It("returns a session key when the request is successful", func() {
			httpClient.Res = http.Response{
				Body:       io.NopCloser(bytes.NewBufferString(`{"session":{"name":"Navidrome","key":"SESSION_KEY","subscriber":0}}`)),
				StatusCode: 200,
			}

			Expect(client.getSession(context.Background(), "TOKEN")).To(Equal("SESSION_KEY"))
			queryParams := httpClient.SavedRequest.URL.Query()
			Expect(queryParams.Get("method")).To(Equal("auth.getSession"))
			Expect(queryParams.Get("format")).To(Equal("json"))
			Expect(queryParams.Get("token")).To(Equal("TOKEN"))
			Expect(queryParams.Get("api_key")).To(Equal("API_KEY"))
			Expect(queryParams.Get("api_sig")).ToNot(BeEmpty())
		})
	})

	Describe("scrobble", func() {
		It("sends parameters in request body for POST", func() {
			httpClient.Res = http.Response{
				Body:       io.NopCloser(bytes.NewBufferString(`{"scrobbles":{"scrobble":{"ignoredMessage":{"code":"0"}},"@attr":{"accepted":1}}}`)),
				StatusCode: 200,
			}

			info := ScrobbleInfo{
				artist:      "U2",
				track:       "One",
				album:       "Achtung Baby",
				trackNumber: 1,
				duration:    276,
				albumArtist: "U2",
			}
			err := client.scrobble(context.Background(), "SESSION_KEY", info)
			Expect(err).To(BeNil())

			req := httpClient.SavedRequest
			Expect(req.Method).To(Equal(http.MethodPost))
			Expect(req.Header.Get("Content-Type")).To(Equal("application/x-www-form-urlencoded"))
			Expect(req.URL.RawQuery).To(BeEmpty())

			body, _ := io.ReadAll(req.Body)
			bodyParams, _ := url.ParseQuery(string(body))
			Expect(bodyParams.Get("method")).To(Equal("track.scrobble"))
			Expect(bodyParams.Get("artist")).To(Equal("U2"))
			Expect(bodyParams.Get("track")).To(Equal("One"))
			Expect(bodyParams.Get("sk")).To(Equal("SESSION_KEY"))
			Expect(bodyParams.Get("api_key")).To(Equal("API_KEY"))
			Expect(bodyParams.Get("api_sig")).ToNot(BeEmpty())
		})
	})

	Describe("updateNowPlaying", func() {
		It("sends parameters in request body for POST", func() {
			httpClient.Res = http.Response{
				Body:       io.NopCloser(bytes.NewBufferString(`{"nowplaying":{"ignoredMessage":{"code":"0"}}}`)),
				StatusCode: 200,
			}

			info := ScrobbleInfo{
				artist:      "U2",
				track:       "One",
				album:       "Achtung Baby",
				trackNumber: 1,
				duration:    276,
				albumArtist: "U2",
			}
			err := client.updateNowPlaying(context.Background(), "SESSION_KEY", info)
			Expect(err).To(BeNil())

			req := httpClient.SavedRequest
			Expect(req.Method).To(Equal(http.MethodPost))
			Expect(req.Header.Get("Content-Type")).To(Equal("application/x-www-form-urlencoded"))
			Expect(req.URL.RawQuery).To(BeEmpty())

			body, _ := io.ReadAll(req.Body)
			bodyParams, _ := url.ParseQuery(string(body))
			Expect(bodyParams.Get("method")).To(Equal("track.updateNowPlaying"))
			Expect(bodyParams.Get("artist")).To(Equal("U2"))
			Expect(bodyParams.Get("track")).To(Equal("One"))
			Expect(bodyParams.Get("sk")).To(Equal("SESSION_KEY"))
			Expect(bodyParams.Get("api_key")).To(Equal("API_KEY"))
			Expect(bodyParams.Get("api_sig")).ToNot(BeEmpty())
		})
	})

	Describe("sign", func() {
		It("adds an api_sig param with the signature", func() {
			params := url.Values{}
			params.Add("d", "444")
			params.Add("callback", "https://myserver.com")
			params.Add("a", "111")
			params.Add("format", "json")
			params.Add("c", "333")
			params.Add("b", "222")
			client.sign(params)
			Expect(params).To(HaveKey("api_sig"))
			sig := params.Get("api_sig")
			expected := fmt.Sprintf("%x", md5.Sum([]byte("a111b222c333d444SECRET")))
			Expect(sig).To(Equal(expected))
		})
	})
})
