package core

import (
	"context"
	"io"
	"os"
	"strings"
	"sync"

	"github.com/navidrome/navidrome/conf"
	"github.com/navidrome/navidrome/conf/configtest"
	"github.com/navidrome/navidrome/log"
	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/model/request"
	"github.com/navidrome/navidrome/tests"
	"github.com/navidrome/navidrome/utils"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("MediaStreamer", func() {
	var streamer MediaStreamer
	var ds model.DataStore
	ffmpeg := newFakeFFmpeg("fake data")
	ctx := log.NewContext(context.TODO())

	BeforeEach(func() {
		DeferCleanup(configtest.SetupConfig())
		conf.Server.DataFolder, _ = os.MkdirTemp("", "file_caches")
		conf.Server.TranscodingCacheSize = "100MB"
		ds = &tests.MockDataStore{MockedTranscoding: &tests.MockTranscodingRepo{}}
		ds.MediaFile(ctx).(*tests.MockMediaFileRepo).SetData(model.MediaFiles{
			{ID: "123", Path: "tests/fixtures/test.mp3", Suffix: "mp3", BitRate: 128, Duration: 257.0},
		})
		testCache := GetTranscodingCache()
		Eventually(func() bool { return testCache.Ready(context.TODO()) }).Should(BeTrue())
		streamer = NewMediaStreamer(ds, ffmpeg, testCache)
	})
	AfterEach(func() {
		_ = os.RemoveAll(conf.Server.DataFolder)
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
			s, err := streamer.NewStream(ctx, "123", "mp3", 32)
			Expect(err).To(BeNil())
			_, _ = io.ReadAll(s)
			_ = s.Close()
			Eventually(func() bool { return ffmpeg.IsClosed() }, "3s").Should(BeTrue())

			s, err = streamer.NewStream(ctx, "123", "mp3", 32)
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
			It("returns the DefaultDownsamplingFormat if a maxBitrate but not the format", func() {
				conf.Server.DefaultDownsamplingFormat = "opus"
				mf.Suffix = "FLAC"
				mf.BitRate = 960
				format, bitRate := selectTranscodingOptions(ctx, ds, mf, "", 128)
				Expect(format).To(Equal("opus"))
				Expect(bitRate).To(Equal(128))

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
				p := model.Player{ID: "player1", TranscodingId: t.ID, MaxBitRate: 80}
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

func newFakeFFmpeg(data string) *fakeFFmpeg {
	return &fakeFFmpeg{Reader: strings.NewReader(data)}
}

type fakeFFmpeg struct {
	io.Reader
	lock   sync.Mutex
	closed utils.AtomicBool
}

func (ff *fakeFFmpeg) Start(ctx context.Context, cmd, path string, maxBitRate int) (f io.ReadCloser, err error) {
	return ff, nil
}

func (ff *fakeFFmpeg) Read(p []byte) (n int, err error) {
	ff.lock.Lock()
	defer ff.lock.Unlock()
	return ff.Reader.Read(p)
}

func (ff *fakeFFmpeg) Close() error {
	ff.closed.Set(true)
	return nil
}

func (ff *fakeFFmpeg) IsClosed() bool {
	return ff.closed.Get()
}
