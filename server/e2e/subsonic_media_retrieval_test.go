package e2e

import (
	"net/http"

	"github.com/navidrome/navidrome/conf"
	"github.com/navidrome/navidrome/conf/configtest"
	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/server/subsonic/responses"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Media Retrieval Endpoints", Ordered, func() {
	BeforeAll(func() {
		setupTestDB()
	})

	Describe("Stream", func() {
		var trackID string

		BeforeAll(func() {
			// All test tracks are mp3 at 320kbps
			songs, err := ds.MediaFile(ctx).GetAll(model.QueryOptions{Max: 1, Sort: "title"})
			Expect(err).ToNot(HaveOccurred())
			Expect(songs).ToNot(BeEmpty())
			trackID = songs[0].ID
		})

		It("returns error when id parameter is missing", func() {
			resp := doReq("stream")

			Expect(resp.Status).To(Equal(responses.StatusFailed))
			Expect(resp.Error).ToNot(BeNil())
		})

		It("streams raw when no format or bitrate specified", func() {
			w := doRawReq("stream", "id", trackID)

			Expect(w.Code).To(Equal(http.StatusOK))
			Expect(spy.LastRequest.Format).To(Equal("raw"))
		})

		It("streams raw when format=raw", func() {
			w := doRawReq("stream", "id", trackID, "format", "raw")

			Expect(w.Code).To(Equal(http.StatusOK))
			Expect(spy.LastRequest.Format).To(Equal("raw"))
		})

		It("transcodes to different format with bitrate", func() {
			w := doRawReq("stream", "id", trackID, "format", "opus", "maxBitRate", "128")

			Expect(w.Code).To(Equal(http.StatusOK))
			Expect(spy.LastRequest.Format).To(Equal("opus"))
			Expect(spy.LastRequest.BitRate).To(Equal(128))
		})

		It("downsamples when only maxBitRate is specified (lower than source)", func() {
			DeferCleanup(configtest.SetupConfig())
			conf.Server.DefaultDownsamplingFormat = "opus"

			w := doRawReq("stream", "id", trackID, "maxBitRate", "128")

			Expect(w.Code).To(Equal(http.StatusOK))
			Expect(spy.LastRequest.Format).To(Equal("opus"))
			Expect(spy.LastRequest.BitRate).To(Equal(128))
		})

		It("streams raw when maxBitRate is higher than source", func() {
			w := doRawReq("stream", "id", trackID, "maxBitRate", "999")

			Expect(w.Code).To(Equal(http.StatusOK))
			Expect(spy.LastRequest.Format).To(Equal("raw"))
		})

		It("streams raw when format matches source and no bitrate reduction", func() {
			w := doRawReq("stream", "id", trackID, "format", "mp3", "maxBitRate", "320")

			Expect(w.Code).To(Equal(http.StatusOK))
			Expect(spy.LastRequest.Format).To(Equal("raw"))
		})

		It("transcodes when same format but lower bitrate", func() {
			w := doRawReq("stream", "id", trackID, "format", "mp3", "maxBitRate", "128")

			Expect(w.Code).To(Equal(http.StatusOK))
			Expect(spy.LastRequest.Format).To(Equal("mp3"))
			Expect(spy.LastRequest.BitRate).To(Equal(128))
		})

		It("falls back to raw for unknown format", func() {
			w := doRawReq("stream", "id", trackID, "format", "xyz")

			Expect(w.Code).To(Equal(http.StatusOK))
			Expect(spy.LastRequest.Format).To(Equal("raw"))
		})

		It("passes timeOffset through", func() {
			w := doRawReq("stream", "id", trackID, "format", "opus", "maxBitRate", "128", "timeOffset", "30")

			Expect(w.Code).To(Equal(http.StatusOK))
			Expect(spy.LastRequest.Format).To(Equal("opus"))
			Expect(spy.LastRequest.Offset).To(Equal(30))
		})
	})

	Describe("Download", func() {
		var trackID string

		BeforeAll(func() {
			// All test tracks are mp3 at 320kbps
			songs, err := ds.MediaFile(ctx).GetAll(model.QueryOptions{Max: 1, Sort: "title"})
			Expect(err).ToNot(HaveOccurred())
			Expect(songs).ToNot(BeEmpty())
			trackID = songs[0].ID
		})

		It("returns error when id parameter is missing", func() {
			resp := doReq("download")

			Expect(resp.Status).To(Equal(responses.StatusFailed))
			Expect(resp.Error).ToNot(BeNil())
		})

		It("downloads raw when no format specified and AutoTranscodeDownload is false", func() {
			DeferCleanup(configtest.SetupConfig())
			conf.Server.EnableDownloads = true
			conf.Server.AutoTranscodeDownload = false

			w := doRawReq("download", "id", trackID)

			Expect(w.Code).To(Equal(http.StatusOK))
			Expect(spy.LastRequest.Format).To(Equal("raw"))
		})

		It("downloads with explicit format and bitrate", func() {
			DeferCleanup(configtest.SetupConfig())
			conf.Server.EnableDownloads = true

			w := doRawReq("download", "id", trackID, "format", "opus", "bitrate", "128")

			Expect(w.Code).To(Equal(http.StatusOK))
			Expect(spy.LastRequest.Format).To(Equal("opus"))
			Expect(spy.LastRequest.BitRate).To(Equal(128))
		})

		It("returns error when downloads are disabled", func() {
			DeferCleanup(configtest.SetupConfig())
			conf.Server.EnableDownloads = false

			resp := doReq("download", "id", trackID)

			Expect(resp.Status).To(Equal(responses.StatusFailed))
		})
	})

	Describe("GetCoverArt", func() {
		It("handles request without error", func() {
			w := doRawReq("getCoverArt")

			Expect(w.Code).To(Equal(http.StatusOK))
		})
	})

	Describe("GetAvatar", func() {
		It("returns placeholder avatar when gravatar disabled", func() {
			w := doRawReq("getAvatar", "username", "admin")

			Expect(w.Code).To(Equal(http.StatusOK))
		})
	})

	Describe("GetLyrics", func() {
		It("returns empty lyrics when no match found", func() {
			resp := doReq("getLyrics", "artist", "NonExistentArtist", "title", "NonExistentTitle")

			Expect(resp.Status).To(Equal(responses.StatusOK))
			Expect(resp.Lyrics).ToNot(BeNil())
			Expect(resp.Lyrics.Value).To(BeEmpty())
		})
	})

	Describe("GetLyricsBySongId", func() {
		It("returns error when id parameter is missing", func() {
			resp := doReq("getLyricsBySongId")

			Expect(resp.Status).To(Equal(responses.StatusFailed))
			Expect(resp.Error).ToNot(BeNil())
		})

		It("returns error for non-existent song id", func() {
			resp := doReq("getLyricsBySongId", "id", "non-existent-id")

			Expect(resp.Status).To(Equal(responses.StatusFailed))
			Expect(resp.Error).ToNot(BeNil())
		})
	})
})
