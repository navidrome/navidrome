package listenbrainz

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"

	"github.com/navidrome/navidrome/core/external_playlists"
	"github.com/navidrome/navidrome/model"
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

		It("parses playlists response properly", func() {
			var response listenBrainzResponse
			f, _ := os.ReadFile("tests/fixtures/listenbrainz.playlists.json")
			date, _ := time.Parse(time.RFC3339Nano, "2023-03-18T11:01:46.635538+00:00")
			modified, _ := time.Parse(time.RFC3339Nano, "2023-03-18T11:01:46.635538+00:00")

			err := json.Unmarshal(f, &response)
			Expect(err).ToNot(HaveOccurred())
			Expect(response.PlaylistCount).To(Equal(3))
			Expect(response.Playlists).To(HaveLen(3))
			Expect(response.Playlists[0]).To(Equal(overallPlaylist{
				Playlist: lbPlaylist{
					Annotation: "Daily jams playlist!",
					Creator:    "troi-bot",
					Date:       date,
					Identifier: "https://listenbrainz.org/playlist/00000000-0000-0000-0000-000000000000",
					Title:      "Daily Jams for example, 2023-03-18 Sat",
					Extension: plsExtension{
						Extension: playlistExtension{
							Collaborators: []string{"example"},
							CreatedFor:    "example",
							LastModified:  modified,
							Public:        true,
						},
					},
					Tracks: []lbTrack{},
				},
			}))
		})

		It("parses playlist response properly", func() {
			var response listenBrainzResponse
			f, _ := os.ReadFile("tests/fixtures/listenbrainz.playlist.json")

			date, _ := time.Parse(time.RFC3339Nano, "2023-01-05T04:12:45.668903+00:00")
			modified, _ := time.Parse(time.RFC3339Nano, "2023-01-05T04:12:45.668903+00:00")

			err := json.Unmarshal(f, &response)
			Expect(err).ToNot(HaveOccurred())
			Expect(response.Playlist).To(Equal(lbPlaylist{
				Annotation: "<p>\n              This playlist contains the top tracks for example that were first listened to in 2022.\n              </p>\n              <p>\n                For more information on how this playlist is generated, please see our\n                <a href=\"https://musicbrainz.org/doc/YIM2022Playlists\">Year in Music 2022 Playlists</a> page.\n              </p>\n           ",
				Creator:    "troi-bot",
				Date:       date,
				Identifier: "https://listenbrainz.org/playlist/00000000-0000-0000-0000-000000000004",
				Title:      "Top Discoveries of 2022 for example",
				Extension: plsExtension{
					Extension: playlistExtension{
						Collaborators: []string{"example"},
						CreatedFor:    "example",
						LastModified:  modified,
						Public:        true,
					},
				},
				Tracks: []lbTrack{
					{
						Creator:    "Poets of the Fall",
						Identifier: "https://musicbrainz.org/recording/684bedbb-78d7-4946-9038-5402d5fa83b0",
						Title:      "Requiem for My Harlequin",
					},
					{
						Creator:    "Old Gods of Asgard",
						Identifier: "https://musicbrainz.org/recording/9f42783a-423b-4ed6-8a10-fdf4cb44456f",
						Title:      "Take Control",
					},
				},
			}))
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
						TrackNumber:  1,
						TrackMbzID:   "mbz-123",
						ArtistMbzIDs: []string{"mbz-789"},
						ReleaseMbID:  "mbz-456",
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

	Describe("getPlaylists", func() {
		Describe("successful responses", func() {
			BeforeEach(func() {
				f, _ := os.Open("tests/fixtures/listenbrainz.playlists.json")

				httpClient.Res = http.Response{
					Body:       f,
					StatusCode: 200,
				}
			})

			test := func(offset, count int, plsType, extra string) {
				It("Handles "+plsType+" correctly", func() {
					resp, err := client.getPlaylists(context.Background(), offset, count, "LB-TOKEN", "example", plsType)
					Expect(err).To(BeNil())
					Expect(resp.Playlists).To(HaveLen(3))
					Expect(httpClient.SavedRequest.URL.String()).To(Equal(fmt.Sprintf("BASE_URL/user/example/playlists%s?count=%d&offset=%d", extra, count, offset)))
					Expect(httpClient.SavedRequest.Method).To(Equal(http.MethodGet))
					Expect(httpClient.SavedRequest.Header.Get("Authorization")).To(Equal("Token LB-TOKEN"))
					Expect(httpClient.SavedRequest.Header.Get("Content-Type")).To(Equal("application/json; charset=UTF-8"))
				})
			}

			test(10, 30, "user", "")
			test(11, 50, "created", "/createdfor")
			test(0, 25, "collab", "/collaborator")
		})

		It("fails when provided an unsupported type", func() {
			resp, err := client.getPlaylists(context.Background(), 0, 25, "LB-TOKEN", "example", "notatype")
			Expect(resp).To(BeNil())
			Expect(err).To(Equal(external_playlists.ErrorUnsupportedType))
		})
	})

	Describe("getPlaylist", func() {
		It("Should fetch playlist properly", func() {
			f, _ := os.Open("tests/fixtures/listenbrainz.playlist.json")

			httpClient.Res = http.Response{
				Body:       f,
				StatusCode: 200,
			}

			list, err := client.getPlaylist(context.Background(), "LB-TOKEN", "00000000-0000-0000-0000-000000000004")
			Expect(err).To(BeNil())
			Expect(list.Playlist.Tracks).To(HaveLen(2))
			Expect(httpClient.SavedRequest.URL.String()).To(Equal("BASE_URL/playlist/00000000-0000-0000-0000-000000000004"))
			Expect(httpClient.SavedRequest.Method).To(Equal(http.MethodGet))
			Expect(httpClient.SavedRequest.Header.Get("Authorization")).To(Equal("Token LB-TOKEN"))
			Expect(httpClient.SavedRequest.Header.Get("Content-Type")).To(Equal("application/json; charset=UTF-8"))
		})

		It("Returns correct error on 404", func() {
			httpClient.Res = http.Response{
				Body:       io.NopCloser(bytes.NewBufferString(`{"Code":404}`)),
				StatusCode: 404,
			}

			resp, err := client.getPlaylist(context.Background(), "LB-TOKEN", "00000000-0000-0000-0000-000000000004")
			Expect(resp).To(BeNil())
			Expect(err).To(Equal(model.ErrNotFound))
		})
	})
})
