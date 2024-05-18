package core_test

import (
	"context"
	"io"
	"os"

	"github.com/navidrome/navidrome/conf"
	"github.com/navidrome/navidrome/conf/configtest"
	"github.com/navidrome/navidrome/core"
	"github.com/navidrome/navidrome/log"
	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/tests"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("MediaStreamer", func() {
	var streamer core.MediaStreamer
	var ds model.DataStore
	ffmpeg := tests.NewMockFFmpeg("fake data")
	ctx := log.NewContext(context.TODO())

	BeforeEach(func() {
		DeferCleanup(configtest.SetupConfig())
		conf.Server.CacheFolder, _ = os.MkdirTemp("", "file_caches")
		conf.Server.TranscodingCacheSize = "100MB"
		ds = &tests.MockDataStore{MockedTranscoding: &tests.MockTranscodingRepo{}}
		ds.MediaFile(ctx).(*tests.MockMediaFileRepo).SetData(model.MediaFiles{
			{ID: "123", Path: "tests/fixtures/test.mp3", Suffix: "mp3", BitRate: 128, Duration: 257.0},
		})
		testCache := core.NewTranscodingCache()
		Eventually(func() bool { return testCache.Available(context.TODO()) }).Should(BeTrue())
		streamer = core.NewMediaStreamer(ds, ffmpeg, testCache)
	})
	AfterEach(func() {
		_ = os.RemoveAll(conf.Server.CacheFolder)
	})

	Context("NewStream", func() {
		It("returns a seekable stream if format is 'raw'", func() {
			s, err := streamer.NewStream(ctx, "123", "raw", 0, 0)
			Expect(err).ToNot(HaveOccurred())
			Expect(s.Seekable()).To(BeTrue())
		})
		It("returns a seekable stream if maxBitRate is 0", func() {
			s, err := streamer.NewStream(ctx, "123", "mp3", 0, 0)
			Expect(err).ToNot(HaveOccurred())
			Expect(s.Seekable()).To(BeTrue())
		})
		It("returns a seekable stream if maxBitRate is higher than file bitRate", func() {
			s, err := streamer.NewStream(ctx, "123", "mp3", 320, 0)
			Expect(err).ToNot(HaveOccurred())
			Expect(s.Seekable()).To(BeTrue())
		})
		It("returns a NON seekable stream if transcode is required", func() {
			s, err := streamer.NewStream(ctx, "123", "mp3", 64, 0)
			Expect(err).To(BeNil())
			Expect(s.Seekable()).To(BeFalse())
			Expect(s.Duration()).To(Equal(float32(257.0)))
		})
		It("returns a seekable stream if the file is complete in the cache", func() {
			s, err := streamer.NewStream(ctx, "123", "mp3", 32, 0)
			Expect(err).To(BeNil())
			_, _ = io.ReadAll(s)
			_ = s.Close()
			Eventually(func() bool { return ffmpeg.IsClosed() }, "3s").Should(BeTrue())

			s, err = streamer.NewStream(ctx, "123", "mp3", 32, 0)
			Expect(err).To(BeNil())
			Expect(s.Seekable()).To(BeTrue())
		})
	})
})
