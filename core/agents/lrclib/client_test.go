package lrclib

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
	lrcError := `{"code": 404, "name": "TrackNotFound", "message": "Failed to find specified track"}`

	Describe("responses", func() {
		It("parses a successful response correctly", func() {
			var response lyricInfo
			simpleLyrics, _ := os.ReadFile("tests/fixtures/lrclib.get.success.json")
			err := json.Unmarshal(simpleLyrics, &response)
			Expect(err).ToNot(HaveOccurred())
			Expect(response.Id).To(Equal(3396226))
			Expect(response.Instrumental).To(Equal(false))
			Expect(response.PlainLyrics).To(Equal("I feel your breath upon my neck...\nThe clock won't stop and this is what we get\n"))
			Expect(response.SyncedLyrics).To(Equal("[00:17.12] I feel your breath upon my neck...\n[03:20.31] The clock won't stop and this is what we get\n"))
		})

		It("parses error successfully", func() {
			var response lrclibError
			err := json.Unmarshal([]byte(lrcError), &response)
			Expect(err).ToNot(HaveOccurred())
			Expect(response.Code).To(Equal(404))
			Expect(response.Message).To(Equal("Failed to find specified track"))
			Expect(response.Name).To(Equal("TrackNotFound"))
		})
	})

	Describe("client", func() {
		var httpClient *tests.FakeHttpClient
		var client *client

		BeforeEach(func() {
			httpClient = &tests.FakeHttpClient{}
			client = newClient(httpClient)
		})

		Describe("getLyrics", func() {
			It("should return lyrics properly", func() {
				f, _ := os.Open("tests/fixtures/lrclib.get.success.json")
				httpClient.Res = http.Response{Body: f, StatusCode: 200}

				lyrics, err := client.getLyrics(context.Background(), "trackName", "artistName", "albumName", 100)
				Expect(err).ToNot(HaveOccurred())
				Expect(lyrics).ToNot(BeNil())
				Expect(lyrics.Id).To(Equal(3396226))
				Expect(lyrics.Instrumental).To(Equal(false))
				Expect(lyrics.PlainLyrics).To(Equal("I feel your breath upon my neck...\nThe clock won't stop and this is what we get\n"))
				Expect(lyrics.SyncedLyrics).To(Equal("[00:17.12] I feel your breath upon my neck...\n[03:20.31] The clock won't stop and this is what we get\n"))

				Expect(httpClient.SavedRequest.URL.String()).To(Equal("https://lrclib.net/api/get?album_name=albumName&artist_name=artistName&duration=100&track_name=trackName"))
			})

			It("should parse instrumental possible", func() {
				f, _ := os.Open("tests/fixtures/lrclib.get.instrumental.json")
				httpClient.Res = http.Response{Body: f, StatusCode: 200}

				lyrics, err := client.getLyrics(context.Background(), "trackName", "artistName", "albumName", 100)
				Expect(err).ToNot(HaveOccurred())
				Expect(lyrics).ToNot(BeNil())
				Expect(lyrics.Id).To(Equal(3396226))
				Expect(lyrics.Instrumental).To(Equal(true))
				Expect(lyrics.PlainLyrics).To(Equal(""))
				Expect(lyrics.SyncedLyrics).To(Equal(""))

				Expect(httpClient.SavedRequest.URL.String()).To(Equal("https://lrclib.net/api/get?album_name=albumName&artist_name=artistName&duration=100&track_name=trackName"))
			})

			It("should handle error", func() {
				httpClient.Res = http.Response{
					Body:       io.NopCloser(bytes.NewBufferString(lrcError)),
					StatusCode: 404,
				}

				lyrics, err := client.getLyrics(context.Background(), "trackName", "artistName", "albumName", 100)
				Expect(lyrics).To(BeNil())
				Expect(err).To(Equal(lrclibError{
					Code:    404,
					Name:    "TrackNotFound",
					Message: "Failed to find specified track",
				}))
			})
		})
	})
})
