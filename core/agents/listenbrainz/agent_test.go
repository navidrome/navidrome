package listenbrainz

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"os"
	"time"

	"github.com/navidrome/navidrome/consts"
	"github.com/navidrome/navidrome/core/agents"
	"github.com/navidrome/navidrome/core/external_playlists"
	"github.com/navidrome/navidrome/core/scrobbler"
	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/tests"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gstruct"
)

var _ = Describe("listenBrainzAgent", func() {
	var ds model.DataStore
	var ctx context.Context
	var agent *listenBrainzAgent
	var httpClient *tests.FakeHttpClient
	var track *model.MediaFile

	BeforeEach(func() {
		ds = &tests.MockDataStore{}
		ctx = context.Background()
		_ = ds.UserProps(ctx).Put("user-1", sessionKeyProperty, "SK-1")
		_ = ds.UserProps(ctx).Put("user-1", sessionKeyProperty+agents.UserSuffix, "example")
		httpClient = &tests.FakeHttpClient{}
		agent = listenBrainzConstructor(ds)
		agent.client = newClient("http://localhost:8080", httpClient)
		track = &model.MediaFile{
			ID:          "123",
			Title:       "Track Title",
			Album:       "Track Album",
			Artist:      "Track Artist",
			TrackNumber: 1,
			MbzTrackID:  "mbz-123",
			MbzAlbumID:  "mbz-456",
			MbzArtistID: "mbz-789",
		}
	})

	Describe("formatListen", func() {
		It("constructs the listenInfo properly", func() {
			var idArtistId = func(element interface{}) string {
				return element.(string)
			}

			lr := agent.formatListen(track)
			Expect(lr).To(MatchAllFields(Fields{
				"ListenedAt": Equal(0),
				"TrackMetadata": MatchAllFields(Fields{
					"ArtistName":  Equal(track.Artist),
					"TrackName":   Equal(track.Title),
					"ReleaseName": Equal(track.Album),
					"AdditionalInfo": MatchAllFields(Fields{
						"SubmissionClient":        Equal(consts.AppName),
						"SubmissionClientVersion": Equal(consts.Version),
						"TrackNumber":             Equal(track.TrackNumber),
						"TrackMbzID":              Equal(track.MbzTrackID),
						"ReleaseMbID":             Equal(track.MbzAlbumID),
						"ArtistMbzIDs": MatchAllElements(idArtistId, Elements{
							"mbz-789": Equal(track.MbzArtistID),
						}),
					}),
				}),
			}))
		})
	})

	Describe("NowPlaying", func() {
		It("updates NowPlaying successfully", func() {
			httpClient.Res = http.Response{Body: io.NopCloser(bytes.NewBufferString(`{"status": "ok"}`)), StatusCode: 200}

			err := agent.NowPlaying(ctx, "user-1", track)
			Expect(err).ToNot(HaveOccurred())
		})

		It("returns ErrNotAuthorized if user is not linked", func() {
			err := agent.NowPlaying(ctx, "user-2", track)
			Expect(err).To(MatchError(scrobbler.ErrNotAuthorized))
		})
	})

	Describe("Scrobble", func() {
		var sc scrobbler.Scrobble

		BeforeEach(func() {
			sc = scrobbler.Scrobble{MediaFile: *track, TimeStamp: time.Now()}
		})

		It("sends a Scrobble successfully", func() {
			httpClient.Res = http.Response{Body: io.NopCloser(bytes.NewBufferString(`{"status": "ok"}`)), StatusCode: 200}

			err := agent.Scrobble(ctx, "user-1", sc)
			Expect(err).ToNot(HaveOccurred())
		})

		It("sets the Timestamp properly", func() {
			httpClient.Res = http.Response{Body: io.NopCloser(bytes.NewBufferString(`{"status": "ok"}`)), StatusCode: 200}

			err := agent.Scrobble(ctx, "user-1", sc)
			Expect(err).ToNot(HaveOccurred())

			decoder := json.NewDecoder(httpClient.SavedRequest.Body)
			var lr listenBrainzRequestBody
			err = decoder.Decode(&lr)

			Expect(err).ToNot(HaveOccurred())
			Expect(lr.Payload[0].ListenedAt).To(Equal(int(sc.TimeStamp.Unix())))
		})

		It("returns ErrNotAuthorized if user is not linked", func() {
			err := agent.Scrobble(ctx, "user-2", sc)
			Expect(err).To(MatchError(scrobbler.ErrNotAuthorized))
		})

		It("returns ErrRetryLater on error 503", func() {
			httpClient.Res = http.Response{
				Body:       io.NopCloser(bytes.NewBufferString(`{"code": 503, "error": "Cannot submit listens to queue, please try again later."}`)),
				StatusCode: 503,
			}

			err := agent.Scrobble(ctx, "user-1", sc)
			Expect(err).To(MatchError(scrobbler.ErrRetryLater))
		})

		It("returns ErrRetryLater on error 500", func() {
			httpClient.Res = http.Response{
				Body:       io.NopCloser(bytes.NewBufferString(`{"code": 500, "error": "Something went wrong. Please try again."}`)),
				StatusCode: 500,
			}

			err := agent.Scrobble(ctx, "user-1", sc)
			Expect(err).To(MatchError(scrobbler.ErrRetryLater))
		})

		It("returns ErrRetryLater on http errors", func() {
			httpClient.Res = http.Response{
				Body:       io.NopCloser(bytes.NewBufferString(`Bad Gateway`)),
				StatusCode: 500,
			}

			err := agent.Scrobble(ctx, "user-1", sc)
			Expect(err).To(MatchError(scrobbler.ErrRetryLater))
		})

		It("returns ErrUnrecoverable on other errors", func() {
			httpClient.Res = http.Response{
				Body:       io.NopCloser(bytes.NewBufferString(`{"code": 400, "error": "BadRequest: Invalid JSON document submitted."}`)),
				StatusCode: 400,
			}

			err := agent.Scrobble(ctx, "user-1", sc)
			Expect(err).To(MatchError(scrobbler.ErrUnrecoverable))
		})
	})

	Describe("GetPlaylistTypes", func() {
		It("should return data", func() {
			Expect(agent.GetPlaylistTypes()).To(Equal([]string{"user", "collab", "created"}))
		})
	})

	Describe("playlists", func() {
		parseTime := func(t string) time.Time {
			resp, _ := time.Parse(time.RFC3339Nano, t)
			return resp
		}

		Describe("GetPlaylists", func() {
			It("should return playlist", func() {
				f, _ := os.Open("tests/fixtures/listenbrainz.playlists.json")

				httpClient.Res = http.Response{
					Body:       f,
					StatusCode: 200,
				}

				resp, err := agent.GetPlaylists(ctx, 0, 25, "user-1", "user")
				Expect(err).To(BeNil())
				Expect(resp.Total).To(Equal(3))

				match := func(idx int, pls external_playlists.ExternalPlaylist) {
					list := resp.Lists[idx]
					Expect(list.Name).To(Equal(pls.Name))
					Expect(list.Description).To(Equal(pls.Description))
					Expect(list.ID).To(Equal(pls.ID))
					Expect(list.Url).To(Equal(pls.Url))
					Expect(list.Creator).To(Equal(pls.Creator))
					Expect(list.CreatedAt).To(Equal(pls.CreatedAt))
					Expect(list.UpdatedAt).To(Equal(pls.UpdatedAt))
					Expect(list.Existing).To(Equal(pls.Existing))
					Expect(list.Tracks).To(BeNil())
				}

				match(0, external_playlists.ExternalPlaylist{
					Name:        "Daily Jams for example, 2023-03-18 Sat",
					Description: "Daily jams playlist!",
					ID:          "00000000-0000-0000-0000-000000000000",
					Url:         "https://listenbrainz.org/playlist/00000000-0000-0000-0000-000000000000",
					Creator:     "troi-bot",
					CreatedAt:   parseTime("2023-03-18T11:01:46.635538+00:00"),
					UpdatedAt:   parseTime("2023-03-18T11:01:46.635538+00:00"),
					Existing:    false, // This is false, because the agent itself does not check for this property
				})
				match(1, external_playlists.ExternalPlaylist{
					Name:        "Top Discoveries of 2022 for example",
					Description: "<p>\n              This playlist contains the top tracks for example that were first listened to in 2022.\n              </p>\n              <p>\n                For more information on how this playlist is generated, please see our\n                <a href=\"https://musicbrainz.org/doc/YIM2022Playlists\" rel=\"nofollow\">Year in Music 2022 Playlists</a> page.\n              </p>\n           ",
					ID:          "00000000-0000-0000-0000-000000000001",
					Url:         "https://listenbrainz.org/playlist/00000000-0000-0000-0000-000000000001",
					Creator:     "troi-bot",
					CreatedAt:   parseTime("2023-01-05T04:12:45.668903+00:00"),
					UpdatedAt:   parseTime("2023-01-05T04:12:45.668903+00:00"),
					Existing:    false,
				})
				match(2, external_playlists.ExternalPlaylist{
					Name:        "Top Missed Recordings of 2022 for example",
					Description: "<p>\n                This playlist features recordings that were listened to by users similar to example in 2022.\n                It is a discovery playlist that aims to introduce you to new music that other similar users\n                enjoy. It may require more active listening and may contain tracks that are not to your taste.\n              </p>\n              <p>\n                The users similar to you who contributed to this playlist: <a href=\"https://listenbrainz.org/user/example1/\" rel=\"nofollow\">example1</a>, <a href=\"https://listenbrainz.org/user/example2/\" rel=\"nofollow\">example2</a>, <a href=\"https://listenbrainz.org/user/example3/\" rel=\"nofollow\">example3</a>.\n              </p>\n              <p>\n                For more information on how this playlist is generated, please see our\n                <a href=\"https://musicbrainz.org/doc/YIM2022Playlists\" rel=\"nofollow\">Year in Music 2022 Playlists</a> page.\n              </p>\n           ",
					ID:          "00000000-0000-0000-0000-000000000003",
					Url:         "https://listenbrainz.org/playlist/00000000-0000-0000-0000-000000000003",
					Creator:     "troi-bot",
					CreatedAt:   parseTime("2023-01-05T04:12:45.317875+00:00"),
					UpdatedAt:   parseTime("2023-01-05T04:12:45.317875+00:00"),
					Existing:    false,
				})
			})

			It("should error for nonexistent user", func() {
				resp, err := agent.GetPlaylists(ctx, 0, 25, "user-2", "user")
				Expect(resp).To(BeNil())
				Expect(err).To(MatchError(model.ErrNotFound))
			})

			It("should error when trying to fetch token and failing", func() {
				httpClient.Res = http.Response{
					Body:       io.NopCloser(bytes.NewBufferString(`{"Code":400,"Error":"No"}`)),
					StatusCode: 400,
				}
				_ = ds.UserProps(ctx).Delete("user-1", sessionKeyProperty+agents.UserSuffix)
				resp, err := agent.GetPlaylists(ctx, 0, 25, "user-1", "user")
				Expect(resp).To(BeNil())
				Expect(err).To(Equal(&listenBrainzError{
					Code:    400,
					Message: "No",
				}))
			})

			It("should update session key when missing (and then fail)", func() {
				httpClient.Res = http.Response{
					Body:       io.NopCloser(bytes.NewBufferString(`{"code": 200, "message": "Token valid.", "user_name": "ListenBrainzUser", "valid": true}`)),
					StatusCode: 200,
				}

				_ = ds.UserProps(ctx).Delete("user-1", sessionKeyProperty+agents.UserSuffix)
				resp, err := agent.GetPlaylists(ctx, 0, 25, "user-1", "user")
				Expect(resp).To(BeNil())
				Expect(err).ToNot(BeNil())

				key, err := ds.UserProps(ctx).Get("user-1", sessionKeyProperty+agents.UserSuffix)
				Expect(err).To(BeNil())
				Expect(key).To(Equal("ListenBrainzUser"))
			})

			It("should error for invalid type", func() {
				resp, err := agent.GetPlaylists(ctx, 0, 25, "user-1", "abcde")
				Expect(resp).To(BeNil())
				Expect(err).To(MatchError(external_playlists.ErrorUnsupportedType))
			})
		})

		Describe("SyncPlaylist", func() {
			var pls model.Playlist

			BeforeEach(func() {
				pls = model.Playlist{
					ID:         "1000",
					OwnerID:    "user-1",
					ExternalId: "00000000-0000-0000-0000-000000000004",
					Tracks:     model.PlaylistTracks{},
				}
			})

			AfterEach(func() {
				_ = ds.Playlist(ctx).Delete(pls.ID)
			})

			Describe("Successful test", func() {
				WithMbid := model.MediaFile{ID: "1", Title: "Take Control", MbzTrackID: "9f42783a-423b-4ed6-8a10-fdf4cb44456f", Artist: "Old Gods of Asgard"}

				BeforeEach(func() {
					_ = ds.MediaFile(ctx).Put(&WithMbid)
				})

				AfterEach(func() {
					_ = ds.MediaFile(ctx).Delete(WithMbid.ID)
				})

				It("should return playlist", func() {
					f, _ := os.Open("tests/fixtures/listenbrainz.playlist.json")

					httpClient.Res = http.Response{
						Body:       f,
						StatusCode: 200,
					}

					err := agent.SyncPlaylist(ctx, ds, &pls)
					Expect(err).To(BeNil())
					Expect(pls.Tracks).To(HaveLen(1))
					Expect(pls.Tracks[0].ID).To(Equal("1"))
					Expect(pls.Comment).To(Equal("\n              This playlist contains the top tracks for example that were first listened to in 2022.\n              \n              \n                For more information on how this playlist is generated, please see our\n                Year in Music 2022 Playlists page.\n              \n           "))
				})
			})

			Describe("Failed tests", func() {
				It("should error when trying to fetch token and failing", func() {
					httpClient.Res = http.Response{
						Body:       io.NopCloser(bytes.NewBufferString(`{"Code":404,"Error":"No"}`)),
						StatusCode: 404,
					}
					_ = ds.UserProps(ctx).Delete("user-1", sessionKeyProperty+agents.UserSuffix)
					err := agent.SyncPlaylist(ctx, ds, &pls)
					Expect(err).To(Equal(model.ErrNotFound))
				})

				It("should error when playlist is not found", func() {
					httpClient.Res = http.Response{
						Body:       io.NopCloser(bytes.NewBufferString(`{"Code":404,"Error":"No"}`)),
						StatusCode: 404,
					}
					err := agent.SyncPlaylist(ctx, ds, &pls)
					Expect(err).To(Equal(model.ErrNotFound))
				})
			})
		})
	})
})
