package e2e

import (
	"net/http"

	"github.com/navidrome/navidrome/conf"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("stream.view (legacy streaming)", Ordered, func() {
	var (
		mp3TrackID  string // Come Together (mp3, 320kbps)
		flacTrackID string // TC FLAC Standard (flac, 900kbps)
	)

	BeforeAll(func() {
		setupTestDB()

		songs, err := ds.MediaFile(ctx).GetAll()
		Expect(err).ToNot(HaveOccurred())
		byTitle := map[string]string{}
		for _, s := range songs {
			byTitle[s.Title] = s.ID
		}
		mp3TrackID = byTitle["Come Together"]
		Expect(mp3TrackID).ToNot(BeEmpty())
		flacTrackID = byTitle["TC FLAC Standard"]
		Expect(flacTrackID).ToNot(BeEmpty())
	})

	Describe("raw / direct play", func() {
		It("streams raw when no format or maxBitRate is specified", func() {
			w := doRawReq("stream", "id", flacTrackID)
			Expect(w.Code).To(Equal(http.StatusOK))
			Expect(spy.LastRequest.Format).To(BeElementOf("raw", ""))
		})

		It("streams raw when format=raw is explicitly requested", func() {
			w := doRawReq("stream", "id", flacTrackID, "format", "raw")
			Expect(w.Code).To(Equal(http.StatusOK))
			Expect(spy.LastRequest.Format).To(BeElementOf("raw", ""))
		})

		It("streams raw when maxBitRate is >= source bitrate", func() {
			w := doRawReq("stream", "id", flacTrackID, "maxBitRate", "1000")
			Expect(w.Code).To(Equal(http.StatusOK))
			Expect(spy.LastRequest.Format).To(BeElementOf("raw", ""))
		})

		It("streams raw when format matches source and bitrate is not lower", func() {
			w := doRawReq("stream", "id", mp3TrackID, "format", "mp3", "maxBitRate", "320")
			Expect(w.Code).To(Equal(http.StatusOK))
			Expect(spy.LastRequest.Format).To(Equal("raw"))
		})
	})

	Describe("transcoding with explicit format", func() {
		It("transcodes to mp3 when format=mp3 is requested", func() {
			w := doRawReq("stream", "id", flacTrackID, "format", "mp3")
			Expect(w.Code).To(Equal(http.StatusOK))
			Expect(spy.LastRequest.Format).To(Equal("mp3"))
			// Should use the mp3 default bitrate (192kbps)
			Expect(spy.LastRequest.BitRate).To(Equal(192))
		})

		It("transcodes to opus when format=opus is requested (no maxBitRate)", func() {
			w := doRawReq("stream", "id", flacTrackID, "format", "opus")
			Expect(w.Code).To(Equal(http.StatusOK))
			Expect(spy.LastRequest.Format).To(Equal("opus"))
			// Should use the opus default bitrate (128kbps)
			Expect(spy.LastRequest.BitRate).To(Equal(128))
		})

		It("transcodes to opus with specified maxBitRate", func() {
			w := doRawReq("stream", "id", flacTrackID, "format", "opus", "maxBitRate", "192")
			Expect(w.Code).To(Equal(http.StatusOK))
			Expect(spy.LastRequest.Format).To(Equal("opus"))
			Expect(spy.LastRequest.BitRate).To(Equal(192))
		})

		It("transcodes to mp3 with specified maxBitRate", func() {
			w := doRawReq("stream", "id", flacTrackID, "format", "mp3", "maxBitRate", "128")
			Expect(w.Code).To(Equal(http.StatusOK))
			Expect(spy.LastRequest.Format).To(Equal("mp3"))
			Expect(spy.LastRequest.BitRate).To(Equal(128))
		})

		It("transcodes MP3 to opus when format=opus is requested", func() {
			w := doRawReq("stream", "id", mp3TrackID, "format", "opus")
			Expect(w.Code).To(Equal(http.StatusOK))
			Expect(spy.LastRequest.Format).To(Equal("opus"))
		})

		It("transcodes same format when maxBitRate is lower than source", func() {
			w := doRawReq("stream", "id", mp3TrackID, "format", "mp3", "maxBitRate", "128")
			Expect(w.Code).To(Equal(http.StatusOK))
			Expect(spy.LastRequest.Format).To(Equal("mp3"))
			Expect(spy.LastRequest.BitRate).To(Equal(128))
		})
	})

	Describe("downsampling with maxBitRate only", func() {
		It("transcodes using default downsampling format when maxBitRate < source bitrate", func() {
			conf.Server.DefaultDownsamplingFormat = "opus"
			w := doRawReq("stream", "id", flacTrackID, "maxBitRate", "192")
			Expect(w.Code).To(Equal(http.StatusOK))
			Expect(spy.LastRequest.Format).To(Equal("opus"))
			Expect(spy.LastRequest.BitRate).To(Equal(192))
		})

		It("streams raw when maxBitRate >= source bitrate (no downsampling needed)", func() {
			conf.Server.DefaultDownsamplingFormat = "opus"
			w := doRawReq("stream", "id", mp3TrackID, "maxBitRate", "320")
			Expect(w.Code).To(Equal(http.StatusOK))
			Expect(spy.LastRequest.Format).To(BeElementOf("raw", ""))
		})
	})

	Describe("timeOffset", func() {
		It("passes timeOffset to the stream request", func() {
			w := doRawReq("stream", "id", flacTrackID, "format", "mp3", "timeOffset", "30")
			Expect(w.Code).To(Equal(http.StatusOK))
			Expect(spy.LastRequest.Offset).To(Equal(30))
		})
	})
})
