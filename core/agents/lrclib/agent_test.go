package lrclib

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"os"

	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/tests"
	. "github.com/navidrome/navidrome/utils/gg"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("lrclibAgent", func() {
	var ctx context.Context
	var agent *lrclibAgent
	var httpClient *tests.FakeHttpClient

	BeforeEach(func() {
		ctx = context.Background()
		httpClient = &tests.FakeHttpClient{}
		agent = lrclibConstructor()
		agent.client = newClient(httpClient)
	})

	Describe("getLyrics", func() {
		It("handles parses lyrics successfully", func() {
			f, _ := os.Open("tests/fixtures/lrclib.get.success.json")
			httpClient.Res = http.Response{Body: f, StatusCode: 200}

			mf := model.MediaFile{Album: "album", AlbumArtist: "artist", Title: "title", Duration: 233.5}
			lyrics, err := agent.GetSongLyrics(ctx, &mf)

			Expect(httpClient.SavedRequest.URL.String()).To(Equal("https://lrclib.net/api/get?album_name=album&artist_name=artist&duration=233&track_name=title"))

			Expect(err).ToNot(HaveOccurred())
			Expect(lyrics).To(Equal(model.LyricList{
				{
					DisplayArtist: "",
					DisplayTitle:  "",
					Lang:          "xxx",
					Line: []model.Line{
						{
							Start: P(int64(17120)),
							Value: "I feel your breath upon my neck...",
						},
						{
							Start: P(int64(200310)),
							Value: "The clock won't stop and this is what we get",
						},
					},
					Offset: nil,
					Synced: true,
				},
				{
					DisplayArtist: "",
					DisplayTitle:  "",
					Lang:          "xxx",
					Line: []model.Line{
						{
							Start: nil,
							Value: "I feel your breath upon my neck...",
						},
						{
							Start: nil,
							Value: "The clock won't stop and this is what we get",
						},
					},
					Offset: nil,
					Synced: false,
				},
			}))
		})

		It("returns nil for instrumental tracks", func() {
			f, _ := os.Open("tests/fixtures/lrclib.get.instrumental.json")
			httpClient.Res = http.Response{Body: f, StatusCode: 200}

			mf := model.MediaFile{Album: "album", AlbumArtist: "artist", Title: "title"}
			lyrics, err := agent.GetSongLyrics(ctx, &mf)

			Expect(err).ToNot(HaveOccurred())
			Expect(lyrics).To(BeNil())
		})

		It("handles error correctly", func() {
			httpClient.Res = http.Response{
				Body:       io.NopCloser(bytes.NewBufferString(`{"code": 404, "name": "TrackNotFound", "message": "Failed to find specified track"}`)),
				StatusCode: 404,
			}

			mf := model.MediaFile{Album: "album", AlbumArtist: "artist", Title: "title"}
			lyrics, err := agent.GetSongLyrics(ctx, &mf)

			Expect(lyrics).To(BeNil())
			Expect(err).To(Equal(lrclibError{
				Code:    404,
				Name:    "TrackNotFound",
				Message: "Failed to find specified track",
			}))
		})
	})
})
