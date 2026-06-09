package deezer

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"

	"github.com/navidrome/navidrome/conf"
	"github.com/navidrome/navidrome/conf/configtest"
	"github.com/navidrome/navidrome/core/agents"
	"github.com/navidrome/navidrome/tests"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("deezerAgent", func() {
	var ctx context.Context

	BeforeEach(func() {
		ctx = context.Background()
		DeferCleanup(configtest.SetupConfig())
		conf.Server.Deezer.Enabled = true
	})

	Describe("deezerConstructor", func() {
		It("uses configured languages", func() {
			conf.Server.Deezer.Languages = []string{"pt", "en"}
			agent := deezerConstructor(&tests.MockDataStore{}).(*deezerAgent)
			Expect(agent.languages).To(Equal([]string{"pt", "en"}))
		})
	})

	Describe("GetArtistBiography - Language Fallback", func() {
		var agent *deezerAgent
		var httpClient *langAwareHttpClient

		BeforeEach(func() {
			httpClient = newLangAwareHttpClient()

			// Mock search artist (returns Michael Jackson)
			fSearch, _ := os.Open("tests/fixtures/deezer.search.artist.json")
			httpClient.searchResponse = &http.Response{Body: fSearch, StatusCode: 200}

			// Mock JWT token
			testJWT := createTestJWT(5 * time.Minute)
			httpClient.jwtResponse = &http.Response{
				StatusCode: 200,
				Body:       io.NopCloser(bytes.NewBufferString(fmt.Sprintf(`{"jwt":"%s","refresh_token":""}`, testJWT))),
			}
		})

		setupAgent := func(languages []string) {
			conf.Server.Deezer.Languages = languages
			agent = &deezerAgent{
				dataStore: &tests.MockDataStore{},
				client:    newClient(httpClient),
				languages: languages,
			}
		}

		It("returns content in first language when available (1 bio API call)", func() {
			setupAgent([]string{"fr", "en"})

			// French biography available
			fFr, _ := os.Open("tests/fixtures/deezer.artist.bio.fr.json")
			httpClient.bioResponses["fr"] = &http.Response{Body: fFr, StatusCode: 200}

			bio, err := agent.GetArtistBiography(ctx, "", "Michael Jackson", "")

			Expect(err).ToNot(HaveOccurred())
			Expect(bio).To(ContainSubstring("Guy-Manuel de Homem Christo et Thomas Bangalter"))
			Expect(httpClient.bioRequestCount).To(Equal(1))
			Expect(httpClient.bioRequests[0].Header.Get("Accept-Language")).To(Equal("fr"))
		})

		It("falls back to second language when first returns empty (2 bio API calls)", func() {
			setupAgent([]string{"ja", "en"})

			// Japanese returns empty biography
			fJa, _ := os.Open("tests/fixtures/deezer.artist.bio.empty.json")
			httpClient.bioResponses["ja"] = &http.Response{Body: fJa, StatusCode: 200}
			// English returns full biography
			fEn, _ := os.Open("tests/fixtures/deezer.artist.bio.en.json")
			httpClient.bioResponses["en"] = &http.Response{Body: fEn, StatusCode: 200}

			bio, err := agent.GetArtistBiography(ctx, "", "Michael Jackson", "")

			Expect(err).ToNot(HaveOccurred())
			Expect(bio).To(ContainSubstring("Schoolmates Thomas and Guy-Manuel"))
			Expect(httpClient.bioRequestCount).To(Equal(2))
			Expect(httpClient.bioRequests[0].Header.Get("Accept-Language")).To(Equal("ja"))
			Expect(httpClient.bioRequests[1].Header.Get("Accept-Language")).To(Equal("en"))
		})

		It("returns ErrNotFound when all languages return empty", func() {
			setupAgent([]string{"ja", "xx"})

			// Both languages return empty biography
			fJa, _ := os.Open("tests/fixtures/deezer.artist.bio.empty.json")
			httpClient.bioResponses["ja"] = &http.Response{Body: fJa, StatusCode: 200}
			fXx, _ := os.Open("tests/fixtures/deezer.artist.bio.empty.json")
			httpClient.bioResponses["xx"] = &http.Response{Body: fXx, StatusCode: 200}

			_, err := agent.GetArtistBiography(ctx, "", "Michael Jackson", "")

			Expect(err).To(MatchError(agents.ErrNotFound))
			Expect(httpClient.bioRequestCount).To(Equal(2))
		})
	})
})

// langAwareHttpClient is a mock HTTP client that returns different responses based on the Accept-Language header
type langAwareHttpClient struct {
	searchResponse  *http.Response
	jwtResponse     *http.Response
	bioResponses    map[string]*http.Response
	bioRequests     []*http.Request
	bioRequestCount int
}

func newLangAwareHttpClient() *langAwareHttpClient {
	return &langAwareHttpClient{
		bioResponses: make(map[string]*http.Response),
		bioRequests:  make([]*http.Request, 0),
	}
}

func (c *langAwareHttpClient) Do(req *http.Request) (*http.Response, error) {
	// Handle search artist request
	if req.URL.Host == "api.deezer.com" && req.URL.Path == "/search/artist" {
		if c.searchResponse != nil {
			return c.searchResponse, nil
		}
		return &http.Response{
			StatusCode: 200,
			Body:       io.NopCloser(bytes.NewBufferString(`{"data":[],"total":0}`)),
		}, nil
	}

	// Handle JWT token request
	if req.URL.Host == "auth.deezer.com" && req.URL.Path == "/login/anonymous" {
		if c.jwtResponse != nil {
			return c.jwtResponse, nil
		}
		return &http.Response{
			StatusCode: 500,
			Body:       io.NopCloser(bytes.NewBufferString(`{"error":"no mock"}`)),
		}, nil
	}

	// Handle bio request (GraphQL API)
	if req.URL.Host == "pipe.deezer.com" && req.URL.Path == "/api" {
		c.bioRequestCount++
		c.bioRequests = append(c.bioRequests, req)
		lang := req.Header.Get("Accept-Language")
		if resp, ok := c.bioResponses[lang]; ok {
			return resp, nil
		}
		// Return empty bio by default
		return &http.Response{
			StatusCode: 200,
			Body:       io.NopCloser(bytes.NewBufferString(`{"data":{"artist":{"bio":{"full":""}}}}`)),
		}, nil
	}

	panic("URL not mocked: " + req.URL.String())
}
