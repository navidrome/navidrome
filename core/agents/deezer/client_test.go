package deezer

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("client", func() {
	var httpClient *fakeHttpClient
	var client *client

	BeforeEach(func() {
		httpClient = &fakeHttpClient{}
		client = newClient(httpClient, "en")
	})

	Describe("ArtistImages", func() {
		It("returns artist images from a successful request", func() {
			f, err := os.Open("tests/fixtures/deezer.search.artist.json")
			Expect(err).To(BeNil())
			httpClient.mock("https://api.deezer.com/search/artist", http.Response{Body: f, StatusCode: 200})

			artists, err := client.searchArtists(GinkgoT().Context(), "Michael Jackson", 20)
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

			_, err := client.searchArtists(GinkgoT().Context(), "Michael Jackson", 20)
			Expect(err).To(MatchError(ErrNotFound))
		})
	})

	Describe("ArtistBio", func() {
		BeforeEach(func() {
			// Mock the JWT token endpoint with a valid JWT that expires in 5 minutes
			testJWT := createTestJWT(5 * time.Minute)
			httpClient.mock("https://auth.deezer.com/login/anonymous", http.Response{
				StatusCode: 200,
				Body:       io.NopCloser(bytes.NewBufferString(fmt.Sprintf(`{"jwt":"%s","refresh_token":""}`, testJWT))),
			})
		})

		It("returns artist bio from a successful request", func() {
			f, err := os.Open("tests/fixtures/deezer.artist.bio.json")
			Expect(err).To(BeNil())
			httpClient.mock("https://pipe.deezer.com/api", http.Response{Body: f, StatusCode: 200})

			bio, err := client.getArtistBio(GinkgoT().Context(), 27)
			Expect(err).To(BeNil())
			Expect(bio).To(ContainSubstring("Schoolmates Thomas and Guy-Manuel"))
			Expect(bio).ToNot(ContainSubstring("<p>"))
			Expect(bio).ToNot(ContainSubstring("</p>"))
		})

		It("uses the configured language", func() {
			client = newClient(httpClient, "fr")
			// Mock JWT token for the new client instance with a valid JWT
			testJWT := createTestJWT(5 * time.Minute)
			httpClient.mock("https://auth.deezer.com/login/anonymous", http.Response{
				StatusCode: 200,
				Body:       io.NopCloser(bytes.NewBufferString(fmt.Sprintf(`{"jwt":"%s","refresh_token":""}`, testJWT))),
			})
			f, err := os.Open("tests/fixtures/deezer.artist.bio.json")
			Expect(err).To(BeNil())
			httpClient.mock("https://pipe.deezer.com/api", http.Response{Body: f, StatusCode: 200})

			_, err = client.getArtistBio(GinkgoT().Context(), 27)
			Expect(err).To(BeNil())
			Expect(httpClient.lastRequest.Header.Get("Accept-Language")).To(Equal("fr"))
		})

		It("includes the JWT token in the request", func() {
			f, err := os.Open("tests/fixtures/deezer.artist.bio.json")
			Expect(err).To(BeNil())
			httpClient.mock("https://pipe.deezer.com/api", http.Response{Body: f, StatusCode: 200})

			_, err = client.getArtistBio(GinkgoT().Context(), 27)
			Expect(err).To(BeNil())
			// Verify that the Authorization header has the Bearer token format
			authHeader := httpClient.lastRequest.Header.Get("Authorization")
			Expect(authHeader).To(HavePrefix("Bearer "))
			Expect(len(authHeader)).To(BeNumerically(">", 20)) // JWT tokens are longer than 20 chars
		})

		It("handles GraphQL errors", func() {
			errorResponse := `{
				"data": {
					"artist": {
						"bio": {
							"full": ""
						}
					}
				},
				"errors": [
					{
						"message": "Artist not found"
					},
					{
						"message": "Invalid artist ID"
					}
				]
			}`
			httpClient.mock("https://pipe.deezer.com/api", http.Response{
				StatusCode: 200,
				Body:       io.NopCloser(bytes.NewBufferString(errorResponse)),
			})

			_, err := client.getArtistBio(GinkgoT().Context(), 999)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("GraphQL error"))
			Expect(err.Error()).To(ContainSubstring("Artist not found"))
			Expect(err.Error()).To(ContainSubstring("Invalid artist ID"))
		})

		It("handles empty biography", func() {
			emptyBioResponse := `{
				"data": {
					"artist": {
						"bio": {
							"full": ""
						}
					}
				}
			}`
			httpClient.mock("https://pipe.deezer.com/api", http.Response{
				StatusCode: 200,
				Body:       io.NopCloser(bytes.NewBufferString(emptyBioResponse)),
			})

			_, err := client.getArtistBio(GinkgoT().Context(), 27)
			Expect(err).To(MatchError("deezer: biography not found"))
		})

		It("handles JWT token fetch failure", func() {
			httpClient.mock("https://auth.deezer.com/login/anonymous", http.Response{
				StatusCode: 500,
				Body:       io.NopCloser(bytes.NewBufferString(`{"error":"Internal server error"}`)),
			})

			_, err := client.getArtistBio(GinkgoT().Context(), 27)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("failed to get JWT"))
		})

		It("handles JWT token that expires too soon", func() {
			// Create a JWT that expires in 30 seconds (less than the 1-minute buffer)
			expiredJWT := createTestJWT(30 * time.Second)
			httpClient.mock("https://auth.deezer.com/login/anonymous", http.Response{
				StatusCode: 200,
				Body:       io.NopCloser(bytes.NewBufferString(fmt.Sprintf(`{"jwt":"%s","refresh_token":""}`, expiredJWT))),
			})

			_, err := client.getArtistBio(GinkgoT().Context(), 27)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("JWT token already expired or expires too soon"))
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
