package lastfm

import (
	"bytes"
	"context"
	"crypto/md5"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"

	"github.com/navidrome/navidrome/tests"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Client", func() {
	var httpClient *tests.FakeHttpClient
	var client *Client

	BeforeEach(func() {
		httpClient = &tests.FakeHttpClient{}
		client = NewClient("API_KEY", "SECRET", "pt", httpClient)
	})

	Describe("ArtistGetInfo", func() {
		It("returns an artist for a successful response", func() {
			f, _ := os.Open("tests/fixtures/lastfm.artist.getinfo.json")
			httpClient.Res = http.Response{Body: f, StatusCode: 200}

			artist, err := client.ArtistGetInfo(context.Background(), "U2", "123")
			Expect(err).To(BeNil())
			Expect(artist.Name).To(Equal("U2"))
			Expect(httpClient.SavedRequest.URL.String()).To(Equal(apiBaseUrl + "?api_key=API_KEY&artist=U2&format=json&lang=pt&mbid=123&method=artist.getInfo"))
		})

		It("fails if Last.FM returns an http status != 200", func() {
			httpClient.Res = http.Response{
				Body:       ioutil.NopCloser(bytes.NewBufferString(`Internal Server Error`)),
				StatusCode: 500,
			}

			_, err := client.ArtistGetInfo(context.Background(), "U2", "123")
			Expect(err).To(MatchError("last.fm http status: (500)"))
		})

		It("fails if Last.FM returns an http status != 200", func() {
			httpClient.Res = http.Response{
				Body:       ioutil.NopCloser(bytes.NewBufferString(`{"error":3,"message":"Invalid Method - No method with that name in this package"}`)),
				StatusCode: 400,
			}

			_, err := client.ArtistGetInfo(context.Background(), "U2", "123")
			Expect(err).To(MatchError(&lastFMError{Code: 3, Message: "Invalid Method - No method with that name in this package"}))
		})

		It("fails if Last.FM returns an error", func() {
			httpClient.Res = http.Response{
				Body:       ioutil.NopCloser(bytes.NewBufferString(`{"error":6,"message":"The artist you supplied could not be found"}`)),
				StatusCode: 200,
			}

			_, err := client.ArtistGetInfo(context.Background(), "U2", "123")
			Expect(err).To(MatchError(&lastFMError{Code: 6, Message: "The artist you supplied could not be found"}))
		})

		It("fails if HttpClient.Do() returns error", func() {
			httpClient.Err = errors.New("generic error")

			_, err := client.ArtistGetInfo(context.Background(), "U2", "123")
			Expect(err).To(MatchError("generic error"))
		})

		It("fails if returned body is not a valid JSON", func() {
			httpClient.Res = http.Response{
				Body:       ioutil.NopCloser(bytes.NewBufferString(`<xml>NOT_VALID_JSON</xml>`)),
				StatusCode: 200,
			}

			_, err := client.ArtistGetInfo(context.Background(), "U2", "123")
			Expect(err).To(MatchError("invalid character '<' looking for beginning of value"))
		})

	})

	Describe("ArtistGetSimilar", func() {
		It("returns an artist for a successful response", func() {
			f, _ := os.Open("tests/fixtures/lastfm.artist.getsimilar.json")
			httpClient.Res = http.Response{Body: f, StatusCode: 200}

			similar, err := client.ArtistGetSimilar(context.Background(), "U2", "123", 2)
			Expect(err).To(BeNil())
			Expect(len(similar.Artists)).To(Equal(2))
			Expect(httpClient.SavedRequest.URL.String()).To(Equal(apiBaseUrl + "?api_key=API_KEY&artist=U2&format=json&limit=2&mbid=123&method=artist.getSimilar"))
		})
	})

	Describe("ArtistGetTopTracks", func() {
		It("returns top tracks for a successful response", func() {
			f, _ := os.Open("tests/fixtures/lastfm.artist.gettoptracks.json")
			httpClient.Res = http.Response{Body: f, StatusCode: 200}

			top, err := client.ArtistGetTopTracks(context.Background(), "U2", "123", 2)
			Expect(err).To(BeNil())
			Expect(len(top.Track)).To(Equal(2))
			Expect(httpClient.SavedRequest.URL.String()).To(Equal(apiBaseUrl + "?api_key=API_KEY&artist=U2&format=json&limit=2&mbid=123&method=artist.getTopTracks"))
		})
	})

	Describe("GetToken", func() {
		It("returns a token when the request is successful", func() {
			httpClient.Res = http.Response{
				Body:       ioutil.NopCloser(bytes.NewBufferString(`{"token":"TOKEN"}`)),
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

	Describe("GetSession", func() {
		It("returns a session key when the request is successful", func() {
			httpClient.Res = http.Response{
				Body:       ioutil.NopCloser(bytes.NewBufferString(`{"session":{"name":"Navidrome","key":"SESSION_KEY","subscriber":0}}`)),
				StatusCode: 200,
			}

			Expect(client.GetSession(context.Background(), "TOKEN")).To(Equal("SESSION_KEY"))
			queryParams := httpClient.SavedRequest.URL.Query()
			Expect(queryParams.Get("method")).To(Equal("auth.getSession"))
			Expect(queryParams.Get("format")).To(Equal("json"))
			Expect(queryParams.Get("token")).To(Equal("TOKEN"))
			Expect(queryParams.Get("api_key")).To(Equal("API_KEY"))
			Expect(queryParams.Get("api_sig")).ToNot(BeEmpty())
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
