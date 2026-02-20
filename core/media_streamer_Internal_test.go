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
				format, _ := selectTranscodingOptions(ctx, ds, mf, "raw", 0, 0)
				Expect(format).To(Equal("raw"))
			})
			It("returns raw if a transcoder does not exists", func() {
				mf.Suffix = "flac"
				mf.BitRate = 1000
				format, _ := selectTranscodingOptions(ctx, ds, mf, "m4a", 0, 0)
				Expect(format).To(Equal("raw"))
			})
			It("returns the requested format if a transcoder exists", func() {
				mf.Suffix = "flac"
				mf.BitRate = 1000
				format, bitRate := selectTranscodingOptions(ctx, ds, mf, "mp3", 0, 0)
				Expect(format).To(Equal("mp3"))
				Expect(bitRate).To(Equal(160)) // Default Bit Rate
			})
			It("returns raw if requested format is the same as the original and it is not necessary to downsample", func() {
				mf.Suffix = "mp3"
				mf.BitRate = 112
				format, _ := selectTranscodingOptions(ctx, ds, mf, "mp3", 128, 0)
				Expect(format).To(Equal("raw"))
			})
			It("returns the requested format if requested BitRate is lower than original", func() {
				mf.Suffix = "mp3"
				mf.BitRate = 320
				format, bitRate := selectTranscodingOptions(ctx, ds, mf, "mp3", 192, 0)
				Expect(format).To(Equal("mp3"))
				Expect(bitRate).To(Equal(192))
			})
			It("returns raw if requested format is the same as the original, but requested BitRate is 0", func() {
				mf.Suffix = "mp3"
				mf.BitRate = 320
				format, bitRate := selectTranscodingOptions(ctx, ds, mf, "mp3", 0, 0)
				Expect(format).To(Equal("raw"))
				Expect(bitRate).To(Equal(320))
			})
			It("returns the format when same format is requested but with a lower sample rate", func() {
				mf.Suffix = "flac"
				mf.BitRate = 2118
				mf.SampleRate = 96000
				format, bitRate := selectTranscodingOptions(ctx, ds, mf, "flac", 0, 48000)
				Expect(format).To(Equal("flac"))
				Expect(bitRate).To(Equal(0))
			})
			It("returns raw when same format is requested with same sample rate", func() {
				mf.Suffix = "flac"
				mf.BitRate = 1000
				mf.SampleRate = 48000
				format, _ := selectTranscodingOptions(ctx, ds, mf, "flac", 0, 48000)
				Expect(format).To(Equal("raw"))
			})
			It("returns raw when same format is requested with no sample rate constraint", func() {
				mf.Suffix = "flac"
				mf.BitRate = 1000
				mf.SampleRate = 96000
				format, _ := selectTranscodingOptions(ctx, ds, mf, "flac", 0, 0)
				Expect(format).To(Equal("raw"))
			})
			Context("Downsampling", func() {
				BeforeEach(func() {
					conf.Server.DefaultDownsamplingFormat = "opus"
					mf.Suffix = "FLAC"
					mf.BitRate = 960
				})
				It("returns the DefaultDownsamplingFormat if a maxBitrate is requested but not the format", func() {
					format, bitRate := selectTranscodingOptions(ctx, ds, mf, "", 128, 0)
					Expect(format).To(Equal("opus"))
					Expect(bitRate).To(Equal(128))
				})
				It("returns raw if maxBitrate is equal or greater than original", func() {
					// This happens with DSub (and maybe other clients?). See https://github.com/navidrome/navidrome/issues/2066
					format, bitRate := selectTranscodingOptions(ctx, ds, mf, "", 960, 0)
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
				format, _ := selectTranscodingOptions(ctx, ds, mf, "raw", 0, 0)
				Expect(format).To(Equal("raw"))
			})
			It("returns configured format/bitrate as default", func() {
				mf.Suffix = "flac"
				mf.BitRate = 1000
				format, bitRate := selectTranscodingOptions(ctx, ds, mf, "", 0, 0)
				Expect(format).To(Equal("oga"))
				Expect(bitRate).To(Equal(96))
			})
			It("returns requested format", func() {
				mf.Suffix = "flac"
				mf.BitRate = 1000
				format, bitRate := selectTranscodingOptions(ctx, ds, mf, "mp3", 0, 0)
				Expect(format).To(Equal("mp3"))
				Expect(bitRate).To(Equal(160)) // Default Bit Rate
			})
			It("returns requested bitrate", func() {
				mf.Suffix = "flac"
				mf.BitRate = 1000
				format, bitRate := selectTranscodingOptions(ctx, ds, mf, "", 80, 0)
				Expect(format).To(Equal("oga"))
				Expect(bitRate).To(Equal(80))
			})
			It("returns raw if selected bitrate and format is the same as original", func() {
				mf.Suffix = "mp3"
				mf.BitRate = 192
				format, bitRate := selectTranscodingOptions(ctx, ds, mf, "mp3", 192, 0)
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
				format, _ := selectTranscodingOptions(ctx, ds, mf, "raw", 0, 0)
				Expect(format).To(Equal("raw"))
			})
			It("returns configured format/bitrate as default", func() {
				mf.Suffix = "flac"
				mf.BitRate = 1000
				format, bitRate := selectTranscodingOptions(ctx, ds, mf, "", 0, 0)
				Expect(format).To(Equal("oga"))
				Expect(bitRate).To(Equal(192))
			})
			It("returns requested format", func() {
				mf.Suffix = "flac"
				mf.BitRate = 1000
				format, bitRate := selectTranscodingOptions(ctx, ds, mf, "mp3", 0, 0)
				Expect(format).To(Equal("mp3"))
				Expect(bitRate).To(Equal(160)) // Default Bit Rate
			})
			It("returns requested bitrate", func() {
				mf.Suffix = "flac"
				mf.BitRate = 1000
				format, bitRate := selectTranscodingOptions(ctx, ds, mf, "", 160, 0)
				Expect(format).To(Equal("oga"))
				Expect(bitRate).To(Equal(160))
			})
		})
	})
})
