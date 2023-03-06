package webhook

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/navidrome/navidrome/core/scrobbler"
	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/tests"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("agent", func() {
	var ds model.DataStore
	var ctx context.Context
	BeforeEach(func() {
		ds = &tests.MockDataStore{}
		ctx = context.Background()
	})

	Describe("webhookConstructor", func() {
		It("Creates an agent", func() {
			agent := webhookConstructor(ds, "Service", "BASE_URL", "API_KEY")

			Expect(agent.apiKey).To(Equal("API_KEY"))
			Expect(agent.name).To(Equal("Service"))
			Expect(agent.url).To(Equal("BASE_URL"))
		})
	})

	Describe("scrobbling", func() {
		var agent *webhookAgent
		var httpClient *tests.FakeHttpClient
		var track *model.MediaFile

		BeforeEach(func() {
			_ = ds.UserProps(ctx).Put("user-1", sessionBaseKeyProperty+"Service", "token")
			httpClient = &tests.FakeHttpClient{}
			client := newClient("BASE_URL", "API_KEY", httpClient)
			agent = webhookConstructor(ds, "Service", "BASE_URL", "API_KEY")
			agent.client = client
			track = &model.MediaFile{
				ID:          "123",
				Title:       "Track Title",
				Album:       "Track Album",
				Artist:      "Track Artist",
				AlbumArtist: "Album Artist",
				TrackNumber: 5,
				Duration:    410,
				MbzTrackID:  "mbz-356",
			}
		})

		testParams := func(isSubmission bool) {
			b := strconv.FormatBool(isSubmission)
			It("calls scrobble with submission = "+b, func() {
				httpClient.Res = http.Response{Body: io.NopCloser(bytes.NewBufferString("{}")), StatusCode: 200}

				ts := time.Unix(0, 0)

				var err error
				if isSubmission {
					err = agent.Scrobble(ctx, "user-1", scrobbler.Scrobble{MediaFile: *track, TimeStamp: ts})
				} else {
					err = agent.NowPlaying(ctx, "user-1", track)
				}

				Expect(err).ToNot(HaveOccurred())
				Expect(httpClient.SavedRequest.Method).To(Equal(http.MethodPost))
				Expect(httpClient.SavedRequest.URL.String()).To(Equal("BASE_URL/scrobble"))
				Expect(httpClient.SavedRequest.Header.Get("Content-Type")).To(Equal("application/json; charset=UTF-8"))

				body, _ := io.ReadAll(httpClient.SavedRequest.Body)
				f, _ := os.ReadFile("tests/fixtures/webhook.scrobble." + b + ".request.json")
				Expect(body).To(MatchJSON(f))
			})
		}

		Describe("NowPlaying", func() {
			testParams(false)

			It("returns ErrNotAuthorized if user is not linked", func() {
				err := agent.NowPlaying(ctx, "user-2", track)
				Expect(err).To(MatchError(scrobbler.ErrNotAuthorized))
			})
		})

		Describe("Scrobble", func() {
			testParams(true)

			It("returns ErrNotAuthorized if user is not linked", func() {
				err := agent.Scrobble(ctx, "user-2", scrobbler.Scrobble{MediaFile: *track, TimeStamp: time.Now()})
				Expect(err).To(MatchError(scrobbler.ErrNotAuthorized))
			})

			It("skips songs with less than 31 seconds", func() {
				track.Duration = 29
				httpClient.Res = http.Response{Body: io.NopCloser(bytes.NewBufferString("{}")), StatusCode: 200}

				err := agent.Scrobble(ctx, "user-1", scrobbler.Scrobble{MediaFile: *track, TimeStamp: time.Now()})

				Expect(err).ToNot(HaveOccurred())
				Expect(httpClient.SavedRequest).To(BeNil())
			})
		})
	})
})
