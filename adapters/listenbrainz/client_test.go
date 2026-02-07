package listenbrainz

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"

	"github.com/navidrome/navidrome/conf"
	"github.com/navidrome/navidrome/conf/configtest"
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
		baseUrl := "https://api.listenbrainz.org/1/metadata/artist?"
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
		baseUrl := "https://api.listenbrainz.org/1/popularity/top-recordings-for-artist/"

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
					DurationMs:    211912,
					RecordingName: "world.execute(me);",
					RecordingMbid: "9980309d-3480-4e7e-89ce-fce971a452be",
					ReleaseName:   "Miracle Milk",
					ReleaseMBID:   "38a8f6e1-0e34-4418-a89d-78240a367408",
				},
				{
					ArtistName:    "Mili",
					ArtistMBIDs:   []string{"d2a92ee2-27ce-4e71-bfc5-12e34fe8ef56"},
					DurationMs:    174000,
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
					DurationMs:    211912,
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

	Context("getSimilarArtists", func() {
		var algorithm string

		BeforeEach(func() {
			algorithm = "session_based_days_9000_session_300_contribution_5_threshold_15_limit_50_skip_30"
			DeferCleanup(configtest.SetupConfig())
		})

		getUrl := func(mbid string) string {
			return fmt.Sprintf("https://labs.api.listenbrainz.org/similar-artists/json?algorithm=%s&artist_mbids=%s", algorithm, mbid)
		}

		mbid := "db92a151-1ac2-438b-bc43-b82e149ddd50"

		It("handles a malformed request with status code", func() {
			httpClient.Res = http.Response{
				Body:       io.NopCloser(bytes.NewBufferString(`Bad request`)),
				StatusCode: 400,
			}
			_, err := client.getSimilarArtists(context.Background(), "1", 2)
			Expect(err.Error()).To(Equal("ListenBrainz: HTTP Error, Status: (400)"))
			Expect(httpClient.SavedRequest.Method).To(Equal(http.MethodGet))
			Expect(httpClient.SavedRequest.URL.String()).To(Equal(getUrl("1")))
			Expect(httpClient.SavedRequest.Header.Get("Content-Type")).To(Equal("application/json; charset=UTF-8"))
		})

		It("handles real data properly", func() {
			f, _ := os.Open("tests/fixtures/listenbrainz.labs.similar-artists.json")
			httpClient.Res = http.Response{Body: f, StatusCode: 200}
			resp, err := client.getSimilarArtists(context.Background(), mbid, 2)
			Expect(err).ToNot(HaveOccurred())
			Expect(httpClient.SavedRequest.Method).To(Equal(http.MethodGet))
			Expect(httpClient.SavedRequest.URL.String()).To(Equal(getUrl(mbid)))
			Expect(httpClient.SavedRequest.Header.Get("Content-Type")).To(Equal("application/json; charset=UTF-8"))
			Expect(resp).To(Equal([]artist{
				{MBID: "f27ec8db-af05-4f36-916e-3d57f91ecf5e", Name: "Michael Jackson", Score: 800},
				{MBID: "7364dea6-ca9a-48e3-be01-b44ad0d19897", Name: "a-ha", Score: 792},
			}))
		})

		It("truncates data when requested", func() {
			f, _ := os.Open("tests/fixtures/listenbrainz.labs.similar-artists.json")
			httpClient.Res = http.Response{Body: f, StatusCode: 200}
			resp, err := client.getSimilarArtists(context.Background(), "db92a151-1ac2-438b-bc43-b82e149ddd50", 1)
			Expect(err).ToNot(HaveOccurred())
			Expect(httpClient.SavedRequest.Method).To(Equal(http.MethodGet))
			Expect(httpClient.SavedRequest.URL.String()).To(Equal(getUrl(mbid)))
			Expect(httpClient.SavedRequest.Header.Get("Content-Type")).To(Equal("application/json; charset=UTF-8"))
			Expect(resp).To(Equal([]artist{
				{MBID: "f27ec8db-af05-4f36-916e-3d57f91ecf5e", Name: "Michael Jackson", Score: 800},
			}))
		})

		It("fetches a different endpoint when algorithm changes", func() {
			algorithm = "session_based_days_1825_session_300_contribution_3_threshold_10_limit_100_filter_True_skip_30"
			conf.Server.ListenBrainz.ArtistAlgorithm = algorithm

			f, _ := os.Open("tests/fixtures/listenbrainz.labs.similar-artists.json")
			httpClient.Res = http.Response{Body: f, StatusCode: 200}
			resp, err := client.getSimilarArtists(context.Background(), mbid, 2)
			Expect(err).ToNot(HaveOccurred())
			Expect(httpClient.SavedRequest.Method).To(Equal(http.MethodGet))
			Expect(httpClient.SavedRequest.URL.String()).To(Equal(getUrl(mbid)))
			Expect(httpClient.SavedRequest.Header.Get("Content-Type")).To(Equal("application/json; charset=UTF-8"))
			Expect(resp).To(Equal([]artist{
				{MBID: "f27ec8db-af05-4f36-916e-3d57f91ecf5e", Name: "Michael Jackson", Score: 800},
				{MBID: "7364dea6-ca9a-48e3-be01-b44ad0d19897", Name: "a-ha", Score: 792},
			}))
		})
	})

	Context("getSimilarRecordings", func() {
		var algorithm string

		BeforeEach(func() {
			algorithm = "session_based_days_9000_session_300_contribution_5_threshold_15_limit_50_skip_30"
			DeferCleanup(configtest.SetupConfig())
		})

		getUrl := func(mbid string) string {
			return fmt.Sprintf("https://labs.api.listenbrainz.org/similar-recordings/json?algorithm=%s&recording_mbids=%s", algorithm, mbid)
		}

		mbid := "8f3471b5-7e6a-48da-86a9-c1c07a0f47ae"

		It("handles a malformed request with status code", func() {
			httpClient.Res = http.Response{
				Body:       io.NopCloser(bytes.NewBufferString(`Bad request`)),
				StatusCode: 400,
			}
			_, err := client.getSimilarRecordings(context.Background(), "1", 2)
			Expect(err.Error()).To(Equal("ListenBrainz: HTTP Error, Status: (400)"))
			Expect(httpClient.SavedRequest.Method).To(Equal(http.MethodGet))
			Expect(httpClient.SavedRequest.URL.String()).To(Equal(getUrl("1")))
			Expect(httpClient.SavedRequest.Header.Get("Content-Type")).To(Equal("application/json; charset=UTF-8"))
		})

		It("handles real data properly", func() {
			f, _ := os.Open("tests/fixtures/listenbrainz.labs.similar-recordings.json")
			httpClient.Res = http.Response{Body: f, StatusCode: 200}
			resp, err := client.getSimilarRecordings(context.Background(), mbid, 2)
			Expect(err).ToNot(HaveOccurred())
			Expect(httpClient.SavedRequest.Method).To(Equal(http.MethodGet))
			Expect(httpClient.SavedRequest.URL.String()).To(Equal(getUrl(mbid)))
			Expect(httpClient.SavedRequest.Header.Get("Content-Type")).To(Equal("application/json; charset=UTF-8"))
			Expect(resp).To(Equal([]recording{
				{
					MBID:        "12f65dca-de8f-43fe-a65d-f12a02aaadf3",
					Name:        "Take On Me",
					Artist:      "a‐ha",
					ReleaseName: "Hunting High and Low",
					ReleaseMBID: "4ec07fe8-e7c6-3106-a0aa-fdf92f13f7fc",
					Score:       124,
				},
				{
					MBID:        "80033c72-aa19-4ba8-9227-afb075fec46e",
					Name:        "Wake Me Up Before You Go‐Go",
					Artist:      "Wham!",
					ReleaseName: "Make It Big",
					ReleaseMBID: "c143d542-48dc-446b-b523-1762da721638",
					Score:       65,
				},
			}))
		})

		It("truncates data when requested", func() {
			f, _ := os.Open("tests/fixtures/listenbrainz.labs.similar-recordings.json")
			httpClient.Res = http.Response{Body: f, StatusCode: 200}
			resp, err := client.getSimilarRecordings(context.Background(), mbid, 1)
			Expect(err).ToNot(HaveOccurred())
			Expect(httpClient.SavedRequest.Method).To(Equal(http.MethodGet))
			Expect(httpClient.SavedRequest.URL.String()).To(Equal(getUrl(mbid)))
			Expect(httpClient.SavedRequest.Header.Get("Content-Type")).To(Equal("application/json; charset=UTF-8"))
			Expect(resp).To(Equal([]recording{
				{
					MBID:        "12f65dca-de8f-43fe-a65d-f12a02aaadf3",
					Name:        "Take On Me",
					Artist:      "a‐ha",
					ReleaseName: "Hunting High and Low",
					ReleaseMBID: "4ec07fe8-e7c6-3106-a0aa-fdf92f13f7fc",
					Score:       124,
				},
			}))
		})

		It("properly sorts by score and truncates duplicates", func() {
			f, _ := os.Open("tests/fixtures/listenbrainz.labs.similar-recordings-real-out-of-order.json")
			httpClient.Res = http.Response{Body: f, StatusCode: 200}
			// There are actually 5 items. The dedup should happen FIRST
			resp, err := client.getSimilarRecordings(context.Background(), mbid, 4)
			Expect(err).ToNot(HaveOccurred())
			Expect(httpClient.SavedRequest.Method).To(Equal(http.MethodGet))
			Expect(httpClient.SavedRequest.URL.String()).To(Equal(getUrl(mbid)))
			Expect(httpClient.SavedRequest.Header.Get("Content-Type")).To(Equal("application/json; charset=UTF-8"))
			Expect(resp).To(Equal([]recording{
				{
					MBID:        "12f65dca-de8f-43fe-a65d-f12a02aaadf3",
					Name:        "Take On Me",
					Artist:      "a‐ha",
					ReleaseName: "Hunting High and Low",
					ReleaseMBID: "4ec07fe8-e7c6-3106-a0aa-fdf92f13f7fc",
					Score:       124,
				},
				{
					MBID:        "e4b347be-ecb2-44ff-aaa8-3d4c517d7ea5",
					Name:        "Everybody Wants to Rule the World",
					Artist:      "Tears for Fears",
					ReleaseName: "Songs From the Big Chair",
					ReleaseMBID: "21f19b06-81f1-347a-add5-5d0c77696597",
					Score:       68,
				},
				{
					MBID:        "80033c72-aa19-4ba8-9227-afb075fec46e",
					Name:        "Wake Me Up Before You Go‐Go",
					Artist:      "Wham!",
					ReleaseName: "Make It Big",
					ReleaseMBID: "c143d542-48dc-446b-b523-1762da721638",
					Score:       65,
				},
				{
					MBID:        "ef4c6855-949e-4e22-b41e-8e0a2d372d5f",
					Name:        "Tainted Love",
					Artist:      "Soft Cell",
					ReleaseName: "Non-Stop Erotic Cabaret",
					ReleaseMBID: "1acaa870-6e0c-4b6e-9e91-fdec4e5ea4b1",
					Score:       61,
				},
			}))
		})

		It("uses a different algorithm when configured", func() {
			algorithm = "session_based_days_180_session_300_contribution_5_threshold_15_limit_50_skip_30"
			conf.Server.ListenBrainz.TrackAlgorithm = algorithm

			f, _ := os.Open("tests/fixtures/listenbrainz.labs.similar-recordings.json")
			httpClient.Res = http.Response{Body: f, StatusCode: 200}
			resp, err := client.getSimilarRecordings(context.Background(), mbid, 1)
			Expect(err).ToNot(HaveOccurred())
			Expect(httpClient.SavedRequest.Method).To(Equal(http.MethodGet))
			Expect(httpClient.SavedRequest.URL.String()).To(Equal(getUrl(mbid)))
			Expect(httpClient.SavedRequest.Header.Get("Content-Type")).To(Equal("application/json; charset=UTF-8"))
			Expect(resp).To(Equal([]recording{
				{
					MBID:        "12f65dca-de8f-43fe-a65d-f12a02aaadf3",
					Name:        "Take On Me",
					Artist:      "a‐ha",
					ReleaseName: "Hunting High and Low",
					ReleaseMBID: "4ec07fe8-e7c6-3106-a0aa-fdf92f13f7fc",
					Score:       124,
				},
			}))
		})
	})
})
