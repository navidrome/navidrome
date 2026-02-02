package tidal

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"os"

	"github.com/navidrome/navidrome/conf"
	"github.com/navidrome/navidrome/conf/configtest"
	"github.com/navidrome/navidrome/core/agents"
	"github.com/navidrome/navidrome/tests"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("tidalAgent", func() {
	var ctx context.Context

	BeforeEach(func() {
		ctx = context.Background()
		DeferCleanup(configtest.SetupConfig())
		conf.Server.Tidal.Enabled = true
		conf.Server.Tidal.ClientID = "test-client-id"
		conf.Server.Tidal.ClientSecret = "test-client-secret"
	})

	Describe("tidalConstructor", func() {
		It("returns nil when client ID is empty", func() {
			conf.Server.Tidal.ClientID = ""
			agent := tidalConstructor(&tests.MockDataStore{})
			Expect(agent).To(BeNil())
		})

		It("returns nil when client secret is empty", func() {
			conf.Server.Tidal.ClientSecret = ""
			agent := tidalConstructor(&tests.MockDataStore{})
			Expect(agent).To(BeNil())
		})

		It("returns agent when credentials are configured", func() {
			agent := tidalConstructor(&tests.MockDataStore{})
			Expect(agent).ToNot(BeNil())
			Expect(agent.AgentName()).To(Equal("tidal"))
		})
	})

	Describe("GetArtistImages", func() {
		var agent *tidalAgent
		var httpClient *mockHttpClient

		BeforeEach(func() {
			httpClient = newMockHttpClient()
			agent = &tidalAgent{
				ds:     &tests.MockDataStore{},
				client: newClient("test-id", "test-secret", httpClient),
			}
		})

		It("returns artist images from search result", func() {
			// Mock token response
			httpClient.tokenResponse = &http.Response{
				StatusCode: 200,
				Body:       io.NopCloser(bytes.NewBufferString(`{"access_token":"test-token","token_type":"Bearer","expires_in":86400}`)),
			}

			// Mock search response
			fSearch, _ := os.Open("tests/fixtures/tidal.search.artist.json")
			httpClient.searchResponse = &http.Response{Body: fSearch, StatusCode: 200}

			images, err := agent.GetArtistImages(ctx, "", "Daft Punk", "")

			Expect(err).ToNot(HaveOccurred())
			Expect(images).To(HaveLen(3))
			Expect(images[0].URL).To(ContainSubstring("resources.tidal.com"))
			Expect(images[0].Size).To(Equal(750))
		})

		It("returns ErrNotFound when artist is not found", func() {
			// Mock token response
			httpClient.tokenResponse = &http.Response{
				StatusCode: 200,
				Body:       io.NopCloser(bytes.NewBufferString(`{"access_token":"test-token","token_type":"Bearer","expires_in":86400}`)),
			}

			// Mock empty search response
			httpClient.searchResponse = &http.Response{
				StatusCode: 200,
				Body:       io.NopCloser(bytes.NewBufferString(`{"artists":[]}`)),
			}

			_, err := agent.GetArtistImages(ctx, "", "Nonexistent Artist", "")

			Expect(err).To(MatchError(agents.ErrNotFound))
		})

		It("returns ErrNotFound when artist name doesn't match", func() {
			// Mock token response
			httpClient.tokenResponse = &http.Response{
				StatusCode: 200,
				Body:       io.NopCloser(bytes.NewBufferString(`{"access_token":"test-token","token_type":"Bearer","expires_in":86400}`)),
			}

			// Mock search response with different artist
			fSearch, _ := os.Open("tests/fixtures/tidal.search.artist.json")
			httpClient.searchResponse = &http.Response{Body: fSearch, StatusCode: 200}

			_, err := agent.GetArtistImages(ctx, "", "Wrong Artist Name", "")

			Expect(err).To(MatchError(agents.ErrNotFound))
		})
	})

	Describe("GetSimilarArtists", func() {
		var agent *tidalAgent
		var httpClient *mockHttpClient

		BeforeEach(func() {
			httpClient = newMockHttpClient()
			agent = &tidalAgent{
				ds:     &tests.MockDataStore{},
				client: newClient("test-id", "test-secret", httpClient),
			}
		})

		It("returns similar artists", func() {
			// Mock token response
			httpClient.tokenResponse = &http.Response{
				StatusCode: 200,
				Body:       io.NopCloser(bytes.NewBufferString(`{"access_token":"test-token","token_type":"Bearer","expires_in":86400}`)),
			}

			// Mock search response
			fSearch, _ := os.Open("tests/fixtures/tidal.search.artist.json")
			httpClient.searchResponse = &http.Response{Body: fSearch, StatusCode: 200}

			// Mock similar artists response
			fSimilar, _ := os.Open("tests/fixtures/tidal.similar.artists.json")
			httpClient.similarResponse = &http.Response{Body: fSimilar, StatusCode: 200}

			similar, err := agent.GetSimilarArtists(ctx, "", "Daft Punk", "", 5)

			Expect(err).ToNot(HaveOccurred())
			Expect(similar).To(HaveLen(3))
			Expect(similar[0].Name).To(Equal("Justice"))
		})
	})

	Describe("GetArtistTopSongs", func() {
		var agent *tidalAgent
		var httpClient *mockHttpClient

		BeforeEach(func() {
			httpClient = newMockHttpClient()
			agent = &tidalAgent{
				ds:     &tests.MockDataStore{},
				client: newClient("test-id", "test-secret", httpClient),
			}
		})

		It("returns artist top songs", func() {
			// Mock token response
			httpClient.tokenResponse = &http.Response{
				StatusCode: 200,
				Body:       io.NopCloser(bytes.NewBufferString(`{"access_token":"test-token","token_type":"Bearer","expires_in":86400}`)),
			}

			// Mock search response
			fSearch, _ := os.Open("tests/fixtures/tidal.search.artist.json")
			httpClient.searchResponse = &http.Response{Body: fSearch, StatusCode: 200}

			// Mock top tracks response
			fTracks, _ := os.Open("tests/fixtures/tidal.artist.tracks.json")
			httpClient.tracksResponse = &http.Response{Body: fTracks, StatusCode: 200}

			songs, err := agent.GetArtistTopSongs(ctx, "", "Daft Punk", "", 5)

			Expect(err).ToNot(HaveOccurred())
			Expect(songs).To(HaveLen(3))
			Expect(songs[0].Name).To(Equal("Get Lucky"))
			Expect(songs[0].Duration).To(Equal(uint32(369000))) // 369 seconds * 1000
		})
	})

	Describe("GetArtistURL", func() {
		var agent *tidalAgent
		var httpClient *mockHttpClient

		BeforeEach(func() {
			httpClient = newMockHttpClient()
			agent = &tidalAgent{
				ds:     &tests.MockDataStore{},
				client: newClient("test-id", "test-secret", httpClient),
			}
		})

		It("returns artist URL", func() {
			// Mock token response
			httpClient.tokenResponse = &http.Response{
				StatusCode: 200,
				Body:       io.NopCloser(bytes.NewBufferString(`{"access_token":"test-token","token_type":"Bearer","expires_in":86400}`)),
			}

			// Mock search response
			fSearch, _ := os.Open("tests/fixtures/tidal.search.artist.json")
			httpClient.searchResponse = &http.Response{Body: fSearch, StatusCode: 200}

			url, err := agent.GetArtistURL(ctx, "", "Daft Punk", "")

			Expect(err).ToNot(HaveOccurred())
			Expect(url).To(Equal("https://tidal.com/browse/artist/4837227"))
		})
	})

	Describe("GetAlbumImages", func() {
		var agent *tidalAgent
		var httpClient *mockHttpClient

		BeforeEach(func() {
			httpClient = newMockHttpClient()
			agent = &tidalAgent{
				ds:     &tests.MockDataStore{},
				client: newClient("test-id", "test-secret", httpClient),
			}
		})

		It("returns album images", func() {
			// Mock token response
			httpClient.tokenResponse = &http.Response{
				StatusCode: 200,
				Body:       io.NopCloser(bytes.NewBufferString(`{"access_token":"test-token","token_type":"Bearer","expires_in":86400}`)),
			}

			// Mock album search response
			fAlbum, _ := os.Open("tests/fixtures/tidal.search.album.json")
			httpClient.albumSearchResponse = &http.Response{Body: fAlbum, StatusCode: 200}

			images, err := agent.GetAlbumImages(ctx, "Random Access Memories", "Daft Punk", "")

			Expect(err).ToNot(HaveOccurred())
			Expect(images).To(HaveLen(3))
			Expect(images[0].URL).To(ContainSubstring("resources.tidal.com"))
			Expect(images[0].Size).To(Equal(750))
		})

		It("returns ErrNotFound when album is not found", func() {
			// Mock token response
			httpClient.tokenResponse = &http.Response{
				StatusCode: 200,
				Body:       io.NopCloser(bytes.NewBufferString(`{"access_token":"test-token","token_type":"Bearer","expires_in":86400}`)),
			}

			// Mock empty album search response
			httpClient.albumSearchResponse = &http.Response{
				StatusCode: 200,
				Body:       io.NopCloser(bytes.NewBufferString(`{"albums":[]}`)),
			}

			_, err := agent.GetAlbumImages(ctx, "Nonexistent Album", "Unknown Artist", "")

			Expect(err).To(MatchError(agents.ErrNotFound))
		})
	})

	Describe("GetSimilarSongsByArtist", func() {
		var agent *tidalAgent
		var httpClient *mockHttpClient

		BeforeEach(func() {
			httpClient = newMockHttpClient()
			agent = &tidalAgent{
				ds:     &tests.MockDataStore{},
				client: newClient("test-id", "test-secret", httpClient),
			}
		})

		It("returns similar songs from similar artists", func() {
			// Mock token response
			httpClient.tokenResponse = &http.Response{
				StatusCode: 200,
				Body:       io.NopCloser(bytes.NewBufferString(`{"access_token":"test-token","token_type":"Bearer","expires_in":86400}`)),
			}

			// Mock search response
			fSearch, _ := os.Open("tests/fixtures/tidal.search.artist.json")
			httpClient.searchResponse = &http.Response{Body: fSearch, StatusCode: 200}

			// Mock similar artists response
			fSimilar, _ := os.Open("tests/fixtures/tidal.similar.artists.json")
			httpClient.similarResponse = &http.Response{Body: fSimilar, StatusCode: 200}

			// Mock top tracks response (will be called for each similar artist)
			fTracks, _ := os.Open("tests/fixtures/tidal.artist.tracks.json")
			httpClient.tracksResponse = &http.Response{Body: fTracks, StatusCode: 200}

			songs, err := agent.GetSimilarSongsByArtist(ctx, "", "Daft Punk", "", 5)

			Expect(err).ToNot(HaveOccurred())
			Expect(songs).To(HaveLen(5))
			Expect(songs[0].Name).To(Equal("Get Lucky"))
			Expect(songs[0].Artist).To(Equal("Justice"))
		})

		It("returns ErrNotFound when no similar artists found", func() {
			// Mock token response
			httpClient.tokenResponse = &http.Response{
				StatusCode: 200,
				Body:       io.NopCloser(bytes.NewBufferString(`{"access_token":"test-token","token_type":"Bearer","expires_in":86400}`)),
			}

			// Mock search response
			fSearch, _ := os.Open("tests/fixtures/tidal.search.artist.json")
			httpClient.searchResponse = &http.Response{Body: fSearch, StatusCode: 200}

			// Mock empty similar artists response
			httpClient.similarResponse = &http.Response{
				StatusCode: 200,
				Body:       io.NopCloser(bytes.NewBufferString(`{"data":[]}`)),
			}

			_, err := agent.GetSimilarSongsByArtist(ctx, "", "Daft Punk", "", 5)

			Expect(err).To(MatchError(agents.ErrNotFound))
		})
	})
})

// mockHttpClient is a mock HTTP client for testing
type mockHttpClient struct {
	tokenResponse       *http.Response
	searchResponse      *http.Response
	albumSearchResponse *http.Response
	artistResponse      *http.Response
	similarResponse     *http.Response
	tracksResponse      *http.Response
}

func newMockHttpClient() *mockHttpClient {
	return &mockHttpClient{}
}

func (c *mockHttpClient) Do(req *http.Request) (*http.Response, error) {
	// Handle token request
	if req.URL.Host == "auth.tidal.com" && req.URL.Path == "/v1/oauth2/token" {
		if c.tokenResponse != nil {
			return c.tokenResponse, nil
		}
		return &http.Response{
			StatusCode: 500,
			Body:       io.NopCloser(bytes.NewBufferString(`{"error":"no mock"}`)),
		}, nil
	}

	// Handle search request
	if req.URL.Host == "openapi.tidal.com" && req.URL.Path == "/search" {
		// Check if it's an album search (has type=ALBUMS parameter)
		if req.URL.Query().Get("type") == "ALBUMS" {
			if c.albumSearchResponse != nil {
				return c.albumSearchResponse, nil
			}
			return &http.Response{
				StatusCode: 200,
				Body:       io.NopCloser(bytes.NewBufferString(`{"albums":[]}`)),
			}, nil
		}
		// Otherwise, it's an artist search
		if c.searchResponse != nil {
			return c.searchResponse, nil
		}
		return &http.Response{
			StatusCode: 200,
			Body:       io.NopCloser(bytes.NewBufferString(`{"artists":[]}`)),
		}, nil
	}

	// Handle artist request
	if req.URL.Host == "openapi.tidal.com" && len(req.URL.Path) > 9 && req.URL.Path[:9] == "/artists/" {
		// Check if it's a similar artists or tracks request
		if len(req.URL.Path) > 17 && req.URL.Path[len(req.URL.Path)-8:] == "/similar" {
			if c.similarResponse != nil {
				return c.similarResponse, nil
			}
			return &http.Response{
				StatusCode: 200,
				Body:       io.NopCloser(bytes.NewBufferString(`{"data":[]}`)),
			}, nil
		}
		if len(req.URL.Path) > 16 && req.URL.Path[len(req.URL.Path)-7:] == "/tracks" {
			if c.tracksResponse != nil {
				// Need to return a new response each time since the body is consumed
				fTracks, _ := os.Open("tests/fixtures/tidal.artist.tracks.json")
				return &http.Response{Body: fTracks, StatusCode: 200}, nil
			}
			return &http.Response{
				StatusCode: 200,
				Body:       io.NopCloser(bytes.NewBufferString(`{"data":[]}`)),
			}, nil
		}
		if c.artistResponse != nil {
			return c.artistResponse, nil
		}
		return &http.Response{
			StatusCode: 404,
			Body:       io.NopCloser(bytes.NewBufferString(`{"errors":[{"status":404,"code":"NOT_FOUND"}]}`)),
		}, nil
	}

	panic("URL not mocked: " + req.URL.String())
}
