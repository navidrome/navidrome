package listenbrainz

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"time"

	"github.com/navidrome/navidrome/consts"
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
})
