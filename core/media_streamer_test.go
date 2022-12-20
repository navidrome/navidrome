package core_test

import (
	"context"
	"io"
	"os"
	"strings"
	"sync"

	"github.com/navidrome/navidrome/conf"
	"github.com/navidrome/navidrome/conf/configtest"
	. "github.com/navidrome/navidrome/core"
	"github.com/navidrome/navidrome/log"
	"github.com/navidrome/navidrome/model"
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
