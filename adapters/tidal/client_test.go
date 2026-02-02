package tidal

import (
	"bytes"
	"io"
	"net/http"
	"os"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("client", func() {
	var httpClient *fakeHttpClient
	var c *client

	BeforeEach(func() {
		httpClient = &fakeHttpClient{}
		c = newClient("test-client-id", "test-client-secret", httpClient)
	})

	Describe("searchArtists", func() {
		BeforeEach(func() {
			// Mock token response
			httpClient.mock("https://auth.tidal.com/v1/oauth2/token", http.Response{
				StatusCode: 200,
				Body:       io.NopCloser(bytes.NewBufferString(`{"access_token":"test-token","token_type":"Bearer","expires_in":86400}`)),
			})
		})

		It("returns artists from a successful request", func() {
			f, err := os.Open("tests/fixtures/tidal.search.artist.json")
			Expect(err).To(BeNil())
			httpClient.mock("https://openapi.tidal.com/search", http.Response{Body: f, StatusCode: 200})

			artists, err := c.searchArtists(GinkgoT().Context(), "Daft Punk", 20)
			Expect(err).To(BeNil())
			Expect(artists).To(HaveLen(2))
			Expect(artists[0].Attributes.Name).To(Equal("Daft Punk"))
		})

		It("returns ErrNotFound when no artists found", func() {
			httpClient.mock("https://openapi.tidal.com/search", http.Response{
				StatusCode: 200,
				Body:       io.NopCloser(bytes.NewBufferString(`{"artists":[]}`)),
			})

			_, err := c.searchArtists(GinkgoT().Context(), "Nonexistent Artist", 20)
			Expect(err).To(MatchError(ErrNotFound))
		})
	})

	Describe("getArtistTopTracks", func() {
		BeforeEach(func() {
			// Mock token response
			httpClient.mock("https://auth.tidal.com/v1/oauth2/token", http.Response{
				StatusCode: 200,
				Body:       io.NopCloser(bytes.NewBufferString(`{"access_token":"test-token","token_type":"Bearer","expires_in":86400}`)),
			})
		})

		It("returns tracks from a successful request", func() {
			f, err := os.Open("tests/fixtures/tidal.artist.tracks.json")
			Expect(err).To(BeNil())
			httpClient.mock("https://openapi.tidal.com/artists/4837227/tracks", http.Response{Body: f, StatusCode: 200})

			tracks, err := c.getArtistTopTracks(GinkgoT().Context(), "4837227", 10)
			Expect(err).To(BeNil())
			Expect(tracks).To(HaveLen(3))
			Expect(tracks[0].Attributes.Title).To(Equal("Get Lucky"))
			Expect(tracks[0].Attributes.Duration).To(Equal(369))
		})
	})

	Describe("getSimilarArtists", func() {
		BeforeEach(func() {
			// Mock token response
			httpClient.mock("https://auth.tidal.com/v1/oauth2/token", http.Response{
				StatusCode: 200,
				Body:       io.NopCloser(bytes.NewBufferString(`{"access_token":"test-token","token_type":"Bearer","expires_in":86400}`)),
			})
		})

		It("returns similar artists from a successful request", func() {
			f, err := os.Open("tests/fixtures/tidal.similar.artists.json")
			Expect(err).To(BeNil())
			httpClient.mock("https://openapi.tidal.com/artists/4837227/similar", http.Response{Body: f, StatusCode: 200})

			artists, err := c.getSimilarArtists(GinkgoT().Context(), "4837227", 10)
			Expect(err).To(BeNil())
			Expect(artists).To(HaveLen(3))
			Expect(artists[0].Attributes.Name).To(Equal("Justice"))
		})
	})

	Describe("getToken", func() {
		It("returns token from a successful request", func() {
			httpClient.mock("https://auth.tidal.com/v1/oauth2/token", http.Response{
				StatusCode: 200,
				Body:       io.NopCloser(bytes.NewBufferString(`{"access_token":"test-token-123","token_type":"Bearer","expires_in":86400}`)),
			})

			token, err := c.getToken(GinkgoT().Context())
			Expect(err).To(BeNil())
			Expect(token).To(Equal("test-token-123"))
		})

		It("caches token for subsequent requests", func() {
			httpClient.mock("https://auth.tidal.com/v1/oauth2/token", http.Response{
				StatusCode: 200,
				Body:       io.NopCloser(bytes.NewBufferString(`{"access_token":"test-token-123","token_type":"Bearer","expires_in":86400}`)),
			})

			token1, err := c.getToken(GinkgoT().Context())
			Expect(err).To(BeNil())

			// Second call should use cached token
			token2, err := c.getToken(GinkgoT().Context())
			Expect(err).To(BeNil())
			Expect(token2).To(Equal(token1))
		})

		It("returns error on failed token request", func() {
			httpClient.mock("https://auth.tidal.com/v1/oauth2/token", http.Response{
				StatusCode: 401,
				Body:       io.NopCloser(bytes.NewBufferString(`{"error":"invalid_client"}`)),
			})

			_, err := c.getToken(GinkgoT().Context())
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("failed to get token"))
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
