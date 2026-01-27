package listenbrainz

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
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
		client = newClient("BASE_URL/", httpClient)
	})

	Describe("listenBrainzResponse", func() {
		It("parses a response properly", func() {
			var response listenBrainzResponse
			err := json.Unmarshal([]byte(`{"code": 200, "message": "Message", "user_name": "UserName", "valid": true, "status": "ok", "error": "Error"}`), &response)

			Expect(err).ToNot(HaveOccurred())
			Expect(response.Code).To(Equal(200))
			Expect(response.Message).To(Equal("Message"))
			Expect(response.UserName).To(Equal("UserName"))
			Expect(response.Valid).To(BeTrue())
			Expect(response.Status).To(Equal("ok"))
			Expect(response.Error).To(Equal("Error"))
		})
	})

	Describe("validateToken", func() {
		BeforeEach(func() {
			httpClient.Res = http.Response{
				Body:       io.NopCloser(bytes.NewBufferString(`{"code": 200, "message": "Token valid.", "user_name": "ListenBrainzUser", "valid": true}`)),
				StatusCode: 200,
			}
		})

		It("formats the request properly", func() {
			_, err := client.validateToken(context.Background(), "LB-TOKEN")
			Expect(err).ToNot(HaveOccurred())
			Expect(httpClient.SavedRequest.Method).To(Equal(http.MethodGet))
			Expect(httpClient.SavedRequest.URL.String()).To(Equal("BASE_URL/validate-token"))
			Expect(httpClient.SavedRequest.Header.Get("Authorization")).To(Equal("Token LB-TOKEN"))
			Expect(httpClient.SavedRequest.Header.Get("Content-Type")).To(Equal("application/json; charset=UTF-8"))
		})

		It("parses and returns the response", func() {
			res, err := client.validateToken(context.Background(), "LB-TOKEN")
			Expect(err).ToNot(HaveOccurred())
			Expect(res.Valid).To(Equal(true))
			Expect(res.UserName).To(Equal("ListenBrainzUser"))
		})
	})

	Context("with listenInfo", func() {
		var li listenInfo
		BeforeEach(func() {
			httpClient.Res = http.Response{
				Body:       io.NopCloser(bytes.NewBufferString(`{"status": "ok"}`)),
				StatusCode: 200,
			}
			li = listenInfo{
				TrackMetadata: trackMetadata{
					ArtistName:  "Track Artist",
					TrackName:   "Track Title",
					ReleaseName: "Track Album",
					AdditionalInfo: additionalInfo{
						TrackNumber:   1,
						ArtistNames:   []string{"Artist 1", "Artist 2"},
						ArtistMBIDs:   []string{"mbz-789", "mbz-012"},
						RecordingMBID: "mbz-123",
						ReleaseMBID:   "mbz-456",
						DurationMs:    142200,
					},
				},
			}
		})

		Describe("updateNowPlaying", func() {
			It("formats the request properly", func() {
				Expect(client.updateNowPlaying(context.Background(), "LB-TOKEN", li)).To(Succeed())
				Expect(httpClient.SavedRequest.Method).To(Equal(http.MethodPost))
				Expect(httpClient.SavedRequest.URL.String()).To(Equal("BASE_URL/submit-listens"))
				Expect(httpClient.SavedRequest.Header.Get("Authorization")).To(Equal("Token LB-TOKEN"))
				Expect(httpClient.SavedRequest.Header.Get("Content-Type")).To(Equal("application/json; charset=UTF-8"))

				body, _ := io.ReadAll(httpClient.SavedRequest.Body)
				f, _ := os.ReadFile("tests/fixtures/listenbrainz.nowplaying.request.json")
				Expect(body).To(MatchJSON(f))
			})
		})

		Describe("scrobble", func() {
			BeforeEach(func() {
				li.ListenedAt = 1635000000
			})

			It("formats the request properly", func() {
				Expect(client.scrobble(context.Background(), "LB-TOKEN", li)).To(Succeed())
				Expect(httpClient.SavedRequest.Method).To(Equal(http.MethodPost))
				Expect(httpClient.SavedRequest.URL.String()).To(Equal("BASE_URL/submit-listens"))
				Expect(httpClient.SavedRequest.Header.Get("Authorization")).To(Equal("Token LB-TOKEN"))
				Expect(httpClient.SavedRequest.Header.Get("Content-Type")).To(Equal("application/json; charset=UTF-8"))

				body, _ := io.ReadAll(httpClient.SavedRequest.Body)
				f, _ := os.ReadFile("tests/fixtures/listenbrainz.scrobble.request.json")
				Expect(body).To(MatchJSON(f))
			})
		})
	})

	Context("getArtistUrl", func() {
		baseUrl := "BASE_URL/metadata/artist?"
		It("handles a malformed request with status code", func() {
			httpClient.Res = http.Response{
				Body:       io.NopCloser(bytes.NewBufferString(`{"code": 400,"error": "artist mbid 1 is not valid."}`)),
				StatusCode: 400,
			}
			_, err := client.getArtistUrl(context.Background(), "1")
			Expect(err.Error()).To(Equal("ListenBrainz error(400): artist mbid 1 is not valid."))
			Expect(httpClient.SavedRequest.Method).To(Equal(http.MethodGet))
			Expect(httpClient.SavedRequest.URL.String()).To(Equal(baseUrl + "artist_mbids=1"))
			Expect(httpClient.SavedRequest.Header.Get("Content-Type")).To(Equal("application/json; charset=UTF-8"))
		})

		It("handles a malformed request without meaningful body", func() {
			httpClient.Res = http.Response{
				Body:       io.NopCloser(bytes.NewBufferString(``)),
				StatusCode: 501,
			}
			_, err := client.getArtistUrl(context.Background(), "1")
			Expect(err.Error()).To(Equal("ListenBrainz: HTTP Error, Status: (501)"))
			Expect(httpClient.SavedRequest.Method).To(Equal(http.MethodGet))
			Expect(httpClient.SavedRequest.URL.String()).To(Equal(baseUrl + "artist_mbids=1"))
			Expect(httpClient.SavedRequest.Header.Get("Content-Type")).To(Equal("application/json; charset=UTF-8"))
		})

		It("It returns not found when the artist has no official homepage", func() {
			f, _ := os.Open("tests/fixtures/listenbrainz.artist.metadata.no_homepage.json")
			httpClient.Res = http.Response{Body: f, StatusCode: 200}
			_, err := client.getArtistUrl(context.Background(), "7c2cc610-f998-43ef-a08f-dae3344b8973")
			Expect(err.Error()).To(Equal("listenbrainz: not found"))
			Expect(httpClient.SavedRequest.Method).To(Equal(http.MethodGet))
			Expect(httpClient.SavedRequest.URL.String()).To(Equal(baseUrl + "artist_mbids=7c2cc610-f998-43ef-a08f-dae3344b8973"))
			Expect(httpClient.SavedRequest.Header.Get("Content-Type")).To(Equal("application/json; charset=UTF-8"))
		})

		It("It returns data when the artist has a homepage", func() {
			f, _ := os.Open("tests/fixtures/listenbrainz.artist.metadata.homepage.json")
			httpClient.Res = http.Response{Body: f, StatusCode: 200}
			url, err := client.getArtistUrl(context.Background(), "d2a92ee2-27ce-4e71-bfc5-12e34fe8ef56")
			Expect(err).ToNot(HaveOccurred())
			Expect(url).To(Equal("http://projectmili.com/"))
			Expect(httpClient.SavedRequest.Method).To(Equal(http.MethodGet))
			Expect(httpClient.SavedRequest.URL.String()).To(Equal(baseUrl + "artist_mbids=d2a92ee2-27ce-4e71-bfc5-12e34fe8ef56"))
			Expect(httpClient.SavedRequest.Header.Get("Content-Type")).To(Equal("application/json; charset=UTF-8"))
		})
	})

	Context("getArtistTopSongs", func() {
		baseUrl := "BASE_URL/popularity/top-recordings-for-artist/"

		It("handles a malformed request with status code", func() {
			httpClient.Res = http.Response{
				Body:       io.NopCloser(bytes.NewBufferString(`{"code":400,"error":"artist_mbid: '1' is not a valid uuid"}`)),
				StatusCode: 400,
			}
			_, err := client.getArtistTopSongs(context.Background(), "1", 50)
			Expect(err.Error()).To(Equal("ListenBrainz error(400): artist_mbid: '1' is not a valid uuid"))
			Expect(httpClient.SavedRequest.Method).To(Equal(http.MethodGet))
			Expect(httpClient.SavedRequest.URL.String()).To(Equal(baseUrl + "1"))
			Expect(httpClient.SavedRequest.Header.Get("Content-Type")).To(Equal("application/json; charset=UTF-8"))
		})

		It("handles a malformed request without standard body", func() {
			httpClient.Res = http.Response{
				Body:       io.NopCloser(bytes.NewBufferString(``)),
				StatusCode: 500,
			}
			_, err := client.getArtistTopSongs(context.Background(), "1", 1)
			Expect(err.Error()).To(Equal("ListenBrainz: HTTP Error, Status: (500)"))
			Expect(httpClient.SavedRequest.Method).To(Equal(http.MethodGet))
			Expect(httpClient.SavedRequest.URL.String()).To(Equal(baseUrl + "1"))
			Expect(httpClient.SavedRequest.Header.Get("Content-Type")).To(Equal("application/json; charset=UTF-8"))
		})

		It("It returns all tracks when given the opportunity", func() {
			f, _ := os.Open("tests/fixtures/listenbrainz.popularity.json")
			httpClient.Res = http.Response{Body: f, StatusCode: 200}
			data, err := client.getArtistTopSongs(context.Background(), "d2a92ee2-27ce-4e71-bfc5-12e34fe8ef56", 5)
			Expect(err).ToNot(HaveOccurred())
			Expect(data).To(Equal([]trackInfo{
				{
					ArtistName:    "Mili",
					ArtistMBIDs:   []string{"d2a92ee2-27ce-4e71-bfc5-12e34fe8ef56"},
					RecordingName: "world.execute(me);",
					RecordingMbid: "9980309d-3480-4e7e-89ce-fce971a452be",
					ReleaseName:   "Miracle Milk",
					ReleaseMBID:   "38a8f6e1-0e34-4418-a89d-78240a367408",
				},
				{
					ArtistName:    "Mili",
					ArtistMBIDs:   []string{"d2a92ee2-27ce-4e71-bfc5-12e34fe8ef56"},
					RecordingName: "String Theocracy",
					RecordingMbid: "afa2c83d-b17f-4029-b9da-790ea9250cf9",
					ReleaseName:   "String Theocracy",
					ReleaseMBID:   "d79a38e3-7016-4f39-a31a-f495ce914b8e",
				},
			}))
			Expect(httpClient.SavedRequest.Method).To(Equal(http.MethodGet))
			Expect(httpClient.SavedRequest.URL.String()).To(Equal(baseUrl + "d2a92ee2-27ce-4e71-bfc5-12e34fe8ef56"))
			Expect(httpClient.SavedRequest.Header.Get("Content-Type")).To(Equal("application/json; charset=UTF-8"))
		})

		It("It returns a subset of tracks when allowed", func() {
			f, _ := os.Open("tests/fixtures/listenbrainz.popularity.json")
			httpClient.Res = http.Response{Body: f, StatusCode: 200}
			data, err := client.getArtistTopSongs(context.Background(), "d2a92ee2-27ce-4e71-bfc5-12e34fe8ef56", 1)
			Expect(err).ToNot(HaveOccurred())
			Expect(data).To(Equal([]trackInfo{
				{
					ArtistName:    "Mili",
					ArtistMBIDs:   []string{"d2a92ee2-27ce-4e71-bfc5-12e34fe8ef56"},
					RecordingName: "world.execute(me);",
					RecordingMbid: "9980309d-3480-4e7e-89ce-fce971a452be",
					ReleaseName:   "Miracle Milk",
					ReleaseMBID:   "38a8f6e1-0e34-4418-a89d-78240a367408",
				},
			}))
			Expect(httpClient.SavedRequest.Method).To(Equal(http.MethodGet))
			Expect(httpClient.SavedRequest.URL.String()).To(Equal(baseUrl + "d2a92ee2-27ce-4e71-bfc5-12e34fe8ef56"))
			Expect(httpClient.SavedRequest.Header.Get("Content-Type")).To(Equal("application/json; charset=UTF-8"))
		})
	})
})
