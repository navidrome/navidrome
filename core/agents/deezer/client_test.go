package deezer

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
		client = newClient(httpClient)
	})

	Describe("ArtistImages", func() {
		It("returns artist images from a successful request", func() {
			f, err := os.Open("tests/fixtures/deezer.search.artist.json")
			Expect(err).To(BeNil())
			httpClient.mock("https://api.deezer.com/search/artist", http.Response{Body: f, StatusCode: 200})

			artists, err := client.searchArtists(context.TODO(), "Michael Jackson", 20)
			Expect(err).To(BeNil())
			Expect(artists).To(HaveLen(17))
			Expect(artists[0].Name).To(Equal("Michael Jackson"))
			Expect(artists[0].PictureXl).To(Equal("https://cdn-images.dzcdn.net/images/artist/97fae13b2b30e4aec2e8c9e0c7839d92/1000x1000-000000-80-0-0.jpg"))
		})

		It("fails if artist was not found", func() {
			httpClient.mock("https://api.deezer.com/search/artist", http.Response{
				StatusCode: 200,
				Body:       io.NopCloser(bytes.NewBufferString(`{"data":[],"total":0}`)),
			})

			_, err := client.searchArtists(context.TODO(), "Michael Jackson", 20)
			Expect(err).To(MatchError(ErrNotFound))
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
