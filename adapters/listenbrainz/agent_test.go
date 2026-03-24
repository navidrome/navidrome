package listenbrainz

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"os"
	"time"

	"github.com/navidrome/navidrome/consts"
	"github.com/navidrome/navidrome/core/agents"
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
		httpClient = &tests.FakeHttpClient{}
		agent = listenBrainzConstructor(ds)
		agent.client = newClient("http://localhost:8080", httpClient)
		track = &model.MediaFile{
			ID:                "123",
			Title:             "Track Title",
			Album:             "Track Album",
			Artist:            "Track Artist",
			TrackNumber:       1,
			MbzRecordingID:    "mbz-123",
			MbzAlbumID:        "mbz-456",
			MbzReleaseGroupID: "mbz-789",
			Duration:          142.2,
			Participants: map[model.Role]model.ParticipantList{
				model.RoleArtist: []model.Participant{
					{Artist: model.Artist{ID: "ar-1", Name: "Artist 1", MbzArtistID: "mbz-111"}},
					{Artist: model.Artist{ID: "ar-2", Name: "Artist 2", MbzArtistID: "mbz-222"}},
				},
			},
		}
	})

	Describe("formatListen", func() {
		It("constructs the listenInfo properly", func() {
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
						"RecordingMBID":           Equal(track.MbzRecordingID),
						"ReleaseMBID":             Equal(track.MbzAlbumID),
						"ReleaseGroupMBID":        Equal(track.MbzReleaseGroupID),
						"ArtistNames":             ConsistOf("Artist 1", "Artist 2"),
						"ArtistMBIDs":             ConsistOf("mbz-111", "mbz-222"),
						"DurationMs":              Equal(142200),
					}),
				}),
			}))
		})
	})

	Describe("NowPlaying", func() {
		It("updates NowPlaying successfully", func() {
			httpClient.Res = http.Response{Body: io.NopCloser(bytes.NewBufferString(`{"status": "ok"}`)), StatusCode: 200}

			err := agent.NowPlaying(ctx, "user-1", track, 0)
			Expect(err).ToNot(HaveOccurred())
		})

		It("returns ErrNotAuthorized if user is not linked", func() {
			err := agent.NowPlaying(ctx, "user-2", track, 0)
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

	Describe("GetArtistUrl", func() {
		var agent *listenBrainzAgent
		var httpClient *tests.FakeHttpClient
		BeforeEach(func() {
			httpClient = &tests.FakeHttpClient{}
			client := newClient("BASE_URL", httpClient)
			agent = listenBrainzConstructor(ds)
			agent.client = client
		})

		It("returns artist url when MBID present", func() {
			f, _ := os.Open("tests/fixtures/listenbrainz.artist.metadata.homepage.json")
			httpClient.Res = http.Response{Body: f, StatusCode: 200}
			Expect(agent.GetArtistURL(ctx, "", "", "d2a92ee2-27ce-4e71-bfc5-12e34fe8ef56")).To(Equal("http://projectmili.com/"))
			Expect(httpClient.RequestCount).To(Equal(1))
			Expect(httpClient.SavedRequest.URL.Query().Get("artist_mbids")).To(Equal("d2a92ee2-27ce-4e71-bfc5-12e34fe8ef56"))
		})

		It("returns error when url not present", func() {
			f, _ := os.Open("tests/fixtures/listenbrainz.artist.metadata.no_homepage.json")
			httpClient.Res = http.Response{Body: f, StatusCode: 200}
			_, err := agent.GetArtistURL(ctx, "", "", "7c2cc610-f998-43ef-a08f-dae3344b8973")
			Expect(err).To(HaveOccurred())
			Expect(httpClient.RequestCount).To(Equal(1))
			Expect(httpClient.SavedRequest.URL.Query().Get("artist_mbids")).To(Equal("7c2cc610-f998-43ef-a08f-dae3344b8973"))
		})

		It("returns error when fetch calls fails", func() {
			httpClient.Err = errors.New("error")
			_, err := agent.GetArtistURL(ctx, "", "", "7c2cc610-f998-43ef-a08f-dae3344b8973")
			Expect(err).To(HaveOccurred())
			Expect(httpClient.RequestCount).To(Equal(1))
			Expect(httpClient.SavedRequest.URL.Query().Get("artist_mbids")).To(Equal("7c2cc610-f998-43ef-a08f-dae3344b8973"))
		})

		It("returns error when ListenBrainz returns an error", func() {
			httpClient.Res = http.Response{
				Body:       io.NopCloser(bytes.NewBufferString(`{"code": 400,"error": "artist mbid 1 is not valid."}`)),
				StatusCode: 400,
			}
			_, err := agent.GetArtistURL(ctx, "", "", "7c2cc610-f998-43ef-a08f-dae3344b8973")
			Expect(err).To(HaveOccurred())
			Expect(httpClient.RequestCount).To(Equal(1))
			Expect(httpClient.SavedRequest.URL.Query().Get("artist_mbids")).To(Equal("7c2cc610-f998-43ef-a08f-dae3344b8973"))
		})
	})

	Describe("GetTopSongs", func() {
		var agent *listenBrainzAgent
		var httpClient *tests.FakeHttpClient
		BeforeEach(func() {
			httpClient = &tests.FakeHttpClient{}
			client := newClient("BASE_URL", httpClient)
			agent = listenBrainzConstructor(ds)
			agent.client = client
		})

		It("returns error when fetch calls", func() {
			httpClient.Err = errors.New("error")
			_, err := agent.GetArtistTopSongs(ctx, "", "", "d2a92ee2-27ce-4e71-bfc5-12e34fe8ef56", 1)
			Expect(err).To(HaveOccurred())
			Expect(httpClient.RequestCount).To(Equal(1))
			Expect(httpClient.SavedRequest.URL.Path).To(Equal("/1/popularity/top-recordings-for-artist/d2a92ee2-27ce-4e71-bfc5-12e34fe8ef56"))
		})

		It("returns an error on listenbrainz error", func() {
			httpClient.Res = http.Response{
				Body:       io.NopCloser(bytes.NewBufferString(`{"code":400,"error":"artist_mbid: '1' is not a valid uuid"}`)),
				StatusCode: 400,
			}
			_, err := agent.GetArtistTopSongs(ctx, "", "", "1", 1)
			Expect(err).To(HaveOccurred())
			Expect(httpClient.RequestCount).To(Equal(1))
			Expect(httpClient.SavedRequest.URL.Path).To(Equal("/1/popularity/top-recordings-for-artist/1"))
		})

		It("returns all tracks when asked", func() {
			f, _ := os.Open("tests/fixtures/listenbrainz.popularity.json")
			httpClient.Res = http.Response{Body: f, StatusCode: 200}
			data, err := agent.GetArtistTopSongs(ctx, "", "", "d2a92ee2-27ce-4e71-bfc5-12e34fe8ef56", 2)
			Expect(err).ToNot(HaveOccurred())
			Expect(data).To(Equal([]agents.Song{
				{
					ID:         "",
					Name:       "world.execute(me);",
					MBID:       "9980309d-3480-4e7e-89ce-fce971a452be",
					Artist:     "Mili",
					ArtistMBID: "d2a92ee2-27ce-4e71-bfc5-12e34fe8ef56",
					Album:      "Miracle Milk",
					AlbumMBID:  "38a8f6e1-0e34-4418-a89d-78240a367408",
					Duration:   211912,
				},
				{
					ID:         "",
					Name:       "String Theocracy",
					MBID:       "afa2c83d-b17f-4029-b9da-790ea9250cf9",
					Artist:     "Mili",
					ArtistMBID: "d2a92ee2-27ce-4e71-bfc5-12e34fe8ef56",
					Album:      "String Theocracy",
					AlbumMBID:  "d79a38e3-7016-4f39-a31a-f495ce914b8e",
					Duration:   174000,
				},
			}))
		})

		It("returns only one track when prompted", func() {
			f, _ := os.Open("tests/fixtures/listenbrainz.popularity.json")
			httpClient.Res = http.Response{Body: f, StatusCode: 200}
			data, err := agent.GetArtistTopSongs(ctx, "", "", "d2a92ee2-27ce-4e71-bfc5-12e34fe8ef56", 1)
			Expect(err).ToNot(HaveOccurred())
			Expect(data).To(Equal([]agents.Song{
				{
					ID:         "",
					Name:       "world.execute(me);",
					MBID:       "9980309d-3480-4e7e-89ce-fce971a452be",
					Artist:     "Mili",
					ArtistMBID: "d2a92ee2-27ce-4e71-bfc5-12e34fe8ef56",
					Album:      "Miracle Milk",
					AlbumMBID:  "38a8f6e1-0e34-4418-a89d-78240a367408",
					Duration:   211912,
				},
			}))
		})
	})

	Describe("GetSimilarArtists", func() {
		var agent *listenBrainzAgent
		var httpClient *tests.FakeHttpClient
		baseUrl := "https://labs.api.listenbrainz.org/similar-artists/json?algorithm=session_based_days_9000_session_300_contribution_5_threshold_15_limit_50_skip_30&artist_mbids="
		mbid := "db92a151-1ac2-438b-bc43-b82e149ddd50"

		BeforeEach(func() {
			httpClient = &tests.FakeHttpClient{}
			client := newClient("BASE_URL", httpClient)
			agent = listenBrainzConstructor(ds)
			agent.client = client
		})

		It("returns error when fetch calls", func() {
			httpClient.Err = errors.New("error")
			_, err := agent.GetSimilarArtists(ctx, "", "", mbid, 1)
			Expect(err).To(HaveOccurred())
			Expect(httpClient.RequestCount).To(Equal(1))
			Expect(httpClient.SavedRequest.URL.String()).To(Equal(baseUrl + mbid))
		})

		It("returns an error on listenbrainz error", func() {
			httpClient.Res = http.Response{
				Body:       io.NopCloser(bytes.NewBufferString(`Bad request`)),
				StatusCode: 400,
			}
			_, err := agent.GetSimilarArtists(ctx, "", "", "1", 1)
			Expect(err).To(HaveOccurred())
			Expect(httpClient.RequestCount).To(Equal(1))
			Expect(httpClient.SavedRequest.URL.String()).To(Equal(baseUrl + "1"))
		})

		It("returns all data on call", func() {
			f, _ := os.Open("tests/fixtures/listenbrainz.labs.similar-artists.json")
			httpClient.Res = http.Response{Body: f, StatusCode: 200}

			resp, err := agent.GetSimilarArtists(ctx, "", "", "db92a151-1ac2-438b-bc43-b82e149ddd50", 2)
			Expect(err).ToNot(HaveOccurred())
			Expect(httpClient.RequestCount).To(Equal(1))
			Expect(httpClient.SavedRequest.URL.String()).To(Equal(baseUrl + mbid))
			Expect(resp).To(Equal([]agents.Artist{
				{MBID: "f27ec8db-af05-4f36-916e-3d57f91ecf5e", Name: "Michael Jackson"},
				{MBID: "7364dea6-ca9a-48e3-be01-b44ad0d19897", Name: "a-ha"},
			}))
		})

		It("returns subset of data on call", func() {
			f, _ := os.Open("tests/fixtures/listenbrainz.labs.similar-artists.json")
			httpClient.Res = http.Response{Body: f, StatusCode: 200}

			resp, err := agent.GetSimilarArtists(ctx, "", "", "db92a151-1ac2-438b-bc43-b82e149ddd50", 1)
			Expect(err).ToNot(HaveOccurred())
			Expect(httpClient.RequestCount).To(Equal(1))
			Expect(httpClient.SavedRequest.URL.String()).To(Equal(baseUrl + mbid))
			Expect(resp).To(Equal([]agents.Artist{
				{MBID: "f27ec8db-af05-4f36-916e-3d57f91ecf5e", Name: "Michael Jackson"},
			}))
		})
	})

	Describe("GetSimilarTracks", func() {
		var agent *listenBrainzAgent
		var httpClient *tests.FakeHttpClient
		mbid := "8f3471b5-7e6a-48da-86a9-c1c07a0f47ae"
		baseUrl := "https://labs.api.listenbrainz.org/similar-recordings/json?algorithm=session_based_days_9000_session_300_contribution_5_threshold_15_limit_50_skip_30&recording_mbids="

		BeforeEach(func() {
			httpClient = &tests.FakeHttpClient{}
			client := newClient("BASE_URL", httpClient)
			agent = listenBrainzConstructor(ds)
			agent.client = client
		})

		It("returns error when fetch calls", func() {
			httpClient.Err = errors.New("error")
			_, err := agent.GetSimilarSongsByTrack(ctx, "", "", "", mbid, 1)
			Expect(err).To(HaveOccurred())
			Expect(httpClient.RequestCount).To(Equal(1))
			Expect(httpClient.SavedRequest.URL.String()).To(Equal(baseUrl + mbid))
		})

		It("returns an error on listenbrainz error", func() {
			httpClient.Res = http.Response{
				Body:       io.NopCloser(bytes.NewBufferString(`Bad request`)),
				StatusCode: 400,
			}
			_, err := agent.GetSimilarSongsByTrack(ctx, "", "", "", "1", 1)
			Expect(err).To(HaveOccurred())
			Expect(httpClient.RequestCount).To(Equal(1))
			Expect(httpClient.SavedRequest.URL.String()).To(Equal(baseUrl + "1"))
		})

		It("returns all data on call", func() {
			f, _ := os.Open("tests/fixtures/listenbrainz.labs.similar-recordings.json")
			httpClient.Res = http.Response{Body: f, StatusCode: 200}

			resp, err := agent.GetSimilarSongsByTrack(ctx, "", "", "", mbid, 2)
			Expect(err).ToNot(HaveOccurred())
			Expect(httpClient.RequestCount).To(Equal(1))
			Expect(httpClient.SavedRequest.URL.String()).To(Equal(baseUrl + mbid))
			Expect(resp).To(Equal([]agents.Song{
				{
					ID:         "",
					Name:       "Take On Me",
					MBID:       "12f65dca-de8f-43fe-a65d-f12a02aaadf3",
					ISRC:       "",
					Artist:     "a‐ha",
					ArtistMBID: "",
					Album:      "Hunting High and Low",
					AlbumMBID:  "4ec07fe8-e7c6-3106-a0aa-fdf92f13f7fc",
					Duration:   0,
				},
				{
					ID:         "",
					Name:       "Wake Me Up Before You Go‐Go",
					MBID:       "80033c72-aa19-4ba8-9227-afb075fec46e",
					ISRC:       "",
					Artist:     "Wham!",
					ArtistMBID: "",
					Album:      "Make It Big",
					AlbumMBID:  "c143d542-48dc-446b-b523-1762da721638",
					Duration:   0,
				},
			}))
		})

		It("returns subset of data on call", func() {
			f, _ := os.Open("tests/fixtures/listenbrainz.labs.similar-recordings.json")
			httpClient.Res = http.Response{Body: f, StatusCode: 200}

			resp, err := agent.GetSimilarSongsByTrack(ctx, "", "", "", mbid, 1)
			Expect(err).ToNot(HaveOccurred())
			Expect(httpClient.RequestCount).To(Equal(1))
			Expect(httpClient.SavedRequest.URL.String()).To(Equal(baseUrl + mbid))
			Expect(resp).To(Equal([]agents.Song{
				{
					ID:         "",
					Name:       "Take On Me",
					MBID:       "12f65dca-de8f-43fe-a65d-f12a02aaadf3",
					ISRC:       "",
					Artist:     "a‐ha",
					ArtistMBID: "",
					Album:      "Hunting High and Low",
					AlbumMBID:  "4ec07fe8-e7c6-3106-a0aa-fdf92f13f7fc",
					Duration:   0,
				},
			}))
		})
	})
})
