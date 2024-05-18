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
						TrackNumber:    1,
						RecordingMbzID: "mbz-123",
						ArtistMbzIDs:   []string{"mbz-789"},
						ReleaseMbID:    "mbz-456",
						DurationMs:     142200,
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
})
