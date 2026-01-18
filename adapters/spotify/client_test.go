package spotify

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"os"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("client", func() {
	var httpClient *fakeHttpClient
	var client *client

	BeforeEach(func() {
		httpClient = &fakeHttpClient{}
		client = newClient("SPOTIFY_ID", "SPOTIFY_SECRET", httpClient)
	})

	Describe("ArtistImages", func() {
		It("returns artist images from a successful request", func() {
			f, _ := os.Open("tests/fixtures/spotify.search.artist.json")
			httpClient.mock("https://api.spotify.com/v1/search", http.Response{Body: f, StatusCode: 200})
			httpClient.mock("https://accounts.spotify.com/api/token", http.Response{
				StatusCode: 200,
				Body:       io.NopCloser(bytes.NewBufferString(`{"access_token": "NEW_ACCESS_TOKEN","token_type": "Bearer","expires_in": 3600}`)),
			})

			artists, err := client.searchArtists(context.TODO(), "U2", 10)
			Expect(err).To(BeNil())
			Expect(artists).To(HaveLen(20))
			Expect(artists[0].Popularity).To(Equal(82))

			images := artists[0].Images
			Expect(images).To(HaveLen(3))
			Expect(images[0].Width).To(Equal(640))
			Expect(images[1].Width).To(Equal(320))
			Expect(images[2].Width).To(Equal(160))
		})

		It("fails if artist was not found", func() {
			httpClient.mock("https://api.spotify.com/v1/search", http.Response{
				StatusCode: 200,
				Body: io.NopCloser(bytes.NewBufferString(`{
						  "artists" : {
							"href" : "https://api.spotify.com/v1/search?query=dasdasdas%2Cdna&type=artist&offset=0&limit=20",
							"items" : [ ], "limit" : 20, "next" : null, "offset" : 0, "previous" : null, "total" : 0
						  }}`)),
			})
			httpClient.mock("https://accounts.spotify.com/api/token", http.Response{
				StatusCode: 200,
				Body:       io.NopCloser(bytes.NewBufferString(`{"access_token": "NEW_ACCESS_TOKEN","token_type": "Bearer","expires_in": 3600}`)),
			})

			_, err := client.searchArtists(context.TODO(), "U2", 10)
			Expect(err).To(MatchError(ErrNotFound))
		})

		It("fails if not able to authorize", func() {
			f, _ := os.Open("tests/fixtures/spotify.search.artist.json")
			httpClient.mock("https://api.spotify.com/v1/search", http.Response{Body: f, StatusCode: 200})
			httpClient.mock("https://accounts.spotify.com/api/token", http.Response{
				StatusCode: 400,
				Body:       io.NopCloser(bytes.NewBufferString(`{"error":"invalid_client","error_description":"Invalid client"}`)),
			})

			_, err := client.searchArtists(context.TODO(), "U2", 10)
			Expect(err).To(MatchError("spotify error(invalid_client): Invalid client"))
		})
	})

	Describe("authorize", func() {
		It("returns an access_token on successful authorization", func() {
			httpClient.mock("https://accounts.spotify.com/api/token", http.Response{
				StatusCode: 200,
				Body:       io.NopCloser(bytes.NewBufferString(`{"access_token": "NEW_ACCESS_TOKEN","token_type": "Bearer","expires_in": 3600}`)),
			})

			token, err := client.authorize(context.TODO())
			Expect(err).To(BeNil())
			Expect(token).To(Equal("NEW_ACCESS_TOKEN"))
			auth := httpClient.lastRequest.Header.Get("Authorization")
			Expect(auth).To(Equal("Basic U1BPVElGWV9JRDpTUE9USUZZX1NFQ1JFVA=="))
		})

		It("fails on unsuccessful authorization", func() {
			httpClient.mock("https://accounts.spotify.com/api/token", http.Response{
				StatusCode: 400,
				Body:       io.NopCloser(bytes.NewBufferString(`{"error":"invalid_client","error_description":"Invalid client"}`)),
			})

			_, err := client.authorize(context.TODO())
			Expect(err).To(MatchError("spotify error(invalid_client): Invalid client"))
		})

		It("fails on invalid JSON response", func() {
			httpClient.mock("https://accounts.spotify.com/api/token", http.Response{
				StatusCode: 200,
				Body:       io.NopCloser(bytes.NewBufferString(`{NOT_VALID}`)),
			})

			_, err := client.authorize(context.TODO())
			Expect(err).To(MatchError("invalid character 'N' looking for beginning of object key string"))
		})
	})
})

type fakeHttpClient struct {
	responses   map[string]*http.Response
	lastRequest *http.Request
}

func (c *fakeHttpClient) mock(url string, response http.Response) {
	if c.responses == nil {
		c.responses = make(map[string]*http.Response)
	}
	c.responses[url] = &response
}

func (c *fakeHttpClient) Do(req *http.Request) (*http.Response, error) {
	c.lastRequest = req
	u := req.URL
	u.RawQuery = ""
	if resp, ok := c.responses[u.String()]; ok {
		return resp, nil
	}
	panic("URL not mocked: " + u.String())
}
