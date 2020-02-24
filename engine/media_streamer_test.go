package engine

import (
	"context"
	"io"
	"io/ioutil"
	"os"
	"strings"

	"github.com/deluan/navidrome/conf"
	"github.com/deluan/navidrome/log"
	"github.com/deluan/navidrome/model"
	"github.com/deluan/navidrome/persistence"
	"github.com/djherbis/fscache"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("MediaStreamer", func() {
	var streamer MediaStreamer
	var ds model.DataStore
	var cache fscache.Cache
	var tempDir string
	ffmpeg := &fakeFFmpeg{}
	ctx := log.NewContext(nil)

	BeforeSuite(func() {
		tempDir, _ = ioutil.TempDir("", "stream_tests")
		fs, _ := fscache.NewFs(tempDir, 0755)
		cache, _ = fscache.NewCache(fs, nil)
	})

	BeforeEach(func() {
		conf.Server.EnableDownsampling = true
		ds = &persistence.MockDataStore{}
		ds.MediaFile(ctx).(*persistence.MockMediaFile).SetData(`[{"id": "123", "path": "tests/fixtures/test.mp3", "bitRate": 128, "duration": 257.0}]`, 1)
		streamer = NewMediaStreamer(ds, ffmpeg, cache)
	})

	AfterSuite(func() {
		os.RemoveAll(tempDir)
	})

	Context("NewFileSystem", func() {
		It("returns a seekable stream if format is 'raw'", func() {
			s, err := streamer.NewStream(ctx, "123", 0, "raw")
			Expect(err).ToNot(HaveOccurred())
			Expect(s.Seekable()).To(BeTrue())
		})
		It("returns a seekable stream if maxBitRate is 0", func() {
			s, err := streamer.NewStream(ctx, "123", 0, "mp3")
			Expect(err).ToNot(HaveOccurred())
			Expect(s.Seekable()).To(BeTrue())
		})
		It("returns a seekable stream if maxBitRate is higher than file bitRate", func() {
			s, err := streamer.NewStream(ctx, "123", 320, "mp3")
			Expect(err).ToNot(HaveOccurred())
			Expect(s.Seekable()).To(BeTrue())
		})
		It("returns a NON seekable stream if transcode is required", func() {
			s, err := streamer.NewStream(ctx, "123", 64, "mp3")
			Expect(err).To(BeNil())
			Expect(s.Seekable()).To(BeFalse())
			Expect(s.Duration()).To(Equal(float32(257.0)))
		})
		It("returns a seekable stream if the file is complete in the cache", func() {
			Eventually(func() bool { return ffmpeg.closed }).Should(BeTrue())
			s, err := streamer.NewStream(ctx, "123", 64, "mp3")
			Expect(err).To(BeNil())
			Expect(s.Seekable()).To(BeTrue())
		})
	})
})

type fakeFFmpeg struct {
	r      io.Reader
	closed bool
}

func (ff *fakeFFmpeg) StartTranscoding(ctx context.Context, path string, maxBitRate int, format string) (f io.ReadCloser, err error) {
	ff.r = strings.NewReader("fake data")
	return ff, nil
}

func (ff *fakeFFmpeg) Read(p []byte) (n int, err error) {
	return ff.r.Read(p)
}

func (ff *fakeFFmpeg) Close() error {
	ff.closed = true
	return nil
}
