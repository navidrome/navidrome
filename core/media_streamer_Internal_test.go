package core

import (
	"context"

	"github.com/navidrome/navidrome/conf"
	"github.com/navidrome/navidrome/log"
	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/model/request"
	"github.com/navidrome/navidrome/tests"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("MediaStreamer", func() {
	var ds model.DataStore
	ctx := log.NewContext(context.Background())

	BeforeEach(func() {
		ds = &tests.MockDataStore{MockedTranscoding: &tests.MockTranscodingRepo{}}
	})

	Context("selectTranscodingOptions", func() {
		mf := &model.MediaFile{}
		Context("player is not configured", func() {
			It("returns raw if raw is requested", func() {
				mf.Suffix = "flac"
				mf.BitRate = 1000
				format, _ := selectTranscodingOptions(ctx, ds, mf, "raw", 0)
				Expect(format).To(Equal("raw"))
			})
			It("returns raw if a transcoder does not exists", func() {
				mf.Suffix = "flac"
				mf.BitRate = 1000
				format, _ := selectTranscodingOptions(ctx, ds, mf, "m4a", 0)
				Expect(format).To(Equal("raw"))
			})
			It("returns the requested format if a transcoder exists", func() {
				mf.Suffix = "flac"
				mf.BitRate = 1000
				format, bitRate := selectTranscodingOptions(ctx, ds, mf, "mp3", 0)
				Expect(format).To(Equal("mp3"))
				Expect(bitRate).To(Equal(160)) // Default Bit Rate
			})
			It("returns raw if requested format is the same as the original and it is not necessary to downsample", func() {
				mf.Suffix = "mp3"
				mf.BitRate = 112
				format, _ := selectTranscodingOptions(ctx, ds, mf, "mp3", 128)
				Expect(format).To(Equal("raw"))
			})
			It("returns the requested format if requested BitRate is lower than original", func() {
				mf.Suffix = "mp3"
				mf.BitRate = 320
				format, bitRate := selectTranscodingOptions(ctx, ds, mf, "mp3", 192)
				Expect(format).To(Equal("mp3"))
				Expect(bitRate).To(Equal(192))
			})
			It("returns raw if requested format is the same as the original, but requested BitRate is 0", func() {
				mf.Suffix = "mp3"
				mf.BitRate = 320
				format, bitRate := selectTranscodingOptions(ctx, ds, mf, "mp3", 0)
				Expect(format).To(Equal("raw"))
				Expect(bitRate).To(Equal(320))
			})
			Context("Downsampling", func() {
				BeforeEach(func() {
					conf.Server.DefaultDownsamplingFormat = "opus"
					mf.Suffix = "FLAC"
					mf.BitRate = 960
				})
				It("returns the DefaultDownsamplingFormat if a maxBitrate is requested but not the format", func() {
					format, bitRate := selectTranscodingOptions(ctx, ds, mf, "", 128)
					Expect(format).To(Equal("opus"))
					Expect(bitRate).To(Equal(128))
				})
				It("returns raw if maxBitrate is equal or greater than original", func() {
					// This happens with DSub (and maybe other clients?). See https://github.com/navidrome/navidrome/issues/2066
					format, bitRate := selectTranscodingOptions(ctx, ds, mf, "", 960)
					Expect(format).To(Equal("raw"))
					Expect(bitRate).To(Equal(0))
				})
			})
		})

		Context("player has format configured", func() {
			BeforeEach(func() {
				t := model.Transcoding{ID: "oga1", TargetFormat: "oga", DefaultBitRate: 96}
				ctx = request.WithTranscoding(ctx, t)
			})
			It("returns raw if raw is requested", func() {
				mf.Suffix = "flac"
				mf.BitRate = 1000
				format, _ := selectTranscodingOptions(ctx, ds, mf, "raw", 0)
				Expect(format).To(Equal("raw"))
			})
			It("returns configured format/bitrate as default", func() {
				mf.Suffix = "flac"
				mf.BitRate = 1000
				format, bitRate := selectTranscodingOptions(ctx, ds, mf, "", 0)
				Expect(format).To(Equal("oga"))
				Expect(bitRate).To(Equal(96))
			})
			It("returns requested format", func() {
				mf.Suffix = "flac"
				mf.BitRate = 1000
				format, bitRate := selectTranscodingOptions(ctx, ds, mf, "mp3", 0)
				Expect(format).To(Equal("mp3"))
				Expect(bitRate).To(Equal(160)) // Default Bit Rate
			})
			It("returns requested bitrate", func() {
				mf.Suffix = "flac"
				mf.BitRate = 1000
				format, bitRate := selectTranscodingOptions(ctx, ds, mf, "", 80)
				Expect(format).To(Equal("oga"))
				Expect(bitRate).To(Equal(80))
			})
			It("returns raw if selected bitrate and format is the same as original", func() {
				mf.Suffix = "mp3"
				mf.BitRate = 192
				format, bitRate := selectTranscodingOptions(ctx, ds, mf, "mp3", 192)
				Expect(format).To(Equal("raw"))
				Expect(bitRate).To(Equal(0))
			})
		})

		Context("player has maxBitRate configured", func() {
			BeforeEach(func() {
				t := model.Transcoding{ID: "oga1", TargetFormat: "oga", DefaultBitRate: 96}
				p := model.Player{ID: "player1", TranscodingId: t.ID, MaxBitRate: 192}
				ctx = request.WithTranscoding(ctx, t)
				ctx = request.WithPlayer(ctx, p)
			})
			It("returns raw if raw is requested", func() {
				mf.Suffix = "flac"
				mf.BitRate = 1000
				format, _ := selectTranscodingOptions(ctx, ds, mf, "raw", 0)
				Expect(format).To(Equal("raw"))
			})
			It("returns configured format/bitrate as default", func() {
				mf.Suffix = "flac"
				mf.BitRate = 1000
				format, bitRate := selectTranscodingOptions(ctx, ds, mf, "", 0)
				Expect(format).To(Equal("oga"))
				Expect(bitRate).To(Equal(192))
			})
			It("returns requested format", func() {
				mf.Suffix = "flac"
				mf.BitRate = 1000
				format, bitRate := selectTranscodingOptions(ctx, ds, mf, "mp3", 0)
				Expect(format).To(Equal("mp3"))
				Expect(bitRate).To(Equal(160)) // Default Bit Rate
			})
			It("returns requested bitrate", func() {
				mf.Suffix = "flac"
				mf.BitRate = 1000
				format, bitRate := selectTranscodingOptions(ctx, ds, mf, "", 160)
				Expect(format).To(Equal("oga"))
				Expect(bitRate).To(Equal(160))
			})
		})
	})
})
