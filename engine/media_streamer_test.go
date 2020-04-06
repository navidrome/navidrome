package engine

import (
	"context"
	"io"
	"strings"

	"github.com/deluan/navidrome/log"
	"github.com/deluan/navidrome/model"
	"github.com/deluan/navidrome/persistence"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("MediaStreamer", func() {
	var streamer MediaStreamer
	var ds model.DataStore
	ffmpeg := &fakeFFmpeg{Data: "fake data"}
	ctx := log.NewContext(nil)

	BeforeEach(func() {
		ds = &persistence.MockDataStore{MockedTranscoding: &mockTranscodingRepository{}}
		ds.MediaFile(ctx).(*persistence.MockMediaFile).SetData(`[{"id": "123", "path": "tests/fixtures/test.mp3", "suffix": "mp3", "bitRate": 128, "duration": 257.0}]`, 1)
		streamer = NewMediaStreamer(ds, ffmpeg, testCache)
	})

	Context("NewStream", func() {
		It("returns a seekable stream if format is 'raw'", func() {
			s, err := streamer.NewStream(ctx, "123", "raw", 0)
			Expect(err).ToNot(HaveOccurred())
			Expect(s.Seekable()).To(BeTrue())
		})
		It("returns a seekable stream if maxBitRate is 0", func() {
			s, err := streamer.NewStream(ctx, "123", "mp3", 0)
			Expect(err).ToNot(HaveOccurred())
			Expect(s.Seekable()).To(BeTrue())
		})
		It("returns a seekable stream if maxBitRate is higher than file bitRate", func() {
			s, err := streamer.NewStream(ctx, "123", "mp3", 320)
			Expect(err).ToNot(HaveOccurred())
			Expect(s.Seekable()).To(BeTrue())
		})
		It("returns a NON seekable stream if transcode is required", func() {
			s, err := streamer.NewStream(ctx, "123", "mp3", 64)
			Expect(err).To(BeNil())
			Expect(s.Seekable()).To(BeFalse())
			Expect(s.Duration()).To(Equal(float32(257.0)))
		})
		It("returns a seekable stream if the file is complete in the cache", func() {
			Eventually(func() bool { return ffmpeg.closed }).Should(BeTrue())
			s, err := streamer.NewStream(ctx, "123", "mp3", 64)
			Expect(err).To(BeNil())
			Expect(s.Seekable()).To(BeTrue())
		})
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
		})

		Context("player has format configured", func() {
			BeforeEach(func() {
				t := model.Transcoding{ID: "oga1", TargetFormat: "oga", DefaultBitRate: 96}
				ctx = context.WithValue(ctx, "transcoding", t)
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
		})
		Context("player has maxBitRate configured", func() {
			BeforeEach(func() {
				t := model.Transcoding{ID: "oga1", TargetFormat: "oga", DefaultBitRate: 96}
				p := model.Player{ID: "player1", TranscodingId: t.ID, MaxBitRate: 80}
				ctx = context.WithValue(ctx, "transcoding", t)
				ctx = context.WithValue(ctx, "player", p)
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
				Expect(bitRate).To(Equal(80))
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
		})
	})
})

type fakeFFmpeg struct {
	Data   string
	r      io.Reader
	closed bool
}

func (ff *fakeFFmpeg) Start(ctx context.Context, cmd, path string, maxBitRate int, format string) (f io.ReadCloser, err error) {
	ff.r = strings.NewReader(ff.Data)
	return ff, nil
}

func (ff *fakeFFmpeg) Read(p []byte) (n int, err error) {
	return ff.r.Read(p)
}

func (ff *fakeFFmpeg) Close() error {
	ff.closed = true
	return nil
}
